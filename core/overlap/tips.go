package overlap

import (
	"context"
	"errors"

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
	files := make([][]string, len(tips))
	var missing []int
	var toMerge []string
	cache.Lock()
	for i, tip := range tips {
		if cached, ok := cache.results[[2]string{head, tip}]; ok {
			files[i] = cached
		} else {
			missing = append(missing, i)
			toMerge = append(toMerge, tip)
		}
	}
	cache.Unlock()
	if len(toMerge) == 0 {
		return files, nil
	}
	merged, err := merge(ctx, repoPath, head, toMerge)
	if errors.Is(err, git.ErrUnrelatedHistories) {
		merged, err = mergeSharingHistoryIn(ctx, repoPath, head, toMerge, "refs/heads", "refs/remotes", "refs/current-client")
	}
	if err != nil {
		return nil, err
	}
	cache.Lock()
	if len(cache.results)+len(toMerge) > cacheLimit {
		clear(cache.results)
	}
	for k, i := range missing {
		files[i] = merged[k]
		cache.results[[2]string{head, toMerge[k]}] = merged[k]
	}
	cache.Unlock()
	return files, nil
}
