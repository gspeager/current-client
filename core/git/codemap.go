package git

import (
	"context"
	"strconv"
	"strings"
	"time"
)

// FileLines is a file's size at a commit. A binary file has no lines.
type FileLines struct {
	Path   string
	Lines  int
	Binary bool
}

// Past this size a file counts as binary, and isn't read.
const largestCounted = 10 << 20

// LineCounts is every file at rev with its number of lines, from one diff against
// nothing. Submodules aren't files.
func LineCounts(ctx context.Context, repoPath, rev string) ([]FileLines, error) {
	empty, err := runResult(ctx, repoPath, "hash-object", "-t", "tree", "--stdin")
	if err != nil {
		return nil, err
	}
	result, err := runResult(ctx, repoPath,
		"-c", "core.bigFileThreshold="+strconv.Itoa(largestCounted),
		"diff", "--numstat", "-z", "--no-renames", "--no-ext-diff", "--no-textconv", "--no-relative", "--ignore-submodules",
		strings.TrimSpace(empty.Stdout), rev)
	if err != nil {
		return nil, err
	}
	entries, err := parseNumstat(result.Stdout)
	if err != nil {
		return nil, err
	}
	files := make([]FileLines, len(entries))
	for i, entry := range entries {
		files[i] = FileLines{Path: entry.Path, Lines: entry.Added, Binary: entry.Binary}
	}
	return files, nil
}

// FileActivity is how often a file changed under its name, in each of the periods
// asked for, and when it last did.
type FileActivity struct {
	Path    string
	Commits []int
	Last    time.Time
}

// FileCommits counts the commits that changed each file, newest first, in the last
// so many days of each period; 0 is all of them. It reads at most mostCommits, and
// more says the history goes further back. Merges aren't counted, so a merged
// branch's commits aren't counted twice.
func FileCommits(ctx context.Context, repoPath, rev string, periods []int, mostCommits int) (files []FileActivity, more bool, err error) {
	result, err := runResult(ctx, repoPath, "log", "--no-merges", "--no-renames", "--no-ext-diff", "--no-show-signature",
		"--max-count="+strconv.Itoa(mostCommits+1), "--format=%x01%ct", "--name-only", "-z", rev, "--")
	if err != nil {
		return nil, false, err
	}
	now := time.Now()
	index := map[string]int{}
	var at time.Time
	var counted []bool
	commits, afterHeader := 0, false
	for token := range strings.SplitSeq(result.Stdout, "\x00") {
		if seconds, header := strings.CutPrefix(token, "\x01"); header {
			commits++
			if commits > mostCommits {
				return files, true, nil
			}
			unix, err := strconv.ParseInt(seconds, 10, 64)
			if err != nil {
				return nil, false, err
			}
			at, afterHeader = time.Unix(unix, 0), true
			counted = make([]bool, len(periods))
			for i, days := range periods {
				counted[i] = days == 0 || now.Sub(at) < time.Duration(days)*24*time.Hour
			}
			continue
		}
		path := token
		if afterHeader {
			path, afterHeader = strings.TrimPrefix(token, "\n"), false
		}
		if path == "" {
			continue
		}
		i, known := index[path]
		if !known {
			i = len(files)
			index[path] = i
			files = append(files, FileActivity{Path: path, Commits: make([]int, len(periods))})
		}
		if at.After(files[i].Last) {
			files[i].Last = at
		}
		for period, in := range counted {
			if in {
				files[i].Commits[period]++
			}
		}
	}
	return files, false, nil
}
