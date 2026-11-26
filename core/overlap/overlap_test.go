package overlap

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/internal/gittest"
)

func TestPredict(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "shared.txt", "base\n", "edit shared.txt")
	gittest.CommitFile(t, dir, "other.txt", "base\n", "edit other.txt")

	gittest.Run(t, dir, "switch", "-c", "conflicting")
	gittest.CommitFile(t, dir, "shared.txt", "theirs\n", "edit shared.txt")
	gittest.Run(t, dir, "switch", "-c", "unrelated", "main")
	gittest.CommitFile(t, dir, "new.txt", "new\n", "edit new.txt")
	gittest.Run(t, dir, "switch", "-c", "same-file-clean", "main")
	gittest.CommitFile(t, dir, "other.txt", "base\nappended\n", "edit other.txt")
	gittest.Run(t, dir, "branch", "merged", "main")
	gittest.Run(t, dir, "switch", "main")
	gittest.CommitFile(t, dir, "shared.txt", "ours\n", "edit shared.txt")
	gittest.Run(t, dir, "tag", "v1")

	remote := t.TempDir()
	gittest.Run(t, remote, "init", "--bare", "-b", "main")
	gittest.Run(t, dir, "remote", "add", "origin", remote)
	gittest.Run(t, dir, "push", "origin", "conflicting")

	got, err := Predict(context.Background(), dir)
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}
	want := []Overlap{
		{Branch: "conflicting", Files: []string{"shared.txt"}},
		{Branch: "origin/conflicting", Remote: true, Files: []string{"shared.txt"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Predict = %+v, want %+v", got, want)
	}

	// A second call is answered from the cache, and a moved branch is re-checked.
	gittest.Run(t, dir, "switch", "unrelated")
	gittest.CommitFile(t, dir, "shared.txt", "also theirs\n", "edit shared.txt")
	gittest.Run(t, dir, "switch", "main")
	got, err = Predict(context.Background(), dir)
	if err != nil {
		t.Fatalf("Predict after moving a branch: %v", err)
	}
	if len(got) != 3 || got[1].Branch != "unrelated" {
		t.Errorf("Predict after moving unrelated = %+v, want it to conflict too", got)
	}
}

func TestPredictEmptyRepository(t *testing.T) {
	dir := gittest.InitRepo(t)
	if got, err := Predict(context.Background(), dir); err != nil || got != nil {
		t.Errorf("Predict = %+v, %v; want nothing for a repository with no commits", got, err)
	}
}

// Git 2.38 and 2.39 have merge-tree --write-tree but not --stdin.
func TestPredictFallsBackToOneMergeAtATime(t *testing.T) {
	mergeBatch = func(context.Context, string, [][2]string) ([][]string, error) {
		return nil, git.ErrMergeTreeStdinUnsupported
	}
	t.Cleanup(func() { mergeBatch = git.MergeConflictsBatch })

	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "shared.txt", "base\n", "base")
	gittest.Run(t, dir, "switch", "-c", "conflicting")
	gittest.CommitFile(t, dir, "shared.txt", "theirs\n", "theirs")
	gittest.Run(t, dir, "switch", "-c", "unrelated", "main")
	gittest.CommitFile(t, dir, "new.txt", "new\n", "unrelated")
	gittest.Run(t, dir, "switch", "main")
	gittest.CommitFile(t, dir, "shared.txt", "ours\n", "ours")

	got, err := Predict(context.Background(), dir)
	want := []Overlap{{Branch: "conflicting", Files: []string{"shared.txt"}}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Predict = %+v, %v; want %+v", got, err, want)
	}
}

// An orphan branch such as gh-pages shares no history, which merge-tree
// refuses to merge; it mustn't hide the other branches' conflicts.
func TestPredictWithABranchOfUnrelatedHistory(t *testing.T) {
	for _, tt := range []struct {
		name       string
		mergeBatch func(context.Context, string, [][2]string) ([][]string, error)
	}{
		{"in one batch", git.MergeConflictsBatch},
		{"one at a time", func(context.Context, string, [][2]string) ([][]string, error) {
			return nil, git.ErrMergeTreeStdinUnsupported
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mergeBatch = tt.mergeBatch
			t.Cleanup(func() { mergeBatch = git.MergeConflictsBatch })

			dir := gittest.InitRepo(t)
			gittest.CommitFile(t, dir, "shared.txt", "base\n", "base")
			gittest.Run(t, dir, "switch", "-c", "conflicting")
			gittest.CommitFile(t, dir, "shared.txt", "theirs\n", "theirs")
			gittest.Run(t, dir, "switch", "--orphan", "gh-pages")
			gittest.CommitFile(t, dir, "index.html", "site\n", "site")
			gittest.Run(t, dir, "switch", "main")
			gittest.CommitFile(t, dir, "shared.txt", "ours\n", "ours")

			got, err := Predict(context.Background(), dir)
			want := []Overlap{{Branch: "conflicting", Files: []string{"shared.txt"}}}
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Errorf("Predict = %+v, %v; want %+v", got, err, want)
			}
		})
	}
}

func branchNames(overlaps []Overlap) []string {
	var names []string
	for _, overlap := range overlaps {
		names = append(names, overlap.Branch)
	}
	return names
}

// agedConflicts makes a repository whose branches all conflict with HEAD,
// committed this many days ago, each one local and on origin.
func agedConflicts(t *testing.T, headDays int, branchDays ...int) string {
	t.Helper()
	dir := gittest.InitRepo(t)
	commit := func(content string, days int) {
		when := fmt.Sprintf("%d +0000", time.Now().AddDate(0, 0, -days).Unix())
		t.Setenv("GIT_AUTHOR_DATE", when)
		t.Setenv("GIT_COMMITTER_DATE", when)
		gittest.CommitFile(t, dir, "shared.txt", content, content)
	}
	commit("base\n", 900)
	for _, days := range branchDays {
		name := fmt.Sprintf("age-%d", days)
		gittest.Run(t, dir, "switch", "-c", name, "main")
		commit(name+"\n", days)
		gittest.Run(t, dir, "update-ref", "refs/remotes/origin/"+name, "HEAD")
	}
	gittest.Run(t, dir, "switch", "main")
	commit("ours\n", headDays)
	return dir
}

func TestPredictLeavesOutOldRemoteBranches(t *testing.T) {
	dir := agedConflicts(t, 1, 30, 89, 91, 800)
	got, err := Predict(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"age-30", "age-800", "age-89", "age-91", "origin/age-30", "origin/age-89"}
	if names := branchNames(got); !reflect.DeepEqual(names, want) {
		t.Errorf("Predict = %v, want every local branch and the recent remote ones: %v", names, want)
	}
}

func TestPredictCountsBackFromTheNewestBranch(t *testing.T) {
	dir := agedConflicts(t, 500, 510, 800)
	got, err := Predict(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"age-510", "age-800", "origin/age-510"}
	if names := branchNames(got); !reflect.DeepEqual(names, want) {
		t.Errorf("Predict in a quiet repository = %v, want %v", names, want)
	}
}

func TestPredictIgnoresAnEarlyClock(t *testing.T) {
	dir := agedConflicts(t, -400, 30)
	got, err := Predict(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"age-30", "origin/age-30"}
	if names := branchNames(got); !reflect.DeepEqual(names, want) {
		t.Errorf("Predict with HEAD dated ahead = %v, want %v", names, want)
	}
}
