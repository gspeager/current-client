package overlap

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/internal/gittest"
)

// gridRepo has a conflict between one and two, and between each and main.
func gridRepo(t *testing.T) string {
	t.Helper()
	dir := gittest.InitRepo(t)
	commitAt(t, dir, "shared.txt", "base\n", 100)
	for i, name := range []string{"one", "two"} {
		gittest.Run(t, dir, "switch", "-q", "-c", name, "main")
		commitAt(t, dir, "shared.txt", name+"\n", 20-i)
	}
	gittest.Run(t, dir, "switch", "-q", "-c", "clean", "main")
	commitAt(t, dir, "new.txt", "new\n", 10)
	gittest.Run(t, dir, "branch", "merged", "main")
	gittest.Run(t, dir, "switch", "-q", "main")
	commitAt(t, dir, "shared.txt", "ours\n", 5)
	return dir
}

func commitAt(t *testing.T, dir, name, content string, daysAgo int) {
	t.Helper()
	gittest.CommitAt(t, dir, name, content, "edit "+name, gittest.DaysAgo(daysAgo))
}

func compareAll(t *testing.T, dir string) Grid {
	t.Helper()
	grid, err := compare(context.Background(), dir, time.Minute)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	return grid
}

func names(grid Grid) []string {
	var names []string
	for _, b := range grid.Branches {
		names = append(names, b.Name)
	}
	return names
}

func conflicts(grid Grid) []string {
	var found []string
	for _, c := range grid.Conflicts {
		found = append(found, fmt.Sprintf("%s %s: %s", grid.Branches[c.A].Name, grid.Branches[c.B].Name, strings.Join(c.Files, " ")))
	}
	return found
}

func looseObjects(t *testing.T, dir string) string {
	t.Helper()
	return gittest.Run(t, dir, "count-objects", "-v")
}

func TestCompare(t *testing.T) {
	dir := gridRepo(t)
	before := looseObjects(t, dir)
	grid := compareAll(t, dir)
	if want := []string{"main", "clean", "two", "one"}; !reflect.DeepEqual(names(grid), want) {
		t.Errorf("branches = %v, want the newest first without the merged one: %v", names(grid), want)
	}
	want := []string{"main two: shared.txt", "main one: shared.txt", "two one: shared.txt"}
	if got := conflicts(grid); !reflect.DeepEqual(got, want) {
		t.Errorf("conflicts = %v, want %v", got, want)
	}
	if after := looseObjects(t, dir); after != before {
		t.Errorf("objects after = %s, before = %s", after, before)
	}

	// Cached until a branch moves, and then only its pairs are merged.
	merged := 0
	mergeBatch = func(ctx context.Context, repoPath string, pairs [][2]string) ([][]string, error) {
		merged += len(pairs)
		return git.MergeConflictsBatch(ctx, repoPath, pairs)
	}
	t.Cleanup(func() { mergeBatch = git.MergeConflictsBatch })
	if compareAll(t, dir); merged != 0 {
		t.Errorf("merged %d pairs again with nothing moved", merged)
	}
	gittest.Run(t, dir, "switch", "-q", "clean")
	commitAt(t, dir, "shared.txt", "clean too\n", 1)
	grid = compareAll(t, dir)
	if merged != 3 {
		t.Errorf("merged %d pairs after one of 4 branches moved, want 3", merged)
	}
	if len(grid.Conflicts) != 6 {
		t.Errorf("conflicts = %v, want every pair", conflicts(grid))
	}
}

func TestCompareWithASquashMergedBranch(t *testing.T) {
	dir := gridRepo(t)
	gittest.Run(t, dir, "merge", "-q", "--squash", "clean")
	gittest.Run(t, dir, "commit", "-qm", "squashed")
	if got := names(compareAll(t, dir)); !reflect.DeepEqual(got, []string{"main", "two", "one"}) {
		t.Errorf("branches = %v, want clean left out", got)
	}
}

