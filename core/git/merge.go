package git

import "context"

func MergeBranch(ctx context.Context, repoPath, branch string) error {
	_, err := runResult(ctx, repoPath, "merge", "--no-edit", branch)
	return err
}
