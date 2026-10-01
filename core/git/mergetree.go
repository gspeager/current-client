package git

import (
	"context"
	"errors"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

// ErrMergeTreeUnsupported means Git is older than 2.38, which added
// merge-tree --write-tree.
var ErrMergeTreeUnsupported = errors.New("predicting conflicts needs Git 2.38 or newer")

// MergeConflicts merges two commits in memory, without touching the working
// tree or index, and returns the files that would conflict.
func MergeConflicts(ctx context.Context, repoPath, ours, theirs string) ([]string, error) {
	result, err := gitexec.NewExecutor("").Run(ctx, gitexec.Command{
		Dir:  repoPath,
		Args: []string{"merge-tree", "--write-tree", "--name-only", "--no-messages", "-z", ours, theirs},
	})
	if err != nil {
		return nil, gitexec.WrapRunError(err)
	}
	switch result.ExitCode {
	case 0:
		return nil, nil
	case 1:
		// "<tree>\0<file>\0<file>\0"
		fields := strings.Split(strings.TrimSuffix(result.Stdout, "\x00"), "\x00")
		return fields[1:], nil
	case 129:
		return nil, ErrMergeTreeUnsupported
	}
	return nil, gitexec.WrapResult(result)
}
