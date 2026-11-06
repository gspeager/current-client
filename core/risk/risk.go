package risk

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/gspeager/current-client/core/deps"
	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
)

const (
	RuleSecret         = "secret"
	RuleConflictMarker = "conflict-marker"
	RuleLargeFile      = "large-file"
	RuleLockfile       = "lockfile"
	RuleGenerated      = "generated"
)

const (
	largeFileBytes = 10 << 20
	generatedFiles = 20
	mostFindings   = 100
	// Minified bundles aren't where a key is pasted, and regexes over them are slow.
	longestLineScanned = 8 << 10
)

// Finding.Line is 0 when it's about the whole file.
type Finding struct {
	Rule    string
	Path    string
	Line    int
	Message string
}

// Check reports on what's staged, leaving out the rules in off.
func Check(ctx context.Context, repoPath string, off []string) ([]Finding, error) {
	on := func(rule string) bool { return !slices.Contains(off, rule) }
	staged, err := stagedPaths(ctx, repoPath)
	if err != nil || len(staged) == 0 {
		return nil, err
	}
	sizes, err := indexSizes(ctx, repoPath, append(staged, siblings(staged)...))
	if err != nil {
		return nil, err
	}

	var findings []Finding
	var tooLarge []string
	for _, file := range staged {
		if sizes[file] >= largeFileBytes {
			tooLarge = append(tooLarge, file)
			if on(RuleLargeFile) {
				findings = append(findings, Finding{RuleLargeFile, file, 0, fmt.Sprintf("Large file: %d MB.", sizes[file]>>20)})
			}
		}
	}
	if on(RuleSecret) || on(RuleConflictMarker) {
		lines, err := addedLines(ctx, repoPath, tooLarge)
		if err != nil {
			return nil, err
		}
		findings = append(findings, scanLines(lines, on(RuleSecret), on(RuleConflictMarker))...)
	}
	if on(RuleLockfile) {
		lockfile, err := lockfileFindings(ctx, repoPath, staged, sizes)
		if err != nil {
			return nil, err
		}
		findings = append(findings, lockfile...)
	}
	if on(RuleGenerated) {
		findings = append(findings, generatedFindings(staged)...)
	}
	if len(findings) > mostFindings {
		findings = findings[:mostFindings]
	}
	return findings, nil
}

func stagedPaths(ctx context.Context, repoPath string) ([]string, error) {
	result, err := gitexec.NewExecutor("").RunChecked(ctx, gitexec.Command{
		Dir:  repoPath,
		Args: []string{"diff", "--cached", "--name-only", "-z", "--no-renames", "--diff-filter=d"},
	})
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, file := range strings.Split(result.Stdout, "\x00") {
		if file != "" {
			paths = append(paths, file)
		}
	}
	return paths, nil
}

// siblings names the other half of each staged manifest or lockfile.
func siblings(staged []string) []string {
	var names []string
	for _, file := range staged {
		dir, name := path.Split(file)
		for _, lock := range deps.LocksOf(name) {
			names = append(names, dir+lock)
		}
		if manifest := deps.ManifestOf(name); manifest != "" {
			names = append(names, dir+manifest)
		}
	}
	return names
}

// indexSizes returns the size of each path that's in the index.
func indexSizes(ctx context.Context, repoPath string, paths []string) (map[string]int64, error) {
	var input strings.Builder
	var asked []string
	for _, file := range paths {
		if !strings.Contains(file, "\n") {
			input.WriteString(":" + file + "\n")
			asked = append(asked, file)
		}
	}
	result, err := gitexec.NewExecutor("").RunChecked(ctx, gitexec.Command{
		Dir:   repoPath,
		Args:  []string{"cat-file", "--batch-check=%(objecttype) %(objectsize)"},
		Stdin: strings.NewReader(input.String()),
	})
	if err != nil {
		return nil, err
	}
	sizes := map[string]int64{}
	for i, line := range strings.Split(strings.TrimRight(result.Stdout, "\n"), "\n") {
		kind, size, ok := strings.Cut(line, " ")
		if !ok || kind != "blob" || i >= len(asked) {
			continue
		}
		if bytes, err := strconv.ParseInt(size, 10, 64); err == nil {
			sizes[asked[i]] = bytes
		}
	}
	return sizes, nil
}

type addedLine struct {
	path string
	line int
	text string
}

