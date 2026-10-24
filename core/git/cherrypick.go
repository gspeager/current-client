package git

import (
	"context"

	"github.com/gspeager/current-client/core/gitexec"
)

func CherryPick(ctx context.Context, repoPath, sha string) error {
	return applyCommit(ctx, repoPath, "cherry-pick", sha)
}

func Revert(ctx context.Context, repoPath, sha string) error {
	return applyCommit(ctx, repoPath, "revert", sha)
}

// Git needs -m for a merge commit; parent 1 is the branch it was merged into,
// so the change applied is what the merge brought in.
func applyCommit(ctx context.Context, repoPath, command, sha string) error {
	parents, err := parentCount(ctx, repoPath, sha)
	if err != nil {
		return err
	}
	args := []string{command, "--no-edit"}
	if parents > 1 {
		args = append(args, "-m", "1")
	}
	_, err = runStoppable(ctx, gitexec.Command{Dir: repoPath, Args: append(args, sha)})
	return err
}
