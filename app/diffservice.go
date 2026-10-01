package app

import (
	"context"

	"github.com/gspeager/current-client/core/diff"
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

func (s *DiffService) GetIndexDiff(repoPath, path string, ignoreWhitespace bool) (FileDiff, error) {
	fd, err := diff.GetIndexDiff(context.Background(), repoPath, path, ignoreWhitespace)
	if err != nil {
		return FileDiff{}, err
	}
	return toFileDiff(fd), nil
}

func (s *DiffService) GetRefDiff(repoPath, path, fromRef, toRef string, ignoreWhitespace bool) (FileDiff, error) {
	fd, err := diff.GetRefDiff(context.Background(), repoPath, path, fromRef, toRef, ignoreWhitespace)
	if err != nil {
		return FileDiff{}, err
	}
	return toFileDiff(fd), nil
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
