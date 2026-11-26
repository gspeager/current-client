package risk

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/gspeager/current-client/core/diff"
)

type addedLine struct {
	path string
	line int
	text string
}

type changedLines struct {
	added   []addedLine
	removed []addedLine
}

// newlyAdded leaves out a line that was removed elsewhere in the change, so
// moving code doesn't repeat what it already held.
func (l changedLines) newlyAdded() []addedLine {
	removed := map[string]int{}
	for _, line := range l.removed {
		removed[strings.TrimSpace(line.text)]++
	}
	var added []addedLine
	for _, line := range l.added {
		text := strings.TrimSpace(line.text)
		if removed[text] > 0 {
			removed[text]--
			continue
		}
		added = append(added, line)
	}
	return added
}

// Git writes a file over the limit as binary, so one is never held in memory. The
// rest pin down what a setting would change.
func diffArgs(command string, args ...string) []string {
	return append([]string{
		"-c", "core.quotepath=off", "-c", fmt.Sprintf("core.bigFileThreshold=%d", largeFileBytes-1), command,
		"-U0", "-M", "--no-color", "--no-ext-diff", "--no-textconv", "--no-relative", "--submodule=short",
		"--inter-hunk-context=0", "--src-prefix=a/", "--dst-prefix=b/",
	}, args...)
}

func (c change) lines(ctx context.Context) (changedLines, error) {
	output, err := c.run(ctx, "", diffArgs("diff", c.diff...)...)
	if err != nil {
		return changedLines{}, err
	}
	var lines changedLines
	parseLines(output, func(line addedLine, added bool) {
		if added {
			lines.added = append(lines.added, line)
		} else {
			lines.removed = append(lines.removed, line)
		}
	}, nil)
	return lines, nil
}

// parseLines calls each with every line of a -U0 diff, under its file's new name
// unless it's deleted. other is given the lines between diffs, which git log
// writes.
func parseLines(output string, each func(line addedLine, added bool), other func(line string)) {
	count := func(text string) int {
		if text == "" {
			return 1
		}
		n, _ := strconv.Atoi(text)
		return n
	}
	// Counting a hunk's lines keeps an added "++ x", which the diff shows as
	// "+++ x", or a removed "-- x", from reading as a file header.
	oldPath, newPath := "", ""
	oldLine, oldLeft, newLine, newLeft := 0, 0, 0, 0
	for line := range strings.SplitSeq(output, "\n") {
		switch {
		case oldLeft > 0 && strings.HasPrefix(line, "-"):
			file := newPath
			if file == "" {
				file = oldPath
			}
			each(addedLine{file, oldLine, strings.TrimSuffix(line[1:], "\r")}, false)
			oldLine++
			oldLeft--
		case newLeft > 0 && strings.HasPrefix(line, "+"):
			each(addedLine{newPath, newLine, strings.TrimSuffix(line[1:], "\r")}, true)
			newLine++
			newLeft--
		case strings.HasPrefix(line, "--- "):
			oldPath = diff.HeaderPath(strings.TrimPrefix(line, "--- "))
		case strings.HasPrefix(line, "+++ "):
			newPath = diff.HeaderPath(strings.TrimPrefix(line, "+++ "))
		case strings.HasPrefix(line, "@@ "):
			if match := diff.HunkHeader.FindStringSubmatch(line); match != nil {
				oldLine, _ = strconv.Atoi(match[1])
				oldLeft = count(match[2])
				newLine, _ = strconv.Atoi(match[3])
				newLeft = count(match[4])
			}
		case other != nil:
			other(line)
		}
	}
}

func scanLines(lines []addedLine, secrets, markers bool) (findings []Finding, values map[string]bool) {
	values = map[string]bool{}
	var matches [][]secretMatch
	if secrets {
		matches = secretsInAll(lines)
	}
	for i, line := range lines {
		if markers && (strings.HasPrefix(line.text, "<<<<<<< ") || strings.HasPrefix(line.text, ">>>>>>> ")) {
			findings = append(findings, Finding{Rule: RuleConflictMarker, Path: line.path, Line: line.line, Message: "Conflict marker left in."})
		}
		if !secrets {
			continue
		}
		for _, match := range matches[i] {
			values[match.secret] = true
			findings = append(findings, Finding{Rule: RuleSecret, Path: line.path, Line: line.line, Message: match.message()})
		}
	}
	return findings, values
}

// Hundreds of patterns a line is most of a large change's time, so the lines are
// shared among the processors.
func secretsInAll(lines []addedLine) [][]secretMatch {
	rules := loadSecretRules()
	matches := make([][]secretMatch, len(lines))
	workers := runtime.GOMAXPROCS(0)
	var wg sync.WaitGroup
	for worker := range workers {
		wg.Go(func() {
			for i := worker; i < len(lines); i += workers {
				matches[i] = rules.secretsIn(lines[i])
			}
		})
	}
	wg.Wait()
	return matches
}

func (r secretRules) secretsIn(line addedLine) []secretMatch {
	if len(line.text) > longestLineScanned || r.pathIgnored(line.path) {
		return nil
	}
	return r.scanLine(line.path, line.text)
}

func (m secretMatch) message() string {
	return fmt.Sprintf("Looks like %s: %s", strings.ReplaceAll(m.rule, "-", " "), m.masked)
}

// removedSecrets finds secrets a commit of the range added that its total change
// no longer has. tooMany means there are too many commits to look.
func removedSecrets(ctx context.Context, c change, head string, left map[string]bool) (findings []Finding, tooMany bool, err error) {
	commits := c.from + ".." + head
	count, err := c.run(ctx, "", "rev-list", "--count", commits)
	if err != nil {
		return nil, false, err
	}
	if n, _ := strconv.Atoi(strings.TrimSpace(count)); n < 2 && c.to == head || n > mostCommits {
		return nil, n > mostCommits, nil
	}
	output, err := c.run(ctx, "", diffArgs("log", "-p", "--format=%x01%h", commits)...)
	if err != nil {
		return nil, false, err
	}
	var lines []addedLine
	var commitOf []string
	commit := ""
	// What each commit removed, to tell a secret that was there before.
	removed := map[string]*strings.Builder{}
	parseLines(output, func(line addedLine, added bool) {
		if added {
			lines = append(lines, line)
			commitOf = append(commitOf, commit)
			return
		}
		if removed[commit] == nil {
			removed[commit] = &strings.Builder{}
		}
		removed[commit].WriteString(line.text + "\n")
	}, func(line string) {
		if sha, ok := strings.CutPrefix(line, "\x01"); ok {
			commit = sha
		}
	})
	seen := map[string]bool{}
	for i, matches := range secretsInAll(lines) {
		for _, match := range matches {
			key := lines[i].path + "\x00" + match.secret
			if left[match.secret] || seen[key] {
				continue
			}
			if text := removed[commitOf[i]]; text != nil && strings.Contains(text.String(), match.secret) {
				continue
			}
			seen[key] = true
			findings = append(findings, Finding{
				Rule:    RuleSecret,
				Path:    lines[i].path,
				Message: fmt.Sprintf("%s, added in %s and removed since.", match.message(), commitOf[i]),
			})
		}
	}
	return findings, false, nil
}
