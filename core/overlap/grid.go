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
	// 780 pairs, about a second for branches that left the default one recently.
	mostBranches = 40
	// 66 pairs, for Git 2.38 and 2.39, where each is a process.
	mostBranchesOneByOne = 12
	// Branches looked at for ones that aren't merged.
	mostCandidates = 3 * mostBranches
	// Merging branches years apart takes seconds a pair. After this long, for each of
	// what's merged and what conflicts, nothing more is started.
	gridTime = 4 * time.Second
	// Merges a process, so what's started ends soon after.
	gridBatch = 5
)

// Grid is every local branch against every other: the newest of those not merged,
// and the pairs of them that would conflict.
type Grid struct {
	Branches  []git.Branch
	Conflicts []GridConflict
	// The pairs, and branches that may be merged, there was no time for. Comparing
	// again goes on with them.
	Unchecked   [][2]int
	MaybeMerged int
	// How many older branches there was no room for.
	Older int
	// The ones that share no history with the others, such as gh-pages.
	Unrelated []string
}

// GridConflict is a pair of branches or commits, by index, the first the smaller.
type GridConflict struct {
	A, B  int
	Files []string
}

// By two commits, which never change their answer: a base and a tip merged into it
// or not, and a commit and one it shares no history with.
var mergedInto, unrelated sync.Map

// ErrPartialClone means merging would fetch what the clone leaves out.
var ErrPartialClone = errors.New("merging would fetch what this partial clone leaves out")

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
			// There's no default branch to say which to leave out, or each shares
			// history with it and some pair of them doesn't.
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
	// The default branch, or "", and its tip.
	def         string
	anchor      string
	maybeMerged int
	older       int
}

// unmergedBranches takes one there was no time to check as not merged.
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

// newest is as many as the limit, with the one named keep in place of the last when
// it's older, and how many that leaves.
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

// findUnrelated says whether there are more than were known.
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

// Between merges every pair of commits, some of which may only be where env says. A
// pair sharing no history has nothing to conflict. unchecked are the pairs there was
// no time for.
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

// inBatches does some at a time, starting nothing after until or a failure, and
// returns what it didn't start.
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
