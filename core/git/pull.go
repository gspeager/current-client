package git

import (
	"context"

	"github.com/gspeager/current-client/core/gitexec"
)

func Pull(ctx context.Context, repoPath string) error {
	_, err := runStoppable(ctx, gitexec.Command{Dir: repoPath, Args: []string{"pull"}})
	return err
}

// PullMerge merges the upstream into the current branch, whatever pull.rebase
// is set to. Git needs one or the other once the branches have diverged.
func PullMerge(ctx context.Context, repoPath string) error {
	_, err := runStoppable(ctx, gitexec.Command{Dir: repoPath, Args: []string{"pull", "--no-rebase", "--no-edit"}})
	return err
}

// PullRebase replays local commits on top of the upstream instead of merging,
// whatever pull.rebase is set to.
func PullRebase(ctx context.Context, repoPath string) error {
	_, err := runStoppable(ctx, gitexec.Command{Dir: repoPath, Args: []string{"pull", "--rebase"}})
	return err
}
