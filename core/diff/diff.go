package diff

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

const LargeFileThreshold = 1 << 20 // 1 MiB

type LineKind int

const (
	LineContext LineKind = iota
	LineAdded
	LineRemoved
	// The three kinds below only appear in a conflicted file's diff.
	LineConflictOurs
	LineConflictTheirs
	// Marker lines, plus the base section diff3-style conflicts add.
	LineConflictMarker
)

type Line struct {
	Kind    LineKind
	OldLine int // 1-based; 0 if the line doesn't exist in the old file
	NewLine int // 1-based; 0 if the line doesn't exist in the new file
	Content string
	// Moved flags an added or removed line whose block was relocated unchanged.
	Moved bool
}

type Hunk struct {
	Header   string
	OldStart int
	OldLines int
	NewStart int
	NewLines int
	Lines    []Line
	Raw      string // header + body lines, byte-for-byte, for StageHunk/UnstageHunk
}

type FileDiff struct {
	// Conflicted is set for git's combined diff of an unmerged file; line
	// numbers then count the working file (new) and our side (old).
	Conflicted bool
	OldPath    string
	NewPath    string
	Binary     bool
	TooLarge   bool
	SizeBytes  int64
	Hunks      []Hunk
}

func GetWorkingTreeDiff(ctx context.Context, repoPath, path string, force, ignoreWhitespace bool) (FileDiff, error) {
	if !force {
		if info, err := os.Stat(filepath.Join(repoPath, path)); err == nil && info.Size() > LargeFileThreshold {
			return FileDiff{TooLarge: true, SizeBytes: info.Size()}, nil
		}
	}
	return runDiff(ctx, repoPath, diffArgs(nil, ignoreWhitespace, path))
}

func GetIndexDiff(ctx context.Context, repoPath, path string, ignoreWhitespace bool) (FileDiff, error) {
	return runDiff(ctx, repoPath, diffArgs([]string{"--cached"}, ignoreWhitespace, path))
}

// An empty toRef compares fromRef against the working tree, not a second ref.
func GetRefDiff(ctx context.Context, repoPath, path, fromRef, toRef string, ignoreWhitespace bool) (FileDiff, error) {
	refs := []string{fromRef}
	if toRef != "" {
		refs = append(refs, toRef)
	}
	return runDiff(ctx, repoPath, diffArgs(refs, ignoreWhitespace, path))
}

func diffArgs(revArgs []string, ignoreWhitespace bool, path string) []string {
	args := []string{"diff", "--no-color", "-U20"}
	if ignoreWhitespace {
		args = append(args, "--ignore-all-space")
	}
	args = append(args, revArgs...)
	return append(args, "--", path)
}

func runDiff(ctx context.Context, repoPath string, args []string) (FileDiff, error) {
	result, err := gitexec.NewExecutor("").RunChecked(ctx, gitexec.Command{
		Dir:  repoPath,
		Args: args,
	})
	if err != nil {
		return FileDiff{}, err
	}
	return ParseUnifiedDiff(result.Stdout)
}

func StageHunk(ctx context.Context, repoPath, path, hunkText string) error {
	return applyHunk(ctx, repoPath, path, hunkText, applyOptions{cached: true})
}

func UnstageHunk(ctx context.Context, repoPath, path, hunkText string) error {
	return applyHunk(ctx, repoPath, path, hunkText, applyOptions{cached: true, reverse: true})
}

// DiscardHunk reverts one hunk in the working tree and leaves the index alone.
func DiscardHunk(ctx context.Context, repoPath, path, hunkText string) error {
	return applyHunk(ctx, repoPath, path, hunkText, applyOptions{reverse: true})
}

// DiscardStagedHunk reverts a staged hunk in both the index and the working
// tree. git apply is atomic, so if later unstaged edits touch the same lines
// it fails and changes nothing.
func DiscardStagedHunk(ctx context.Context, repoPath, path, hunkText string) error {
	return applyHunk(ctx, repoPath, path, hunkText, applyOptions{index: true, reverse: true})
}

type applyOptions struct {
	cached  bool
	index   bool
	reverse bool
}

func applyHunk(ctx context.Context, repoPath, path, hunkText string, opts applyOptions) error {
	oldPath, newPath := patchPaths(ctx, repoPath, path, hunkText, opts)
	patch := "diff --git a/" + path + " b/" + path + "\n--- " + oldPath + "\n+++ " + newPath + "\n" + hunkText

	args := []string{"apply"}
	if opts.cached {
		args = append(args, "--cached")
	}
	if opts.index {
		args = append(args, "--index")
	}
	if opts.reverse {
		args = append(args, "--reverse")
	}
	args = append(args, "-")

	_, err := gitexec.NewExecutor("").RunChecked(ctx, gitexec.Command{
		Dir:   repoPath,
		Args:  args,
		Stdin: strings.NewReader(patch),
	})
	return err
}