func TestCompareWithoutADefaultBranch(t *testing.T) {
	dir := gridRepo(t)
	gittest.Run(t, dir, "branch", "-m", "main", "trunk-of-ours")
	gittest.Run(t, dir, "branch", "-q", "-D", "merged")
	grid := compareAll(t, dir)
	if len(grid.Branches) != 4 || len(grid.Conflicts) != 3 {
		t.Errorf("branches = %v, conflicts = %v", names(grid), conflicts(grid))
	}
}

func TestCompareADetachedHeadAndAnEmptyRepository(t *testing.T) {
	dir := gridRepo(t)
	gittest.Run(t, dir, "switch", "-q", "--detach", "one")
	if got := names(compareAll(t, dir)); len(got) != 4 {
		t.Errorf("branches with a detached HEAD = %v", got)
	}
	if grid := compareAll(t, gittest.InitRepo(t)); len(grid.Branches) != 0 {
		t.Errorf("an empty repository has %v", names(grid))
	}
}

func TestCompareLeavesOutUnrelatedHistories(t *testing.T) {
	for _, tt := range []struct {
		name  string
		batch func(context.Context, string, [][2]string) ([][]string, error)
	}{
		{"in one batch", git.MergeConflictsBatch},
		{"one at a time", func(context.Context, string, [][2]string) ([][]string, error) {
			return nil, git.ErrMergeTreeStdinUnsupported
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mergeBatch = tt.batch
			t.Cleanup(func() { mergeBatch = git.MergeConflictsBatch })
			dir := gridRepo(t)
			for _, name := range []string{"gh-pages", "notes"} {
				gittest.Run(t, dir, "switch", "-q", "--orphan", name)
				gittest.CommitFile(t, dir, name, name, name)
			}
			gittest.Run(t, dir, "switch", "-q", "main")
			for range 2 {
				grid := compareAll(t, dir)
				if want := []string{"main", "clean", "two", "one"}; !reflect.DeepEqual(names(grid), want) {
					t.Errorf("branches = %v, want %v", names(grid), want)
				}
				if len(grid.Unrelated) != 2 || len(grid.Conflicts) != 3 {
					t.Errorf("unrelated = %v, conflicts = %v", grid.Unrelated, conflicts(grid))
				}
			}
		})
	}
}

// Nothing says which of them to leave out.
func TestCompareUnrelatedHistoriesWithoutADefaultBranch(t *testing.T) {
	dir := gridRepo(t)
	gittest.Run(t, dir, "branch", "-q", "-D", "merged")
	gittest.Run(t, dir, "switch", "-q", "--orphan", "gh-pages")
	commitAt(t, dir, "index.html", "site", 0)
	gittest.Run(t, dir, "branch", "-m", "main", "trunk-of-ours")
	grid := compareAll(t, dir)
	if len(grid.Branches) != 5 || len(grid.Conflicts) != 3 || len(grid.Unrelated) != 0 {
		t.Errorf("branches = %v, conflicts = %v, unrelated = %v", names(grid), conflicts(grid), grid.Unrelated)
	}
}

func TestAPartialCloneNeverFetches(t *testing.T) {
	dir := gridRepo(t)
	gittest.Run(t, dir, "config", "uploadpack.allowFilter", "true")
	clone := filepath.Join(t.TempDir(), "clone")
	gittest.Run(t, dir, "clone", "-q", "--filter=blob:none", "file://"+filepath.ToSlash(dir), clone)
	gittest.Run(t, clone, "branch", "one", "origin/one")
	missing := func() string {
		return gittest.Run(t, clone, "rev-list", "--objects", "--all", "--missing=print")
	}
	before := missing()
	if !strings.Contains(before, "?") {
		t.Fatal("the clone has every file")
	}
	for name, check := range map[string]func() error{
		"Compare": func() error { _, err := Compare(context.Background(), clone); return err },
		"Predict": func() error { _, err := Predict(context.Background(), clone); return err },
	} {
		if err := check(); !errors.Is(err, ErrPartialClone) {
			t.Errorf("%s: %v, want ErrPartialClone", name, err)
		}
		if after := missing(); after != before {
			t.Errorf("%s fetched: missing after = %s, before = %s", name, after, before)
		}
	}
}

