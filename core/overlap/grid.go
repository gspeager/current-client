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
	// The pairs there was no time for. Comparing again goes on with them.
	Unchecked [][2]int
	// How many older branches there was no room for.
	Older int
	// The ones that share no history with the others, such as gh-pages.
	Unrelated []string
}

// GridConflict is a pair of Branches, by index, the first the smaller.
type GridConflict struct {
	A, B  int
	Files []string
}

// By two commits, which never change their answer: a base and a tip merged into it
// or not, and a commit and one it shares no history with.
var mergedInto, unrelated sync.Map

// Compare merges every pair of branches that isn't merged yet. It returns
// git.ErrMergeTreeUnsupported on Git before 2.38.
func Compare(ctx context.Context, repoPath string) (Grid, error) {
	grid, err, _ := flights.Do("grid\x00"+repoPath, func() (any, error) { return compare(ctx, repoPath, gridTime) })
	if err != nil {
		return Grid{}, err
	}
	return grid.(Grid), nil
}

func compare(ctx context.Context, repoPath string, most time.Duration) (Grid, error) {
	branches, anchor, older, err := unmergedBranches(ctx, repoPath, most)
	if err != nil || len(branches) == 0 {
		return Grid{}, err
	}
	until := time.Now().Add(most)
	mergeSome, limit := mergeBatch, mostBranches
	for {
		grid := newGrid(branches, anchor, limit)
		grid.Older += older
		err := grid.merge(until, func(pairs [][2]string) ([][]string, error) { return mergeSome(ctx, repoPath, pairs) })
		switch {
		case err == nil:
			return grid, nil
		case errors.Is(err, git.ErrMergeTreeStdinUnsupported):
			mergeSome, limit = mergeOneByOne, mostBranchesOneByOne
		case errors.Is(err, git.ErrUnrelatedHistories):
			found, err := findUnrelated(ctx, repoPath, anchor, branches)
			if err != nil {
				return Grid{}, err
			}
			// Each shares history with the anchor, and some pair of them doesn't.
			if !found {
				mergeSome = git.MergeConflictsPairs
			}
		default:
			return Grid{}, err
		}
	}
}

// unmergedBranches are newest first, with the default branch among them, whose tip
// is the anchor. One there was no time to check is taken as not merged.
func unmergedBranches(ctx context.Context, repoPath string, most time.Duration) (branches []git.Branch, anchor string, older int, err error) {
	branches, err = git.ListBranches(ctx, repoPath)
	if err != nil {
		return nil, "", 0, err
	}
	// A detached HEAD is listed too.
	branches = slices.DeleteFunc(branches, func(b git.Branch) bool { return strings.HasPrefix(b.Name, "(") })
	if len(branches) == 0 {
		return nil, "", 0, nil
	}
	slices.SortStableFunc(branches, func(a, b git.Branch) int { return committedAt(b).Compare(committedAt(a)) })
	def, base, err := git.DefaultBase(ctx, repoPath)
	if err != nil || def == "" {
		return branches, branches[0].Tip, 0, err
	}
	i := slices.IndexFunc(branches, func(b git.Branch) bool { return b.Name == def })
	anchor = branches[i].Tip
	if len(branches) > mostCandidates {
		kept := slices.Clone(branches[:mostCandidates])
		if i >= mostCandidates {
			kept[mostCandidates-1] = branches[i]
		}
		older = len(branches) - mostCandidates
		branches = kept
	}
	baseTip, err := git.ResolveCommit(ctx, repoPath, base)
	if err != nil {
		return nil, "", 0, err
	}
	if err := findMerged(ctx, repoPath, baseTip, branches, time.Now().Add(most)); err != nil {
		return nil, "", 0, err
	}
	branches = slices.DeleteFunc(branches, func(b git.Branch) bool {
		merged, _ := mergedInto.Load([2]string{baseTip, b.Tip})
		return merged == true && b.Name != def
	})
	return branches, anchor, older, nil
}

func committedAt(b git.Branch) time.Time {
	at, _ := time.Parse(time.RFC3339, b.LastCommitDate)
	return at
}

func findMerged(ctx context.Context, repoPath, base string, branches []git.Branch, until time.Time) error {
	var tips []string
	for _, b := range branches {
		if _, checked := mergedInto.Load([2]string{base, b.Tip}); !checked {
			tips = append(tips, b.Tip)
		}
	}
	_, err := inBatches(tips, until, func(batch []string) error {
		merged, err := git.MergedInto(ctx, repoPath, base, batch)
		for i := range merged {
			mergedInto.Store([2]string{base, batch[i]}, merged[i])
		}
		return err
	})
	return err
}

func newGrid(branches []git.Branch, anchor string, limit int) Grid {
	var grid Grid
	for _, branch := range branches {
		if _, apart := unrelated.Load([2]string{anchor, branch.Tip}); apart {
			grid.Unrelated = append(grid.Unrelated, branch.Name)
		} else if len(grid.Branches) < limit {
			grid.Branches = append(grid.Branches, branch)
		} else {
			grid.Older++
		}
	}
	return grid
}

func findUnrelated(ctx context.Context, repoPath, anchor string, branches []git.Branch) (found bool, err error) {
	related, err := git.SharingHistory(ctx, repoPath, anchor, "refs/heads")
	if err != nil {
		return false, err
	}
	for _, branch := range branches {
		if !related[branch.Tip] {
			_, before := unrelated.LoadOrStore([2]string{anchor, branch.Tip}, true)
			found = found || !before
		}
	}
	return found, nil
}

// merge fills in the grid, the newest branches first.
func (grid *Grid) merge(until time.Time, mergeSome func(pairs [][2]string) ([][]string, error)) error {
	var cells [][2]int
	var pairs [][2]string
	for b := range grid.Branches {
		for a := range b {
			cells = append(cells, [2]int{a, b})
			pairs = append(pairs, [2]string{grid.Branches[a].Tip, grid.Branches[b].Tip})
		}
	}
	files, missing := known(pairs)
	unchecked, err := inBatches(missing, until, func(batch []int) error {
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
		return err
	}
	for _, i := range unchecked {
		grid.Unchecked = append(grid.Unchecked, cells[i])
	}
	for i, cell := range cells {
		if len(files[i]) > 0 {
			grid.Conflicts = append(grid.Conflicts, GridConflict{A: cell[0], B: cell[1], Files: files[i]})
		}
	}
	return nil
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
