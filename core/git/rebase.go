package git

import (
	"context"

	"github.com/gspeager/current-client/core/gitexec"
)

func RebaseOnto(ctx context.Context, repoPath, onto string) error {
	_, err := runStoppable(ctx, gitexec.Command{Dir: repoPath, Args: []string{"rebase", onto}})
	return err
}