// patchPaths names a side of the patch /dev/null when the file doesn't exist
// there and the hunk leaves that side empty, so git apply adds or removes the
// file instead of leaving an empty one behind (unstaging a new file's only
// hunk, staging a deletion). An existing empty file keeps its name.
func patchPaths(ctx context.Context, repoPath, path, hunkText string, opts applyOptions) (oldPath, newPath string) {
	oldPath, newPath = "a/"+path, "b/"+path
	m := hunkHeaderPattern.FindStringSubmatch(hunkText)
	if m == nil {
		return oldPath, newPath
	}
	// The hunk is HEAD→index for unstaging and discarding a staged hunk, and
	// index→working tree otherwise.
	headToIndex := opts.index || (opts.cached && opts.reverse)
	if m[1] == "0" && m[2] == "0" {
		exists := objectExists(ctx, repoPath, ":"+path)
		if headToIndex {
			exists = objectExists(ctx, repoPath, "HEAD:"+path)
		}
		if !exists {
			oldPath = "/dev/null"
		}
	}
	if m[3] == "0" && m[4] == "0" {
		exists := objectExists(ctx, repoPath, ":"+path)
		if !headToIndex {
			_, err := os.Lstat(filepath.Join(repoPath, path))
			exists = err == nil
		}
		if !exists {
			newPath = "/dev/null"
		}
	}
	return oldPath, newPath
}

func objectExists(ctx context.Context, repoPath, spec string) bool {
	result, err := gitexec.NewExecutor("").Run(ctx, gitexec.Command{Dir: repoPath, Args: []string{"cat-file", "-e", spec}})
	return err == nil && result.ExitCode == 0
}

var hunkHeaderPattern = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// Git prints a conflicted file as a two-parent combined diff (`diff --cc`).
var combinedHunkHeaderPattern = regexp.MustCompile(`^@@@ -(\d+)(?:,(\d+))? -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@@`)

type conflictRegion int

const (
	outsideConflict conflictRegion = iota
	inOurs
	inBase
	inTheirs
)

func ParseUnifiedDiff(output string) (FileDiff, error) {
	if strings.TrimSpace(output) == "" {
		return FileDiff{}, nil
	}

	p := &diffParser{}
	for _, line := range strings.Split(output, "\n") {
		handled, err := p.parseHeaderLine(line)
		if err != nil {
			return FileDiff{}, err
		}
		if handled || p.hunk == nil || line == "" {
			continue
		}
		if err := p.parseBodyLine(line); err != nil {
			return FileDiff{}, err
		}
	}
	p.finishHunk()
	markMovedLines(&p.fd)
	return p.fd, nil
}

type diffParser struct {
	fd        FileDiff
	hunk      *Hunk
	oldLineNo int
	newLineNo int
	region    conflictRegion
	// The current hunk's Raw, built up here: appending to a string line by
	// line made a 40,000-line diff take seconds to parse.
	raw strings.Builder
}

// startHunk takes the old/new start and length groups matched by either
// hunk header pattern.
func (p *diffParser) startHunk(header string, m []string) {
	p.finishHunk()
	p.fd.Hunks = append(p.fd.Hunks, Hunk{
		Header:   header,
		OldStart: atoi(m[1]),
		OldLines: atoiOrDefault(m[2], 1),
		NewStart: atoi(m[3]),
		NewLines: atoiOrDefault(m[4], 1),
	})
	p.hunk = &p.fd.Hunks[len(p.fd.Hunks)-1]
	p.oldLineNo, p.newLineNo = p.hunk.OldStart, p.hunk.NewStart
	p.region = outsideConflict
	p.appendRaw(header)
}

func (p *diffParser) appendRaw(line string) {
	p.raw.WriteString(line)
	p.raw.WriteByte('\n')
}

func (p *diffParser) finishHunk() {
	if p.hunk != nil {
		p.hunk.Raw = p.raw.String()
		p.raw.Reset()
	}
}

