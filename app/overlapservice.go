package app

import (
	"context"
	"errors"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/overlap"
)

type OverlapService struct{}

type OverlapInfo struct {
	Branch string   `json:"branch"`
	Remote bool     `json:"remote"`
	Files  []string `json:"files"`
}

// OverlapReport.Supported is false on Git before 2.38, and in a partial clone, where
// merging would fetch. The prediction is hidden rather than shown as an error.
type OverlapReport struct {
	Supported bool          `json:"supported"`
	Overlaps  []OverlapInfo `json:"overlaps"`
}

func (s *OverlapService) Predict(repoPath string) (OverlapReport, error) {
	overlaps, err := overlap.Predict(context.Background(), repoPath)
	if errors.Is(err, git.ErrMergeTreeUnsupported) || errors.Is(err, overlap.ErrPartialClone) {
		return OverlapReport{}, nil
	}
	if err != nil {
		return OverlapReport{}, err
	}
	return OverlapReport{
		Supported: true,
		Overlaps:  mapSlice(overlaps, func(o overlap.Overlap) OverlapInfo { return OverlapInfo(o) }),
	}, nil
}

// GridInfo is every local branch that isn't merged against every other. Supported
// is false on Git before 2.38, and Partial is true for a partial clone, where
// merging would fetch.
type GridInfo struct {
	Supported   bool               `json:"supported"`
	Partial     bool               `json:"partial"`
	Branches    []GridBranchInfo   `json:"branches"`
	Conflicts   []GridConflictInfo `json:"conflicts"`
	Unchecked   [][2]int           `json:"unchecked"`
	MaybeMerged int                `json:"maybeMerged"`
	Older       int                `json:"older"`
	Unrelated   []string           `json:"unrelated"`
}

type GridBranchInfo struct {
	Name           string `json:"name"`
	Current        bool   `json:"current"`
	LastCommitDate string `json:"lastCommitDate"`
}

// A and B are indexes of Branches.
type GridConflictInfo struct {
	A     int      `json:"a"`
	B     int      `json:"b"`
	Files []string `json:"files"`
}

func (s *OverlapService) Grid(repoPath string) (GridInfo, error) {
	grid, err := overlap.Compare(context.Background(), repoPath)
	if errors.Is(err, git.ErrMergeTreeUnsupported) {
		return GridInfo{}, nil
	}
	if errors.Is(err, overlap.ErrPartialClone) {
		return GridInfo{Supported: true, Partial: true}, nil
	}
	if err != nil {
		return GridInfo{}, err
	}
	return GridInfo{
		Supported: true,
		Branches: mapSlice(grid.Branches, func(b git.Branch) GridBranchInfo {
			return GridBranchInfo{Name: b.Name, Current: b.Current, LastCommitDate: b.LastCommitDate}
		}),
		Conflicts:   mapSlice(grid.Conflicts, func(c overlap.GridConflict) GridConflictInfo { return GridConflictInfo(c) }),
		Unchecked:   grid.Unchecked,
		MaybeMerged: grid.MaybeMerged,
		Older:       grid.Older,
		Unrelated:   grid.Unrelated,
	}, nil
}
