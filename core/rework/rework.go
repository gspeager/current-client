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

// Range reports the rework in the first-parent commits after base up to head.
func Range(ctx context.Context, repoPath, base, head string) (Report, error) {
	span := base + ".." + head
	shas, err := gitexec.OutputIn(ctx, repoPath, nil, "rev-list", "--first-parent", "--no-merges", span)
	if err != nil {
		return Report{}, err
	}
	inRange := map[string]bool{}
	for _, sha := range strings.Fields(shas) {
		inRange[sha] = true
	}
	log, err := gitexec.OutputIn(ctx, repoPath, nil, "-c", "core.quotePath=false", "log", "--first-parent",
		"--no-merges", "--reverse", "--max-count="+strconv.Itoa(MostCommits), "-p", "-U0", "-M", "--no-color",
		"--no-ext-diff", "--format=%x00%H%x1f%s", span)
	if err != nil {
		return Report{}, err
	}
	report := Report{Capped: len(inRange) > MostCommits}
	files := map[string]*File{}
	// Every name a file has had since a commit in the range wrote to it.
	written := map[string]bool{}
	// Old name to new, in the order the range renamed them.
	var renames [][2]string
	for _, c := range parseLog(log) {
		commit := Commit{SHA: c.sha, Subject: c.subject}
		moved := map[string]int{}
		for _, f := range c.files {
			for _, line := range f.added {
				moved[line]++
			}
		}
		for _, f := range c.files {
			path := f.newPath
			if path == "" {
				path = f.oldPath
			}
			if f.oldPath != "" && f.newPath != "" && f.oldPath != f.newPath {
				renames = append(renames, [2]string{f.oldPath, f.newPath})
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
				for _, line := range lines {
					text := strings.TrimSpace(line.Content)
					if text == "" {
						continue
					}
					if moved[text] > 0 {
						moved[text]--
						continue
					}
					if inRange[line.SHA] {
						rewritten++
					}
				}
			}
			if len(f.added) > 0 {
				written[path] = true
			}
			file := files[path]
			if file == nil {
				file = &File{Path: path}
				files[path] = file
			}
			file.Written += len(f.added)
			file.Rewritten += rewritten
			file.Commits++
			commit.Written += len(f.added)
			commit.Rewritten += rewritten
		}
		report.Written += commit.Written
		report.Rewritten += commit.Rewritten
		report.Commits = append(report.Commits, commit)
	}
	report.Files = reworked(files, renames)
	return report, nil
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

// reworked follows each file's later renames to its last name, merging what
// was counted under each name.
func reworked(files map[string]*File, renames [][2]string) []File {
	last := func(path string) string {
		for _, r := range renames {
			if r[0] == path {
				path = r[1]
			}
		}
		return path
	}
	merged := map[string]*File{}
	for path, f := range files {
		name := last(path)
		m := merged[name]
		if m == nil {
			m = &File{Path: name}
			merged[name] = m
		}
		m.Written += f.Written
		m.Rewritten += f.Rewritten
		m.Commits += f.Commits
	}
	var out []File
	for _, f := range merged {
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