var hunkHeader = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@`)

// addedLines reads the staged diff, leaving out skip, which is too large to hold.
func addedLines(ctx context.Context, repoPath string, skip []string) ([]addedLine, error) {
	args := []string{
		"-c", "core.quotepath=off", "diff", "--cached", "-U0", "--no-color", "--no-ext-diff", "--no-textconv",
		"--no-renames", "--diff-filter=d", "--src-prefix=a/", "--dst-prefix=b/", "--", ".",
	}
	for _, file := range skip {
		args = append(args, ":(exclude,literal)"+file)
	}
	result, err := gitexec.NewExecutor("").RunChecked(ctx, gitexec.Command{Dir: repoPath, Args: args})
	if err != nil {
		return nil, err
	}
	var lines []addedLine
	// Counting a hunk's new lines keeps an added "++ x", which the diff shows as
	// "+++ x", from reading as a file header.
	current, number, left := "", 0, 0
	for _, line := range strings.Split(result.Stdout, "\n") {
		switch {
		case left > 0 && strings.HasPrefix(line, "+"):
			lines = append(lines, addedLine{current, number, strings.TrimSuffix(line[1:], "\r")})
			number++
			left--
		case strings.HasPrefix(line, "+++ "):
			current = diffPath(strings.TrimPrefix(line, "+++ "))
		case strings.HasPrefix(line, "@@ "):
			if match := hunkHeader.FindStringSubmatch(line); match != nil {
				number, _ = strconv.Atoi(match[1])
				left = 1
				if match[2] != "" {
					left, _ = strconv.Atoi(match[2])
				}
			}
		}
	}
	return lines, nil
}

// Git quotes a name with special characters, in C style, which Go can read.
func diffPath(field string) string {
	// Git ends a name holding a space with a tab.
	field = strings.TrimSuffix(field, "\t")
	if strings.HasPrefix(field, `"`) {
		if unquoted, err := strconv.Unquote(field); err == nil {
			field = unquoted
		}
	}
	return strings.TrimPrefix(field, "b/")
}

func scanLines(lines []addedLine, secrets, markers bool) []Finding {
	rules := loadSecretRules()
	var findings []Finding
	for _, line := range lines {
		if markers && (strings.HasPrefix(line.text, "<<<<<<< ") || strings.HasPrefix(line.text, ">>>>>>> ")) {
			findings = append(findings, Finding{RuleConflictMarker, line.path, line.line, "Conflict marker left in."})
		}
		if !secrets || len(line.text) > longestLineScanned || rules.pathIgnored(line.path) {
			continue
		}
		for _, match := range rules.scanLine(line.path, line.text) {
			findings = append(findings, Finding{RuleSecret, line.path, line.line, fmt.Sprintf("Looks like %s: %s", strings.ReplaceAll(match.rule, "-", " "), match.masked)})
		}
	}
	return findings
}

func lockfileFindings(ctx context.Context, repoPath string, staged []string, sizes map[string]int64) ([]Finding, error) {
	var findings []Finding
	for _, file := range staged {
		dir, name := path.Split(file)
		if manifest := deps.ManifestOf(name); manifest != "" {
			if _, tracked := sizes[dir+manifest]; tracked && !slices.Contains(staged, dir+manifest) {
				findings = append(findings, Finding{RuleLockfile, file, 0, fmt.Sprintf("%s changed without %s.", name, manifest)})
			}
			continue
		}
		var missing string
		for _, lock := range deps.LocksOf(name) {
			if slices.Contains(staged, dir+lock) {
				missing = ""
				break
			}
			if _, tracked := sizes[dir+lock]; tracked && missing == "" {
				missing = lock
			}
		}
		if missing == "" {
			continue
		}
		changed, err := packagesChanged(ctx, repoPath, file)
		if err != nil {
			return nil, err
		}
		if changed {
			findings = append(findings, Finding{RuleLockfile, file, 0, fmt.Sprintf("Packages changed in %s without %s.", name, missing)})
		}
	}
	return findings, nil
}

// A manifest also holds scripts and metadata, which don't touch the lockfile.
// One that doesn't parse can't be ruled out, so it counts as changed.
func packagesChanged(ctx context.Context, repoPath, file string) (bool, error) {
	before, after := git.RevPath{Rev: "HEAD", Path: file}, git.RevPath{Rev: "", Path: file}
	files, err := git.FilesAt(ctx, repoPath, []git.RevPath{before, after})
	if err != nil {
		return false, err
	}
	changes, err := deps.Summarize(file, files[before], files[after])
	return err != nil || len(changes) > 0, nil
}

var generatedPath = regexp.MustCompile(`(^|/)(dist|build|vendor|node_modules|generated|__generated__|target)/|\.min\.(js|css)$|\.pb\.go$|[._]generated\.[a-z]+$`)

func generatedFindings(staged []string) []Finding {
	var first string
	count := 0
	for _, file := range staged {
		if generatedPath.MatchString(file) {
			if count == 0 {
				first = file
			}
			count++
		}
	}
	if count < generatedFiles {
		return nil
	}
	return []Finding{{RuleGenerated, first, 0, fmt.Sprintf("%d generated files staged.", count)}}
}
