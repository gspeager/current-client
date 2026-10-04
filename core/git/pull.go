package git

import "context"

func Pull(ctx context.Context, repoPath string) error {
	_, err := runResult(ctx, repoPath, "pull")
	return err
}

// PullRebase replays local commits on top of the upstream instead of merging,
// whatever pull.rebase is set to.
func PullRebase(ctx context.Context, repoPath string) error {
	_, err := runResult(ctx, repoPath, "pull", "--rebase")
	return err
}
