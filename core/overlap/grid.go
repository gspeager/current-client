package overlap

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gspeager/current-client/core/git"
)

const (
	// 40 branches make 780 pairs, about a second for branches that left the default
	// one recently.
	mostBranches = 40
	// 12 branches make 66 pairs, for Git 2.38 and 2.39, where each pair is a process.
	mostBranchesOneByOne = 12
	// The newest branches checked for being merged, before mostBranches is taken.
	mostCandidates = 3 * mostBranches
	// Merging branches years apart takes seconds a pair. After this long, nothing
	// more is started, once for the merged check and once for the conflicts.
	gridTime = 4 * time.Second
	// Merges a process, few enough that the batches started before gridTime end
	// soon after it.
	gridBatch = 5
)

// Grid is every local branch against every other: the newest of those not merged,
// and the pairs of them that would conflict.
type Grid struct {
	Branches  []git.Branch
	Conflicts []GridConflict
	// Unchecked are the pairs there was no time to merge. Comparing again goes on
	// with them.
	Unchecked [][2]int
	// MaybeMerged counts the branches there was no time to check for being merged.
	MaybeMerged int
	// Older counts the older branches there was no room for.
	Older int
	// Unrelated are the branches that share no history with the others, such as gh-pages.
	Unrelated []string
}

// GridConflict is a pair of branches or commits by index, with A < B.
type GridConflict struct {
	A, B  int
	Files []string
}

// Both are keyed by a pair of commits, whose answer never changes: mergedInto by a
// base and a tip, to whether the tip is merged into it, and unrelated by two
// commits that share no history.
var mergedInto, unrelated sync.Map

// Compare merges every pair of branches that isn't merged yet. It returns
// git.ErrMergeTreeUnsupported on Git before 2.38, and ErrPartialClone.
func Compare(ctx context.Context, repoPath string) (Grid, error) {
	grid, err, _ := flights.Do("grid\x00"+repoPath, func() (any, error) { return compare(ctx, repoPath, gridTime) })
	if err != nil {
		return Grid{}, err
	}
	return grid.(Grid), nil
}

func compare(ctx context.Context, repoPath string, most time.Duration) (Grid, error) {
	if git.PartialClone(ctx, repoPath) {
		return Grid{}, ErrPartialClone
	}
	found, err := unmergedBranches(ctx, repoPath, most)
	if err != nil || len(found.branches) == 0 {
		return Grid{}, err
	}
	until := time.Now().Add(most)
	mergeSome, limit := mergeBatch, mostBranches
	for {
		grid := found.grid(limit)
		err := grid.merge(until, func(pairs [][2]string) ([][]string, error) { return mergeSome(ctx, repoPath, pairs) })
		switch {
		case err == nil:
			return grid, nil
		case errors.Is(err, git.ErrMergeTreeStdinUnsupported):
			mergeSome, limit = mergeOneByOne, mostBranchesOneByOne
		case errors.Is(err, git.ErrUnrelatedHistories):
			apart, err := found.findUnrelated(ctx, repoPath)
			if err != nil {
				return Grid{}, err
			}
			// No branch is newly found unrelated: there's no default branch to say which
			// to leave out, or each shares history with it while some pair doesn't.
			if !apart {
				mergeSome = git.MergeConflictsPairs
			}
		default:
			return Grid{}, err
		}
	}
}

// candidates are the branches that aren't merged, newest first, with the default
// branch among them.
type candidates struct {
	branches []git.Branch
	// def is the default branch, or "", and anchor its tip.
	def         string
	anchor      string
	maybeMerged int
	older       int
}

// unmergedBranches keeps a branch there was no time to check, as if it weren't merged.
func unmergedBranches(ctx context.Context, repoPath string, most time.Duration) (candidates, error) {
	branches, err := git.ListBranches(ctx, repoPath)
	if err != nil {
		return candidates{}, err
	}
	// A detached HEAD is listed too.
	branches = slices.DeleteFunc(branches, func(b git.Branch) bool { return strings.HasPrefix(b.Name, "(") })
	slices.SortStableFunc(branches, func(a, b git.Branch) int { return committedAt(b).Compare(committedAt(a)) })
	def, base, err := git.DefaultBase(ctx, repoPath)
	if err != nil || def == "" {
		return candidates{branches: branches}, err
	}
	found := candidates{def: def}
	for _, b := range branches {
		if b.Name == def {
			found.anchor = b.Tip
		}
	}
	found.branches, found.older = newest(branches, mostCandidates, def)
	baseTip, err := git.ResolveCommit(ctx, repoPath, base)
	if err != nil {
		return candidates{}, err
	}
	if found.maybeMerged, err = findMerged(ctx, repoPath, baseTip, found.branches, time.Now().Add(most)); err != nil {
		return candidates{}, err
	}
	found.branches = slices.DeleteFunc(found.branches, func(b git.Branch) bool {
		merged, _ := mergedInto.Load([2]string{baseTip, b.Tip})
		return merged == true && b.Name != def
	})
	return found, nil
}

