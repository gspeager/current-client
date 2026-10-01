package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gspeager/current-client/core/gitexec"
)

type ConflictOperation string

const (
	ConflictNone       ConflictOperation = ""
	ConflictMerge      ConflictOperation = "merge"
	ConflictCherryPick ConflictOperation = "cherry-pick"
	ConflictRevert     ConflictOperation = "revert"
	ConflictRebase     ConflictOperation = "rebase"
	ConflictAm         ConflictOperation = "am"
)

type ConflictState struct {
	Operation       ConflictOperation
	ConflictedPaths []string
}

// DetectConflictState reports an in-progress operation even once every
// conflict is resolved, since it still needs continuing or aborting.
func DetectConflictState(ctx context.Context, repoPath string) (ConflictState, error) {
	dir, err := GitDir(ctx, repoPath)
	if err != nil {
		return ConflictState{}, err
	}

	op := ConflictNone
	switch {
	case fileExists(filepath.Join(dir, "MERGE_HEAD")):
		op = ConflictMerge
	case fileExists(filepath.Join(dir, "CHERRY_PICK_HEAD")):
		op = ConflictCherryPick
	case fileExists(filepath.Join(dir, "REVERT_HEAD")):
		op = ConflictRevert
	case dirExists(filepath.Join(dir, "rebase-merge")):
		op = ConflictRebase
	case dirExists(filepath.Join(dir, "rebase-apply")):
		// Shared by `git am` and apply-backend rebases; only am leaves an
		// "applying" marker, and `rebase --continue` refuses to resume an am.
		if fileExists(filepath.Join(dir, "rebase-apply", "applying")) {
			op = ConflictAm
		} else {
			op = ConflictRebase
		}
	}
	if op == ConflictNone {
		return ConflictState{}, nil
	}

	statuses, err := GetStatus(ctx, repoPath)
	if err != nil {
		return ConflictState{}, err
	}
	var paths []string
	for _, s := range statuses {
		if s.Conflicted {
			paths = append(paths, s.Path)
		}
	}
	return ConflictState{Operation: op, ConflictedPaths: paths}, nil
}

// ContinueConflictOperation sets GIT_EDITOR=true so git accepts its prepared
// message instead of hanging on an editor that can never open.
func ContinueConflictOperation(ctx context.Context, repoPath string, op ConflictOperation) error {
	args, err := conflictOpArgs(op, "--continue")
	if err != nil {
		return err
	}
	_, err = gitexec.NewExecutor("").RunChecked(ctx, gitexec.Command{
		Dir:  repoPath,
		Args: args,
		Env:  []string{"GIT_EDITOR=true"},
	})
	return err
}

func AbortConflictOperation(ctx context.Context, repoPath string, op ConflictOperation) error {
	args, err := conflictOpArgs(op, "--abort")
	if err != nil {
		return err
	}
	_, err = runResult(ctx, repoPath, args...)
	return err
}

func conflictOpArgs(op ConflictOperation, flag string) ([]string, error) {
	switch op {
	case ConflictMerge:
		return []string{"merge", flag}, nil
	case ConflictCherryPick:
		return []string{"cherry-pick", flag}, nil
	case ConflictRevert:
		return []string{"revert", flag}, nil
	case ConflictRebase:
		return []string{"rebase", flag}, nil
	case ConflictAm:
		return []string{"am", flag}, nil
	default:
		return nil, fmt.Errorf("no conflict operation in progress")
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
