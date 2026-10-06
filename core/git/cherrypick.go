package git

import (
	"context"
	"strings"
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
	args := []string{command, "--no-edit"}
	parents, err := runResult(ctx, repoPath, "rev-list", "--parents", "-n", "1", sha)
	if err != nil {
		return err
	}
	if len(strings.Fields(parents.Stdout)) > 2 {
		args = append(args, "-m", "1")
	}
	_, err = runResult(ctx, repoPath, append(args, sha)...)
	return err
}