// newest keeps the first limit branches, with the one named keep in place of the
// last when keep is older, and counts the branches left out.
func newest(branches []git.Branch, limit int, keep string) (kept []git.Branch, older int) {
	if len(branches) <= limit {
		return slices.Clone(branches), 0
	}
	kept = slices.Clone(branches[:limit])
	at := slices.IndexFunc(branches, func(b git.Branch) bool { return b.Name == keep })
	if at >= limit {
		kept[limit-1] = branches[at]
	}
	return kept, len(branches) - limit
}

func committedAt(b git.Branch) time.Time {
	at, _ := time.Parse(time.RFC3339, b.LastCommitDate)
	return at
}

func findMerged(ctx context.Context, repoPath, base string, branches []git.Branch, until time.Time) (unchecked int, err error) {
	var tips []string
	for _, b := range branches {
		if _, checked := mergedInto.Load([2]string{base, b.Tip}); !checked {
			tips = append(tips, b.Tip)
		}
	}
	left, err := inBatches(tips, until, func(batch []string) error {
		merged, err := git.MergedInto(ctx, repoPath, base, batch)
		for i := range merged {
			mergedInto.Store([2]string{base, batch[i]}, merged[i])
		}
		return err
	})
	return len(left), err
}

func (c candidates) grid(limit int) Grid {
	grid := Grid{MaybeMerged: c.maybeMerged}
	var related []git.Branch
	for _, branch := range c.branches {
		if _, apart := unrelated.Load([2]string{c.anchor, branch.Tip}); apart {
			grid.Unrelated = append(grid.Unrelated, branch.Name)
		} else {
			related = append(related, branch)
		}
	}
	grid.Branches, grid.Older = newest(related, limit, c.def)
	grid.Older += c.older
	return grid
}

// findUnrelated records the branches that share no history with the default
// branch, and says whether any was new.
func (c candidates) findUnrelated(ctx context.Context, repoPath string) (more bool, err error) {
	if c.anchor == "" {
		return false, nil
	}
	related, err := git.SharingHistory(ctx, repoPath, c.anchor, "refs/heads")
	if err != nil {
		return false, err
	}
	for _, branch := range c.branches {
		if !related[branch.Tip] {
			_, before := unrelated.LoadOrStore([2]string{c.anchor, branch.Tip}, true)
			more = more || !before
		}
	}
	return more, nil
}

// Between merges every pair of commits, some of which may only be in the object
// directories env names. A pair sharing no history has nothing to conflict.
// unchecked are the pairs there was no time for.
func Between(ctx context.Context, repoPath string, env, commits []string) (conflicts []GridConflict, unchecked [][2]int, err error) {
	return between(commits, time.Now().Add(gridTime), func(pairs [][2]string) ([][]string, error) {
		return git.MergeConflictsIn(ctx, repoPath, env, pairs)
	})
}

// merge fills in the grid, the newest branches first.
func (grid *Grid) merge(until time.Time, mergeSome func(pairs [][2]string) ([][]string, error)) (err error) {
	tips := make([]string, len(grid.Branches))
	for i, branch := range grid.Branches {
		tips[i] = branch.Tip
	}
	grid.Conflicts, grid.Unchecked, err = between(tips, until, mergeSome)
	return err
}

func between(tips []string, until time.Time, mergeSome func(pairs [][2]string) ([][]string, error)) (conflicts []GridConflict, unchecked [][2]int, err error) {
	var cells [][2]int
	var pairs [][2]string
	for b := range tips {
		for a := range b {
			cells = append(cells, [2]int{a, b})
			pairs = append(pairs, [2]string{tips[a], tips[b]})
		}
	}
	files, missing := known(pairs)
	left, err := inBatches(missing, until, func(batch []int) error {
		some := make([][2]string, len(batch))
		for k, i := range batch {
			some[k] = pairs[i]
		}
		merged, err := mergeSome(some)
		if err != nil {
			return err
		}
		remember(some, merged)
		for k, i := range batch {
			files[i] = merged[k]
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	for _, i := range left {
		unchecked = append(unchecked, cells[i])
	}
	for i, cell := range cells {
		if len(files[i]) > 0 {
			conflicts = append(conflicts, GridConflict{A: cell[0], B: cell[1], Files: files[i]})
		}
	}
	return conflicts, unchecked, nil
}

// inBatches runs do over gridBatch items at a time, starting nothing after until or
// a failure, and returns the items it didn't start.
func inBatches[T any](items []T, until time.Time, do func(batch []T) error) (left []T, err error) {
	var mu sync.Mutex
	var wg sync.WaitGroup
	slots := make(chan struct{}, parallelMerges)
	for batch := range slices.Chunk(items, gridBatch) {
		slots <- struct{}{}
		mu.Lock()
		stop := err != nil || !time.Now().Before(until)
		mu.Unlock()
		if stop {
			<-slots
			left = append(left, batch...)
			continue
		}
		wg.Go(func() {
			defer func() { <-slots }()
			if failed := do(batch); failed != nil {
				mu.Lock()
				err = errors.Join(err, failed)
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	return left, err
}

func mergeOneByOne(ctx context.Context, repoPath string, pairs [][2]string) ([][]string, error) {
	files := make([][]string, len(pairs))
	for i, pair := range pairs {
		var err error
		if files[i], err = git.MergeConflicts(ctx, repoPath, pair[0], pair[1]); err != nil {
			return nil, err
		}
	}
	return files, nil
}
