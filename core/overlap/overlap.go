// Package overlap predicts which branches would conflict with HEAD if merged,
// using in-memory merges (git merge-tree --write-tree, Git 2.38+).
package overlap

import (
	"context"
	"sync"

	"github.com/gspeager/current-client/core/git"
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

// Predict returns the local and remote-tracking branches whose merge with HEAD
// would conflict. It returns git.ErrMergeTreeUnsupported on Git before 2.38.
func Predict(ctx context.Context, repoPath string) ([]Overlap, error) {
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
	files := make([][]string, len(branches))
	errs := make([]error, len(branches))
	slots := make(chan struct{}, parallelMerges)
	var wg sync.WaitGroup
	for i, ref := range branches {
		wg.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()
			files[i], errs[i] = conflicts(ctx, repoPath, head, ref.SHA)
		})
	}
	wg.Wait()
	var overlaps []Overlap
	for i, ref := range branches {
		if errs[i] != nil {
			return nil, errs[i]
		}
		if len(files[i]) > 0 {
			overlaps = append(overlaps, Overlap{Branch: ref.Name, Remote: ref.Kind == git.RefRemoteBranch, Files: files[i]})
		}
	}
	return overlaps, nil
}

func conflicts(ctx context.Context, repoPath, head, tip string) ([]string, error) {
	key := [2]string{head, tip}
	cache.Lock()
	files, ok := cache.results[key]
	cache.Unlock()
	if ok {
		return files, nil
	}
	files, err := git.MergeConflicts(ctx, repoPath, head, tip)
	if err != nil {
		return nil, err
	}
	cache.Lock()
	if len(cache.results) >= cacheLimit {
		clear(cache.results)
	}
	cache.results[key] = files
	cache.Unlock()
	return files, nil
}
