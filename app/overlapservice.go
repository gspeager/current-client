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

// OverlapReport.Supported is false on Git before 2.38, where the prediction is
// hidden rather than shown as an error.
type OverlapReport struct {
	Supported bool          `json:"supported"`
	Overlaps  []OverlapInfo `json:"overlaps"`
}

func (s *OverlapService) Predict(repoPath string) (OverlapReport, error) {
	overlaps, err := overlap.Predict(context.Background(), repoPath)
	if errors.Is(err, git.ErrMergeTreeUnsupported) {
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
