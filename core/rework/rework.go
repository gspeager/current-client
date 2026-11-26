// Package rework finds what a range of commits wrote and then changed again: the
// lines a commit deletes that an earlier commit in the same range wrote. A line
// a commit deletes and adds back unchanged, apart from indentation, is a move,
// not rework. Blank lines count for neither.
package rework

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
)

// MostCommits is how many of a range's newest commits are read.
const MostCommits = 300

type Commit struct {
	SHA     string `json:"sha"`
	Subject string `json:"subject"`
	Written int    `json:"written"`
	// Rewritten is the lines this commit changed or removed that an earlier
	// commit in the range wrote.
	Rewritten int `json:"rewritten"`
}

type File struct {
	// Path is the file's name at the range's end, or its last name when it's gone.
	Path      string `json:"path"`
	Written   int    `json:"written"`
	Rewritten int    `json:"rewritten"`
	Commits   int    `json:"commits"`
}

type Report struct {
	Written   int `json:"written"`
	Rewritten int `json:"rewritten"`
	// Commits are oldest first.
	Commits []Commit `json:"commits"`
	// Files are the ones with rework, the most first.
	Files []File `json:"files"`
	// Capped means only the newest MostCommits were read.
	Capped bool `json:"capped"`
}

type fileDiff struct {
	oldPath, newPath string
	deleted          [][2]int
	added            []string
}

type commitDiff struct {
	sha, subject string
	files        []fileDiff
}

// readLimit is MostCommits, lowered by tests.
var readLimit = MostCommits

// Range reports the rework in the first-parent commits after base up to head.
func Range(ctx context.Context, repoPath, base, head string) (Report, error) {
	span := base + ".." + head
	count, err := gitexec.OutputIn(ctx, repoPath, nil, "rev-list", "--count", "--first-parent", "--no-merges", span)
	if err != nil {
		return Report{}, err
	}
	total, _ := strconv.Atoi(strings.TrimSpace(count))
	log, err := gitexec.OutputIn(ctx, repoPath, nil, "-c", "core.quotePath=false", "log", "--first-parent",
		"--no-merges", "--reverse", "--max-count="+strconv.Itoa(readLimit), "-p", "-U0", "-M", "--no-color",
		"--no-ext-diff", "--format=%x00%H%x1f%s", span)
	if err != nil {
		return Report{}, err
	}
	commits := parseLog(log)
	// Only commits read count as writers, so what's rewritten is never more than what's written.
	read := map[string]bool{}
	for _, c := range commits {
		read[c.sha] = true
	}
	report := Report{Capped: total > readLimit}
	// By each file's name now; a rename moves what was counted to the new name.
	files := map[string]*File{}
	// The names a commit in the range has written to.
	written := map[string]bool{}
	for _, c := range commits {
		commit := Commit{SHA: c.sha, Subject: c.subject}
		added := map[string]int{}
		for _, f := range c.files {
			for _, line := range f.added {
				added[line]++
			}
		}
		for _, f := range c.files {
			path := f.newPath
			if path == "" {
				path = f.oldPath
			}
			if f.oldPath != "" && f.newPath != "" && f.oldPath != f.newPath {
				if files[f.oldPath] != nil {
					files[f.newPath] = merge(files[f.newPath], files[f.oldPath], f.newPath)
					delete(files, f.oldPath)
				}
				if written[f.oldPath] {
					written[f.newPath] = true
				}
			}
			rewritten := 0
			if len(f.deleted) > 0 && written[f.oldPath] {
				lines, err := deletedLines(ctx, repoPath, f.oldPath, c.sha+"^", f.deleted)
				if err != nil {
					return Report{}, err
				}
				rewritten = countRewritten(lines, added, read)
			}
			if len(f.added) > 0 {
				written[path] = true
			}
			file := merge(files[path], &File{Written: len(f.added), Rewritten: rewritten, Commits: 1}, path)
			files[path] = file
			commit.Written += len(f.added)
			commit.Rewritten += rewritten
		}
		report.Written += commit.Written
		report.Rewritten += commit.Rewritten
		report.Commits = append(report.Commits, commit)
	}
	report.Files = reworked(files)
	return report, nil
}

func merge(into, from *File, path string) *File {
	if into == nil {
		into = &File{Path: path}
	}
	into.Written += from.Written
	into.Rewritten += from.Rewritten
	into.Commits += from.Commits
	return into
}

// distinctive is how many letters and digits a line needs to count as moved on
// its own, as Git's --color-moved has it. A shorter line, such as a closing
// brace, is moved only beside a moved line that has them.
const distinctive = 20

