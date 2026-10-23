package overlap

import (
	"context"

	"github.com/gspeager/current-client/core/git"
)

// PredictTips returns, for each commit in tips, the files that would
// conflict if it were merged with HEAD; an empty list means a clean merge.
// It shares Predict's commit-pair cache, so a tip already checked against
// the same HEAD runs no merge. tips are commits fetched under
// refs/current-client/ (a host's change heads), which Predict never sees.
func PredictTips(ctx context.Context, repoPath string, tips []string) ([][]string, error) {
	head, err := git.ResolveCommit(ctx, repoPath, "HEAD")
	if err != nil {
		return nil, err
	}
	return conflictFiles(ctx, repoPath, head, tips, "refs/heads", "refs/remotes", "refs/current-client")
}
