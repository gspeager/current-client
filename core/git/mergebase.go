package git

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/gspeager/current-client/core/gitexec"
)

const parallelProcesses = 4

// MergeBase is the last commit two lines of history share, "" when they
// share none.
func MergeBase(ctx context.Context, repoPath, a, b string) (string, error) {
	result, err := gitexec.NewExecutor("").Run(ctx, gitexec.Command{Dir: repoPath, Args: []string{"merge-base", a, b}})
	if err != nil {
		return "", gitexec.WrapRunError(err)
	}
	switch result.ExitCode {
	case 0:
		return strings.TrimSpace(result.Stdout), nil
	case 1:
		return "", nil
	}
	return "", gitexec.WrapResult(result)
}

// MergeBases is MergeBase for each pair, in the same order. Git answers
// merge-base for one pair at a time, so the pairs run a few processes at
// once rather than one after another.
func MergeBases(ctx context.Context, repoPath string, pairs [][2]string) ([]string, error) {
	bases := make([]string, len(pairs))
	errs := make([]error, len(pairs))
	slots := make(chan struct{}, parallelProcesses)
	var wg sync.WaitGroup
	for i, pair := range pairs {
		wg.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()
			bases[i], errs[i] = MergeBase(ctx, repoPath, pair[0], pair[1])
		})
	}
	wg.Wait()
	return bases, errors.Join(errs...)
}
