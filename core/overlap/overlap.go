// Package overlap predicts which branches would conflict with HEAD if merged,
// using in-memory merges (git merge-tree --write-tree, Git 2.38+).
package overlap

import (
	"context"
	"errors"
	"sync"
	"time"

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

// Merging a remote branch years behind HEAD takes seconds, and a large
// repository keeps dozens, so one this much older than the newest is left out.
// Local branches are always checked.
const staleRemoteAge = 90 * 24 * time.Hour

// mergeBatch is replaced in tests to stand in for Git 2.38 and 2.39.
var mergeBatch = git.MergeConflictsBatch

// flights makes callers that ask together, such as the views refreshing
// after an action, share one prediction.
var flights singleflight.Group

// Predict returns the local branches, and the remote-tracking ones updated
// recently, whose merge with HEAD would conflict. It returns git.ErrMergeTreeUnsupported on Git before 2.38.
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
	oldest := oldestChecked(refs)
	for _, ref := range refs {
		stale := ref.Kind == git.RefRemoteBranch && ref.CommittedAt < oldest
		if ref.Kind != git.RefTag && ref.SHA != head && !stale {
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

// oldestChecked counts back from the newest branch rather than from today, so
// a repository nobody has touched for a year still shows its conflicts.
func oldestChecked(refs []git.Ref) int64 {
	var newest int64
	for _, ref := range refs {
		if ref.Kind != git.RefTag {
			newest = max(newest, ref.CommittedAt)
		}
	}
	// A commit dated in the future would make every other branch look old.
	return min(newest, time.Now().Unix()) - int64(staleRemoteAge.Seconds())
}

// conflictFiles merges each tip with head. Where the batch refuses, it merges only
// the tips that share history with head: an orphan branch such as gh-pages can't be
// merged, so it can't conflict either.
func conflictFiles(ctx context.Context, repoPath, head string, tips []string, namespaces ...string) ([][]string, error) {
	pairs := make([][2]string, len(tips))
	for i, tip := range tips {
		pairs[i] = [2]string{head, tip}
	}
	return cached(pairs, func(missing [][2]string) ([][]string, error) {
		merged, err := merge(ctx, repoPath, missing)
		if errors.Is(err, git.ErrUnrelatedHistories) {
			return mergeSharingHistory(ctx, repoPath, head, missing, namespaces...)
		}
		return merged, err
	})
}

// cached answers from the cache where it can, and merges the rest together.
func cached(pairs [][2]string, mergeMissing func(missing [][2]string) ([][]string, error)) ([][]string, error) {
	files, at := known(pairs)
	if len(at) == 0 {
		return files, nil
	}
	missing := make([][2]string, len(at))
	for k, i := range at {
		missing[k] = pairs[i]
	}
	merged, err := mergeMissing(missing)
	if err != nil {
		return nil, err
	}
	remember(missing, merged)
	for k, i := range at {
		files[i] = merged[k]
	}
	return files, nil
}

// known is what the cache answers, and the indexes of the pairs it doesn't.
func known(pairs [][2]string) (files [][]string, missing []int) {
	files = make([][]string, len(pairs))
	cache.Lock()
	defer cache.Unlock()
	for i, pair := range pairs {
		found, ok := cache.results[pair]
		if !ok {
			missing = append(missing, i)
		}
		files[i] = found
	}
	return files, missing
}

func remember(pairs [][2]string, files [][]string) {
	cache.Lock()
	defer cache.Unlock()
	if len(cache.results)+len(pairs) > cacheLimit {
		clear(cache.results)
	}
	for i, pair := range pairs {
		cache.results[pair] = files[i]
	}
}

// merge merges every pair in one process, or one per process on Git 2.38 and 2.39.
func merge(ctx context.Context, repoPath string, pairs [][2]string) ([][]string, error) {
	merged, err := mergeBatch(ctx, repoPath, pairs)
	if errors.Is(err, git.ErrMergeTreeStdinUnsupported) {
		return mergeEach(ctx, repoPath, pairs)
	}
	return merged, err
}

func mergeSharingHistory(ctx context.Context, repoPath, head string, pairs [][2]string, namespaces ...string) ([][]string, error) {
	related, err := git.SharingHistory(ctx, repoPath, head, namespaces...)
	if err != nil {
		return nil, err
	}
	var relatedPairs [][2]string
	for _, pair := range pairs {
		if related[pair[1]] {
			relatedPairs = append(relatedPairs, pair)
		}
	}
	merged := make([][]string, len(pairs))
	if len(relatedPairs) == 0 {
		return merged, nil
	}
	relatedMerged, err := merge(ctx, repoPath, relatedPairs)
	if err != nil {
		return nil, err
	}
	next := 0
	for i, pair := range pairs {
		if related[pair[1]] {
			merged[i] = relatedMerged[next]
			next++
		}
	}
	return merged, nil
}

func mergeEach(ctx context.Context, repoPath string, pairs [][2]string) ([][]string, error) {
	files := make([][]string, len(pairs))
	errs := make([]error, len(pairs))
	slots := make(chan struct{}, parallelMerges)
	var wg sync.WaitGroup
	for i, pair := range pairs {
		wg.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()
			files[i], errs[i] = git.MergeConflicts(ctx, repoPath, pair[0], pair[1])
		})
	}
	wg.Wait()
	return files, errors.Join(errs...)
}
