package app

import (
	"context"
	"errors"

	"github.com/gspeager/current-client/core/diff"
	"github.com/gspeager/current-client/core/gitexec"
)

type DiffService struct{}

type DiffLine struct {
	Kind    string `json:"kind"`
	OldLine int    `json:"oldLine"`
	NewLine int    `json:"newLine"`
	Content string `json:"content"`
	Moved   bool   `json:"moved"`
}

type DiffHunk struct {
	Header string     `json:"header"`
	Raw    string     `json:"raw"`
	Lines  []DiffLine `json:"lines"`
}

type FileDiff struct {
	Conflicted bool       `json:"conflicted"`
	OldPath    string     `json:"oldPath"`
	NewPath    string     `json:"newPath"`
	Binary     bool       `json:"binary"`
	TooLarge   bool       `json:"tooLarge"`
	SizeBytes  int64      `json:"sizeBytes"`
	Hunks      []DiffHunk `json:"hunks"`
}

func (s *DiffService) GetWorkingTreeDiff(repoPath, path string, force, ignoreWhitespace bool) (FileDiff, error) {
	fd, err := diff.GetWorkingTreeDiff(context.Background(), repoPath, path, force, ignoreWhitespace)
	if err != nil {
		return FileDiff{}, err
	}
	return toFileDiff(fd), nil
}

// GetIndexDiff takes origPath for a staged rename, so the diff shows the edit
// rather than the whole file as new. Like the working-tree diff, a large file
// is only diffed when force is set.
func (s *DiffService) GetIndexDiff(repoPath, path, origPath string, force, ignoreWhitespace bool) (FileDiff, error) {
	ctx := context.Background()
	if large, size := largestVersion(ctx, repoPath, path, force, ""); large {
		return FileDiff{TooLarge: true, SizeBytes: size}, nil
	}
	var fd diff.FileDiff
	var err error
	if origPath != "" {
		fd, err = diff.GetRenamedIndexDiff(ctx, repoPath, origPath, path, ignoreWhitespace)
	} else {
		fd, err = diff.GetIndexDiff(ctx, repoPath, path, ignoreWhitespace)
	}
	if err != nil {
		return FileDiff{}, err
	}
	return toFileDiff(fd), nil
}

func (s *DiffService) GetRefDiff(repoPath, path, fromRef, toRef string, force, ignoreWhitespace bool) (FileDiff, error) {
	ctx := context.Background()
	revs := []string{fromRef}
	if toRef != "" {
		revs = append(revs, toRef)
	}
	if large, size := largestVersion(ctx, repoPath, path, force, revs...); large {
		return FileDiff{TooLarge: true, SizeBytes: size}, nil
	}
	fd, err := diff.GetRefDiff(ctx, repoPath, path, fromRef, toRef, ignoreWhitespace)
	if err != nil {
		return FileDiff{}, err
	}
	return toFileDiff(fd), nil
}

// largestVersion reports whether path is over diff.LargeFileThreshold at any
// of revs ("" is the index), and its size there.
func largestVersion(ctx context.Context, repoPath, path string, force bool, revs ...string) (bool, int64) {
	if force {
		return false, 0
	}
	for _, rev := range revs {
		if size, found := diff.RevisionFileSize(ctx, repoPath, rev, path); found && size > diff.LargeFileThreshold {
			return true, size
		}
	}
	return false, 0
}

func (s *DiffService) StageHunk(repoPath, path, hunkText string) error {
	return diff.StageHunk(context.Background(), repoPath, path, hunkText)
}

func (s *DiffService) UnstageHunk(repoPath, path, hunkText string) error {
	return diff.UnstageHunk(context.Background(), repoPath, path, hunkText)
}

func (s *DiffService) DiscardHunk(repoPath, path, hunkText string) error {
	return diff.DiscardHunk(context.Background(), repoPath, path, hunkText)
}

func (s *DiffService) DiscardStagedHunk(repoPath, path, hunkText string) error {
	return diff.DiscardStagedHunk(context.Background(), repoPath, path, hunkText)
}

// LineSelection picks changed lines in a hunk: added lines by new line
// number, removed lines by old line number.
type LineSelection struct {
	Added   []int `json:"added"`
	Removed []int `json:"removed"`
}

func (s *DiffService) StageLines(repoPath, path, hunkText string, sel LineSelection) error {
	return diff.StageLines(context.Background(), repoPath, path, hunkText, diff.LineSelection(sel))
}

func (s *DiffService) UnstageLines(repoPath, path, hunkText string, sel LineSelection) error {
	return diff.UnstageLines(context.Background(), repoPath, path, hunkText, diff.LineSelection(sel))
}

func (s *DiffService) DiscardLines(repoPath, path, hunkText string, sel LineSelection) error {
	return diff.DiscardLines(context.Background(), repoPath, path, hunkText, diff.LineSelection(sel))
}

func (s *DiffService) OpenInExternalTool(repoPath, path string, cached bool) error {
	return diff.OpenInExternalTool(context.Background(), repoPath, path, cached)
}

func toFileDiff(fd diff.FileDiff) FileDiff {
	hunks := make([]DiffHunk, len(fd.Hunks))
	for i, h := range fd.Hunks {
		lines := make([]DiffLine, len(h.Lines))
		for j, l := range h.Lines {
			lines[j] = DiffLine{
				Kind:    lineKindString(l.Kind),
				OldLine: l.OldLine,
				NewLine: l.NewLine,
				Content: l.Content,
				Moved:   l.Moved,
			}
		}
		hunks[i] = DiffHunk{Header: h.Header, Raw: h.Raw, Lines: lines}
	}
	return FileDiff{
		Conflicted: fd.Conflicted,
		OldPath:    fd.OldPath,
		NewPath:    fd.NewPath,
		Binary:     fd.Binary,
		TooLarge:   fd.TooLarge,
		SizeBytes:  fd.SizeBytes,
		Hunks:      hunks,
	}
}

func lineKindString(k diff.LineKind) string {
	switch k {
	case diff.LineAdded:
		return "added"
	case diff.LineRemoved:
		return "removed"
	case diff.LineConflictOurs:
		return "conflict-ours"
	case diff.LineConflictTheirs:
		return "conflict-theirs"
	case diff.LineConflictMarker:
		return "conflict-marker"
	default:
		return "context"
	}
}

// FileSource says which version of a file to read: Kind is "commit" (at Rev),
// "index" or "worktree".
type FileSource struct {
	Kind string `json:"kind"`
	Rev  string `json:"rev"`
}

type FileContent struct {
	Found    bool   `json:"found"`
	TooLarge bool   `json:"tooLarge"`
	Data     []byte `json:"data"`
}

// Images are sent to the frontend whole, so very large ones are refused.
const maxFileContentBytes = 20 << 20

func (s *DiffService) GetFileContent(repoPath, path string, source FileSource) (FileContent, error) {
	var data []byte
	var found bool
	var err error
	switch source.Kind {
	case "commit":
		data, found, err = diff.ReadRevisionFile(context.Background(), repoPath, source.Rev, path, maxFileContentBytes)
	case "index":
		data, found, err = diff.ReadRevisionFile(context.Background(), repoPath, "", path, maxFileContentBytes)
	case "worktree":
		data, found, err = diff.ReadWorkingFile(repoPath, path, maxFileContentBytes)
	default:
		return FileContent{}, &gitexec.AppError{Message: "Unknown file version.", Detail: source.Kind}
	}
	if errors.Is(err, diff.ErrFileTooLarge) {
		return FileContent{Found: true, TooLarge: true}, nil
	}
	return FileContent{Found: found, Data: data}, err
}
