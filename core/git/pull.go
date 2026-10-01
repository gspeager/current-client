package git

import "context"

func Pull(ctx context.Context, repoPath string) error {
	_, err := runResult(ctx, repoPath, "pull")
	return err
}
