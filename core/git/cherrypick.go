package git

import "context"

func CherryPick(ctx context.Context, repoPath, sha string) error {
	_, err := runResult(ctx, repoPath, "cherry-pick", "--no-edit", sha)
	return err
}

func Revert(ctx context.Context, repoPath, sha string) error {
	_, err := runResult(ctx, repoPath, "revert", "--no-edit", sha)
	return err
}