// parseHeaderLine reports handled=false for hunk content lines.
func (p *diffParser) parseHeaderLine(line string) (handled bool, err error) {
	switch {
	case strings.HasPrefix(line, "--- "):
		p.fd.OldPath = trimDiffPathPrefix(strings.TrimPrefix(line, "--- "))
		return true, nil
	case strings.HasPrefix(line, "+++ "):
		p.fd.NewPath = trimDiffPathPrefix(strings.TrimPrefix(line, "+++ "))
		return true, nil
	case strings.HasPrefix(line, "@@ "):
		m := hunkHeaderPattern.FindStringSubmatch(line)
		if m == nil {
			return true, fmt.Errorf("malformed hunk header: %q", line)
		}
		p.startHunk(line, m)
		return true, nil
	case strings.HasPrefix(line, "@@@ "):
		m := combinedHunkHeaderPattern.FindStringSubmatch(line)
		if m == nil {
			return true, fmt.Errorf("malformed combined hunk header: %q", line)
		}
		p.startHunk(line, m)
		return true, nil
	case strings.HasPrefix(line, "diff --cc "):
		p.fd.Conflicted = true
		return true, nil
	case strings.HasPrefix(line, "\\"):
		if p.hunk != nil {
			p.appendRaw(line)
		}
		return true, nil // git's "no newline at end of file" marker, not content
	case strings.HasPrefix(line, "Binary files ") && strings.HasSuffix(line, " differ"):
		p.fd.Binary = true
		return true, nil
	case strings.HasPrefix(line, "diff --git "),
		strings.HasPrefix(line, "index "),
		strings.HasPrefix(line, "old mode "),
		strings.HasPrefix(line, "new mode "),
		strings.HasPrefix(line, "new file mode "),
		strings.HasPrefix(line, "deleted file mode "),
		strings.HasPrefix(line, "similarity index "),
		strings.HasPrefix(line, "rename from "),
		strings.HasPrefix(line, "rename to "),
		strings.HasPrefix(line, "copy from "),
		strings.HasPrefix(line, "copy to "):
		return true, nil
	}
	return false, nil
}

func (p *diffParser) parseBodyLine(line string) error {
	if p.fd.Conflicted {
		return p.parseCombinedBodyLine(line)
	}
	switch line[0] {
	case '+':
		p.hunk.Lines = append(p.hunk.Lines, Line{Kind: LineAdded, NewLine: p.newLineNo, Content: line[1:]})
		p.newLineNo++
	case '-':
		p.hunk.Lines = append(p.hunk.Lines, Line{Kind: LineRemoved, OldLine: p.oldLineNo, Content: line[1:]})
		p.oldLineNo++
	case ' ':
		p.hunk.Lines = append(p.hunk.Lines, Line{Kind: LineContext, OldLine: p.oldLineNo, NewLine: p.newLineNo, Content: line[1:]})
		p.oldLineNo++
		p.newLineNo++
	default:
		return fmt.Errorf("malformed diff line: %q", line)
	}
	p.appendRaw(line)
	return nil
}

// Each combined line has one column per parent: ' ' in that parent and the
// working file, '+' only in the working file, '-' only in that parent. Lines
// in the working file are classified by the conflict markers around them.
func (p *diffParser) parseCombinedBodyLine(line string) error {
	if len(line) < 2 {
		return fmt.Errorf("malformed combined diff line: %q", line)
	}
	ours, theirs, content := line[0], line[1], line[2:]
	p.appendRaw(line)

	if ours == '-' || theirs == '-' {
		l := Line{Kind: LineRemoved, Content: content}
		if ours == '-' {
			l.OldLine = p.oldLineNo
			p.oldLineNo++
		}
		p.hunk.Lines = append(p.hunk.Lines, l)
		return nil
	}

	l := Line{Kind: LineContext, NewLine: p.newLineNo, Content: content}
	p.newLineNo++
	if ours == ' ' {
		l.OldLine = p.oldLineNo
		p.oldLineNo++
	}
	if region, isMarker := conflictMarkerRegion(content); isMarker && ours == '+' && theirs == '+' {
		l.Kind = LineConflictMarker
		p.region = region
	} else {
		switch p.region {
		case inOurs:
			l.Kind = LineConflictOurs
		case inTheirs:
			l.Kind = LineConflictTheirs
		case inBase:
			l.Kind = LineConflictMarker
		}
	}
	p.hunk.Lines = append(p.hunk.Lines, l)
	return nil
}

// conflictMarkerRegion reports which section a marker line opens.
func conflictMarkerRegion(content string) (conflictRegion, bool) {
	switch {
	case strings.HasPrefix(content, "<<<<<<<"):
		return inOurs, true
	case strings.HasPrefix(content, "|||||||"):
		return inBase, true
	case strings.HasPrefix(content, "======="):
		return inTheirs, true
	case strings.HasPrefix(content, ">>>>>>>"):
		return outsideConflict, true
	}
	return outsideConflict, false
}

func trimDiffPathPrefix(path string) string {
	if path == "/dev/null" {
		return ""
	}
	if len(path) > 2 && (path[:2] == "a/" || path[:2] == "b/") {
		return path[2:]
	}
	return path
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func atoiOrDefault(s string, def int) int {
	if s == "" {
		return def
	}
	return atoi(s)
}
