package app

import (
	"context"

	"github.com/gspeager/current-client/core/git"
)

type CompareService struct{}

func (s *CompareService) GetChangedFiles(repoPath, fromRef, toRef string) ([]ChangedFile, error) {
	ctx := context.Background()
	files, err := git.RefRangeChangedFiles(ctx, repoPath, fromRef, toRef)
	if err != nil {
		return nil, err
	}
	stats, err := git.RefRangeNumstat(ctx, repoPath, fromRef, toRef)
	if err != nil {
		return nil, err
	}
	return withLineCounts(files, stats), nil
}
