package app

import (
	"context"

	"github.com/gspeager/current-client/core/git"
)

type ConflictService struct{}

type ConflictStateInfo struct {
	Operation       string   `json:"operation"` // "", "merge", "cherry-pick", "revert", "rebase", or "am"
	ConflictedPaths []string `json:"conflictedPaths"`
}

func (s *ConflictService) GetConflictState(repoPath string) (ConflictStateInfo, error) {
	state, err := git.DetectConflictState(context.Background(), repoPath)
	if err != nil {
		return ConflictStateInfo{}, err
	}
	return ConflictStateInfo{Operation: string(state.Operation), ConflictedPaths: state.ConflictedPaths}, nil
}

func (s *ConflictService) Continue(repoPath string) error {
	return onConflictOperation(repoPath, git.ContinueConflictOperation)
}

func (s *ConflictService) Abort(repoPath string) error {
	return onConflictOperation(repoPath, git.AbortConflictOperation)
}

// onConflictOperation applies finish to whichever operation is in progress.
func onConflictOperation(repoPath string, finish func(context.Context, string, git.ConflictOperation) error) error {
	ctx := context.Background()
	state, err := git.DetectConflictState(ctx, repoPath)
	if err != nil {
		return err
	}
	return finish(ctx, repoPath, state.Operation)
}
