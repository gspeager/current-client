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

type hunk struct {
	start, deleted, added int
}

type deletedLine struct {
	no   int
	text string
	// written is whether a commit that was read wrote it.
	written bool
}

type fileDiff struct {
	oldPath, newPath string
	hunks            []hunk
	deleted          []deletedLine
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
	commits, capped, err := readCommits(ctx, repoPath, base, head)
	if err != nil {
		return Report{}, err
	}
	// One commit has nothing earlier in the range to rewrite.
	if len(commits) > 1 {
		span := base + ".." + head
		if capped {
			span = commits[0].sha + "^.." + head
		}
		// With merges, for the lines they bring, and -w as blame -w has it: a
		// line that's only re-indented is still its first writer's.
		history, err := readLog(ctx, repoPath, "-m", "-w", span)
		if err != nil {
			return Report{}, err
		}
		markWritten(commits, history)
	}
	return tally(commits, capped), nil
}

func readLog(ctx context.Context, repoPath string, args ...string) ([]commitDiff, error) {
	log, err := gitexec.OutputIn(ctx, repoPath, nil, append([]string{"-c", "core.quotePath=false", "log",
		"--first-parent", "--reverse", "-p", "-U0", "-M", "--no-color", "--no-ext-diff", "--format=%x00%H%x1f%s"}, args...)...)
	return parseLog(log), err
}

// readCommits reads the newest readLimit commits, oldest first.
func readCommits(ctx context.Context, repoPath, base, head string) (commits []commitDiff, capped bool, err error) {
	// One more than is kept says whether the range goes further back.
	commits, err = readLog(ctx, repoPath, "--no-merges", "--max-count="+strconv.Itoa(readLimit+1), base+".."+head)
	if capped = len(commits) > readLimit; capped {
		commits = commits[1:]
	}
	return commits, capped, err
}

// markWritten sets which deleted lines a read commit wrote, by following every
// file's lines through history: the first-parent commits that commits are among.
func markWritten(commits, history []commitDiff) {
	read := map[string]*commitDiff{}
	for i := range commits {
		read[commits[i].sha] = &commits[i]
	}
	// For each file, whether a read commit wrote each line; it didn't write those past the end.
	written := map[string][]bool{}
	for _, h := range history {
		c := read[h.sha]
		if c != nil {
			for _, f := range c.files {
				lines := written[f.oldPath]
				for i, d := range f.deleted {
					f.deleted[i].written = d.no <= len(lines) && lines[d.no-1]
				}
			}
		}
		for _, f := range h.files {
			follow(written, f, c != nil)
		}
	}
}

// follow moves written on by one file's hunks, whose added lines a read commit wrote or didn't.
func follow(written map[string][]bool, f fileDiff, read bool) {
	if f.oldPath != f.newPath {
		if f.oldPath != "" && f.newPath != "" {
			written[f.newPath] = written[f.oldPath]
		} else {
			delete(written, f.newPath)
		}
		delete(written, f.oldPath)
	}
	old := written[f.newPath]
	if f.newPath == "" || len(f.hunks) == 0 || (!read && len(old) == 0) {
		return
	}
	var lines []bool
	at := 0
	for _, h := range f.hunks {
		// A hunk that deletes nothing names the line it comes after.
		first := h.start
		if h.deleted > 0 {
			first--
		}
		for ; at < first; at++ {
			lines = append(lines, at < len(old) && old[at])
		}
		at += h.deleted
		for range h.added {
			lines = append(lines, read)
		}
	}
	if at < len(old) {
		lines = append(lines, old[at:]...)
	}
	written[f.newPath] = lines
}

func tally(commits []commitDiff, capped bool) Report {
	report := Report{Capped: capped}
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
			if written[f.oldPath] {
				rewritten = countRewritten(f.deleted, added)
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
	return report
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
func countRewritten(lines []deletedLine, added map[string]int) int {
	moved := make([]bool, len(lines))
	for i, line := range lines {
		moved[i] = added[line.text] > 0 && letters(line.text) >= distinctive
	}
	// Short lines take a neighbouring moved line's part, through a run of lines.
	for changed := true; changed; {
		changed = false
		for i, line := range lines {
			if moved[i] || line.text == "" || added[line.text] == 0 {
				continue
			}
			before := i > 0 && moved[i-1] && lines[i-1].no == line.no-1
			after := i+1 < len(lines) && moved[i+1] && lines[i+1].no == line.no+1
			if before || after {
				moved[i], changed = true, true
			}
		}
	}
	rewritten := 0
	for i, line := range lines {
		switch {
		case line.text == "":
		case moved[i] && added[line.text] > 0:
			added[line.text]--
		case line.written:
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
		// The next deleted line's number in the file before.
		no := 0
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
				if h, ok := parseHunk(line); ok {
					f.hunks = append(f.hunks, h)
					no = h.start
				}
			case strings.HasPrefix(line, "+"):
				if text := strings.TrimSpace(line[1:]); text != "" {
					f.added = append(f.added, text)
				}
			case inHunks && strings.HasPrefix(line, "-"):
				f.deleted = append(f.deleted, deletedLine{no: no, text: strings.TrimSpace(line[1:])})
				no++
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

// parseHunk reads a hunk header, "@@ -start,count +start,count @@".
func parseHunk(header string) (h hunk, ok bool) {
	fields := strings.Fields(header)
	if len(fields) < 3 || !strings.HasPrefix(fields[1], "-") || !strings.HasPrefix(fields[2], "+") {
		return h, false
	}
	var okBefore, okAfter bool
	h.start, h.deleted, okBefore = span(fields[1][1:])
	_, h.added, okAfter = span(fields[2][1:])
	return h, okBefore && okAfter
}

// span reads "start,count", where a count left out is 1.
func span(text string) (start, count int, ok bool) {
	from, n, hasCount := strings.Cut(text, ",")
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
