package git

import "context"

func RebaseOnto(ctx context.Context, repoPath, onto string) error {
	_, err := runResult(ctx, repoPath, "rebase", onto)
	return err
}
