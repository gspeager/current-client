package app

import (
	"context"

	"github.com/gspeager/current-client/core/git"
)

type ConflictService struct{}

type ConflictStateInfo struct {
	Operation       string   `json:"operation"` // "", "merge", "cherry-pick", "revert", "rebase", or "am"
	ConflictedPaths []string `json:"conflictedPaths"`
	// The branch names the resolve actions use: ours is the checked-out side.
	Ours   string `json:"ours"`
	Theirs string `json:"theirs"`
	// The message git prepared for the commit that finishes the operation.
	Message string `json:"message"`
}

func (s *ConflictService) GetConflictState(repoPath string) (ConflictStateInfo, error) {
	ctx := context.Background()
	state, err := git.DetectConflictState(ctx, repoPath)
	if err != nil {
		return ConflictStateInfo{}, err
	}
	info := ConflictStateInfo{Operation: string(state.Operation), ConflictedPaths: state.ConflictedPaths, Message: state.Message}
	if state.Operation != git.ConflictNone {
		if sides, err := git.ConflictSides(ctx, repoPath); err == nil {
			info.Ours, info.Theirs = sides.Ours, sides.Theirs
		}
	}
	return info, nil
}

// ContinueWith finishes the operation with message as the commit's message.
func (s *ConflictService) ContinueWith(repoPath, message string) error {
	return onConflictOperation(repoPath, func(ctx context.Context, repoPath string, op git.ConflictOperation) error {
		return git.ContinueWithMessage(ctx, repoPath, op, message)
	})
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

type ConflictSidesInfo struct {
	Ours   string `json:"ours"`
	Theirs string `json:"theirs"`
}

func (s *ConflictService) GetConflictSides(repoPath string) (ConflictSidesInfo, error) {
	sides, err := git.ConflictSides(context.Background(), repoPath)
	return ConflictSidesInfo(sides), err
}

// KeepSide resolves a conflicted file with one side's whole version, "ours"
// or "theirs", and stages it.
func (s *ConflictService) KeepSide(repoPath, path, side string) error {
	return git.KeepConflictSide(context.Background(), repoPath, path, git.ConflictSide(side))
}

// ConflictMarkerLine is the first conflict marker's line, or 0 when there is none.
func (s *ConflictService) ConflictMarkerLine(repoPath, path string) (int, error) {
	return git.ConflictMarkerLine(repoPath, path)
}
