// Package overlap predicts which branches would conflict with HEAD if merged,
// using in-memory merges (git merge-tree --write-tree, Git 2.38+).
package overlap

import (
	"context"
	"errors"
	"sync"

	"github.com/gspeager/current-client/core/git"
	"golang.org/x/sync/singleflight"
)

type Overlap struct {
	Branch string
	Remote bool
	Files  []string
}

// A merge of two commits always has the same result, so results are cached by
// commit pair: a fetch that moves a branch simply misses the cache.
var cache = struct {
	sync.Mutex
	results map[[2]string][]string
}{results: map[[2]string][]string{}}

const cacheLimit = 4096

const parallelMerges = 4

// mergeMany is replaced in tests to stand in for Git 2.38 and 2.39.
var mergeMany = git.MergeConflictsMany

// flights makes callers that ask together, such as the views refreshing
// after an action, share one prediction.
var flights singleflight.Group

// Predict returns the local and remote-tracking branches whose merge with HEAD
// would conflict. It returns git.ErrMergeTreeUnsupported on Git before 2.38.
func Predict(ctx context.Context, repoPath string) ([]Overlap, error) {
	overlaps, err, _ := flights.Do(repoPath, func() (any, error) { return predict(ctx, repoPath) })
	if err != nil {
		return nil, err
	}
	return overlaps.([]Overlap), nil
}

func predict(ctx context.Context, repoPath string) ([]Overlap, error) {
	head, err := git.ResolveCommit(ctx, repoPath, "HEAD")
	if err != nil {
		return nil, nil
	}
	refs, err := git.ListRefs(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	var branches []git.Ref
	for _, ref := range refs {
		if ref.Kind != git.RefTag && ref.SHA != head {
			branches = append(branches, ref)
		}
	}
	tips := make([]string, len(branches))
	for i, ref := range branches {
		tips[i] = ref.SHA
	}
	files, err := conflictFiles(ctx, repoPath, head, tips)
	if err != nil {
		return nil, err
	}
	var overlaps []Overlap
	for i, ref := range branches {
		if len(files[i]) > 0 {
			overlaps = append(overlaps, Overlap{Branch: ref.Name, Remote: ref.Kind == git.RefRemoteBranch, Files: files[i]})
		}
	}
	return overlaps, nil
}

// conflictFiles merges each tip with head, answering from the cache where it
// can, and merging only the tips that share history with head where the
// batch refuses: an orphan branch such as gh-pages can't be merged, so it
// can't conflict either.
func conflictFiles(ctx context.Context, repoPath, head string, tips []string, namespaces ...string) ([][]string, error) {
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
		merged, err = mergeSharingHistory(ctx, repoPath, head, toMerge, namespaces...)
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

// merge merges every tip in one process, or one per process on Git 2.38 and 2.39.
func merge(ctx context.Context, repoPath, head string, tips []string) ([][]string, error) {
	merged, err := mergeMany(ctx, repoPath, head, tips)
	if errors.Is(err, git.ErrMergeTreeStdinUnsupported) {
		return mergeEach(ctx, repoPath, head, tips)
	}
	return merged, err
}

func mergeSharingHistory(ctx context.Context, repoPath, head string, tips []string, namespaces ...string) ([][]string, error) {
	related, err := git.SharingHistory(ctx, repoPath, head, namespaces...)
	if err != nil {
		return nil, err
	}
	var relatedTips []string
	for _, tip := range tips {
		if related[tip] {
			relatedTips = append(relatedTips, tip)
		}
	}
	merged := make([][]string, len(tips))
	if len(relatedTips) == 0 {
		return merged, nil
	}
	relatedMerged, err := merge(ctx, repoPath, head, relatedTips)
	if err != nil {
		return nil, err
	}
	next := 0
	for i, tip := range tips {
		if related[tip] {
			merged[i] = relatedMerged[next]
			next++
		}
	}
	return merged, nil
}

// mergeEach merges one branch per process, for Git without merge-tree --stdin.
func mergeEach(ctx context.Context, repoPath, head string, tips []string) ([][]string, error) {
	files := make([][]string, len(tips))
	errs := make([]error, len(tips))
	slots := make(chan struct{}, parallelMerges)
	var wg sync.WaitGroup
	for i, tip := range tips {
		wg.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()
			files[i], errs[i] = git.MergeConflicts(ctx, repoPath, head, tip)
		})
	}
	wg.Wait()
	return files, errors.Join(errs...)
}