// Both roots are main's, so each side shares history with it and not with the other.
func TestCompareBranchesThatShareHistoryOnlyWithTheDefault(t *testing.T) {
	dir := gridRepo(t)
	gittest.Run(t, dir, "switch", "-q", "--orphan", "imported")
	gittest.CommitFile(t, dir, "imported.txt", "imported\n", "imported")
	gittest.Run(t, dir, "switch", "-q", "main")
	gittest.Run(t, dir, "merge", "-q", "--allow-unrelated-histories", "--no-edit", "imported")
	gittest.Run(t, dir, "switch", "-q", "imported")
	gittest.CommitFile(t, dir, "more.txt", "more\n", "more")
	grid := compareAll(t, dir)
	if len(grid.Branches) != 5 || len(grid.Unrelated) != 0 || len(grid.Conflicts) != 3 {
		t.Errorf("branches = %v, unrelated = %v, conflicts = %v", names(grid), grid.Unrelated, conflicts(grid))
	}
}

func TestCompareStopsAtALimit(t *testing.T) {
	dir := gridRepo(t)
	for i := range mostBranches {
		gittest.Run(t, dir, "switch", "-q", "-c", fmt.Sprintf("topic-%d", i), "main")
		commitAt(t, dir, fmt.Sprintf("topic-%d.txt", i), "topic\n", 30+i)
	}
	grid := compareAll(t, dir)
	if len(grid.Branches) != mostBranches || grid.Older != 4 {
		t.Errorf("%d branches, %d older; want %d and 4", len(grid.Branches), grid.Older, mostBranches)
	}
	if last := grid.Branches[mostBranches-1].Name; last != "topic-35" {
		t.Errorf("the oldest shown is %s, want topic-35", last)
	}

	// The default branch is kept however old it is.
	gittest.Run(t, dir, "branch", "-m", "main", "recent")
	gittest.Run(t, dir, "branch", "main", "recent~1")
	grid = compareAll(t, dir)
	if last := grid.Branches[mostBranches-1].Name; last != "main" {
		t.Errorf("the last of %v is %s, want main", names(grid), last)
	}

	mergeBatch = func(context.Context, string, [][2]string) ([][]string, error) {
		return nil, git.ErrMergeTreeStdinUnsupported
	}
	t.Cleanup(func() { mergeBatch = git.MergeConflictsBatch })
	cache.Lock()
	clear(cache.results)
	cache.Unlock()
	grid = compareAll(t, dir)
	if len(grid.Branches) != mostBranchesOneByOne || grid.Older != 45-mostBranchesOneByOne {
		t.Errorf("one at a time: %d branches, %d older", len(grid.Branches), grid.Older)
	}
}

func TestCompareStopsInTimeAndGoesOn(t *testing.T) {
	dir := gridRepo(t)
	grid, err := compare(context.Background(), dir, 0)
	if err != nil {
		t.Fatal(err)
	}
	// What's merged wasn't worked out either.
	if len(grid.Branches) != 5 || grid.MaybeMerged != 5 || len(grid.Unchecked) != 10 || len(grid.Conflicts) != 0 {
		t.Errorf("with no time: %v, %d unchecked, %v", names(grid), len(grid.Unchecked), conflicts(grid))
	}
	if grid = compareAll(t, dir); grid.MaybeMerged != 0 || len(grid.Unchecked) != 0 || len(grid.Conflicts) != 3 {
		t.Errorf("going on: %d unchecked, %v", len(grid.Unchecked), conflicts(grid))
	}
}