// countRewritten counts the deleted lines a read commit wrote that weren't
// moved: added back in the same commit.
func countRewritten(lines []git.BlameLine, added map[string]int, read map[string]bool) int {
	texts := make([]string, len(lines))
	moved := make([]bool, len(lines))
	for i, line := range lines {
		texts[i] = strings.TrimSpace(line.Content)
		moved[i] = added[texts[i]] > 0 && letters(texts[i]) >= distinctive
	}
	// Short lines take a neighbouring moved line's part, through a run of lines.
	for changed := true; changed; {
		changed = false
		for i := range lines {
			if moved[i] || texts[i] == "" || added[texts[i]] == 0 {
				continue
			}
			before := i > 0 && moved[i-1] && lines[i-1].LineNo == lines[i].LineNo-1
			after := i+1 < len(lines) && moved[i+1] && lines[i+1].LineNo == lines[i].LineNo+1
			if before || after {
				moved[i], changed = true, true
			}
		}
	}
	rewritten := 0
	for i, line := range lines {
		switch {
		case texts[i] == "":
		case moved[i] && added[texts[i]] > 0:
			added[texts[i]]--
		case read[line.SHA]:
			rewritten++
		}
	}
	return rewritten
}

func letters(text string) int {
	n := 0
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			n++
		}
	}
	return n
}

// mostRanges keeps the command line short; past it the whole file is blamed.
const mostRanges = 100

func deletedLines(ctx context.Context, repoPath, path, rev string, spans [][2]int) ([]git.BlameLine, error) {
	if len(spans) <= mostRanges {
		return git.BlameRanges(ctx, repoPath, path, rev, spans)
	}
	all, err := git.BlameAt(ctx, repoPath, path, rev)
	if err != nil {
		return nil, err
	}
	var lines []git.BlameLine
	for _, line := range all {
		for _, span := range spans {
			if line.LineNo >= span[0] && line.LineNo < span[0]+span[1] {
				lines = append(lines, line)
				break
			}
		}
	}
	return lines, nil
}

func reworked(files map[string]*File) []File {
	var out []File
	for _, f := range files {
		if f.Rewritten > 0 {
			out = append(out, *f)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Rewritten != out[j].Rewritten {
			return out[i].Rewritten > out[j].Rewritten
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func parseLog(log string) []commitDiff {
	var commits []commitDiff
	for _, block := range strings.Split(log, "\x00")[1:] {
		header, body, _ := strings.Cut(block, "\n")
		sha, subject, _ := strings.Cut(header, "\x1f")
		c := commitDiff{sha: sha, subject: subject}
		var f *fileDiff
		inHunks := false
		for _, line := range strings.Split(body, "\n") {
			switch {
			case strings.HasPrefix(line, "diff --git "):
				c.files = append(c.files, fileDiff{})
				f = &c.files[len(c.files)-1]
				inHunks = false
			case f == nil:
			case !inHunks && strings.HasPrefix(line, "--- "):
				f.oldPath = side(line[4:], "a/")
			case !inHunks && strings.HasPrefix(line, "+++ "):
				f.newPath = side(line[4:], "b/")
			case strings.HasPrefix(line, "rename from "):
				f.oldPath = unquote(strings.TrimPrefix(line, "rename from "))
			case strings.HasPrefix(line, "rename to "):
				f.newPath = unquote(strings.TrimPrefix(line, "rename to "))
			case strings.HasPrefix(line, "@@ "):
				inHunks = true
				if start, count, ok := deletedSpan(line); ok && count > 0 {
					f.deleted = append(f.deleted, [2]int{start, count})
				}
			case strings.HasPrefix(line, "+"):
				if text := strings.TrimSpace(line[1:]); text != "" {
					f.added = append(f.added, text)
				}
			}
		}
		commits = append(commits, c)
	}
	return commits
}

// side reads one side of a "---" or "+++" line.
func side(path, prefix string) string {
	// Git ends a name that has a space with a tab, for patch.
	path = strings.TrimSuffix(path, "\t")
	if path == "/dev/null" {
		return ""
	}
	return strings.TrimPrefix(unquote(path), prefix)
}

func unquote(path string) string {
	if strings.HasPrefix(path, `"`) {
		if s, err := strconv.Unquote(path); err == nil {
			return s
		}
	}
	return path
}

// deletedSpan reads the old side of a hunk header, "@@ -start,count +...".
func deletedSpan(header string) (start, count int, ok bool) {
	fields := strings.Fields(header)
	if len(fields) < 2 || !strings.HasPrefix(fields[1], "-") {
		return 0, 0, false
	}
	from, n, hasCount := strings.Cut(fields[1][1:], ",")
	start, err := strconv.Atoi(from)
	if err != nil {
		return 0, 0, false
	}
	count = 1
	if hasCount {
		if count, err = strconv.Atoi(n); err != nil {
			return 0, 0, false
		}
	}
	return start, count, true
}
