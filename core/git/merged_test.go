package git

import (
	"context"
	"reflect"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

// A clone with origin/main as the base, where squash, rebase and merge
// commits land on origin/main the way a host would make them.
func mergedFixture(t *testing.T) string {
	t.Helper()
	origin := gittest.InitRepo(t)
	gittest.CommitFile(t, origin, "a.txt", "a\n", "base")
	dir := t.TempDir()
	gittest.Run(t, origin, "clone", "-q", origin, dir)
	gittest.Run(t, dir, "config", "user.name", "T")
	gittest.Run(t, dir, "config", "user.email", "t@example.com")

	gittest.Run(t, dir, "switch", "-q", "-c", "squashed")
	gittest.CommitFile(t, dir, "s.txt", "s\n", "squashed one")
	gittest.CommitFile(t, dir, "s.txt", "s2\n", "squashed two")
	gittest.Run(t, dir, "push", "-q", "-u", "origin", "squashed")
	gittest.Run(t, dir, "switch", "-q", "-c", "rebased", "main")
	gittest.CommitFile(t, dir, "r.txt", "r\n", "rebased")
	gittest.Run(t, dir, "push", "-q", "-u", "origin", "rebased")
	gittest.Run(t, dir, "switch", "-q", "-c", "merged", "main")
	gittest.CommitFile(t, dir, "m.txt", "m\n", "merged")
	gittest.Run(t, dir, "push", "-q", "-u", "origin", "merged")
	gittest.Run(t, dir, "switch", "-q", "-c", "open", "main")
	gittest.CommitFile(t, dir, "o.txt", "o\n", "still open")
	gittest.Run(t, dir, "push", "-q", "-u", "origin", "open")
	gittest.Run(t, dir, "switch", "-q", "-c", "local-only", "main")
	gittest.CommitFile(t, dir, "l.txt", "l\n", "never pushed")

	gittest.Run(t, origin, "merge", "-q", "--squash", "squashed")
	gittest.Run(t, origin, "commit", "-q", "-m", "squashed (!1)")
	gittest.Run(t, origin, "cherry-pick", "rebased")
	gittest.Run(t, origin, "merge", "-q", "--no-ff", "-m", "merge merged", "merged")
	gittest.Run(t, origin, "branch", "-D", "squashed", "rebased", "merged")
	gittest.Run(t, dir, "fetch", "-q", "--prune")
	gittest.Run(t, dir, "switch", "-q", "main")
	return dir
}

func TestMergedBranchesFindsSquashRebaseAndMergeCommits(t *testing.T) {
	dir := mergedFixture(t)

	report, err := MergedBranches(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, b := range report.Branches {
		names = append(names, b.Name)
		if b.Newer {
			t.Errorf("%s flagged newer", b.Name)
		}
	}
	if report.Base != "origin/main" || !reflect.DeepEqual(names, []string{"merged", "rebased", "squashed"}) {
		t.Fatalf("report = %+v", report)
	}
}

func TestMergedBranchesFlagsCommitsPastTheMergedUpstream(t *testing.T) {
	dir := mergedFixture(t)
	gittest.Run(t, dir, "switch", "-q", "squashed")
	gittest.CommitFile(t, dir, "s.txt", "s3\n", "after the merge")
	gittest.Run(t, dir, "switch", "-q", "main")
	gittest.Run(t, dir, "update-ref", "refs/remotes/origin/squashed", "squashed~1")

	report, err := MergedBranches(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, b := range report.Branches {
		got[b.Name] = b.Newer
	}
	if !reflect.DeepEqual(got, map[string]bool{"merged": false, "rebased": false, "squashed": true}) {
		t.Fatalf("newer = %v", got)
	}
}

func TestMergedBranchesWithoutARemoteUsesTheLocalDefault(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "a.txt", "a\n", "base")
	gittest.Run(t, dir, "switch", "-q", "-c", "feature")
	gittest.CommitFile(t, dir, "f.txt", "f\n", "feature")
	gittest.Run(t, dir, "switch", "-q", "main")
	gittest.Run(t, dir, "merge", "-q", "--squash", "feature")
	gittest.Run(t, dir, "commit", "-q", "-m", "squashed")

	report, err := MergedBranches(context.Background(), dir)
	if err != nil || report.Base != "main" || len(report.Branches) != 0 {
		t.Fatalf("a branch without an upstream is not a candidate: %+v, %v", report, err)
	}
	gittest.Run(t, dir, "branch", "--set-upstream-to=main", "feature")
	report, err = MergedBranches(context.Background(), dir)
	if err != nil || len(report.Branches) != 1 || report.Branches[0].Name != "feature" {
		t.Fatalf("report = %+v, %v", report, err)
	}
}

func TestMergedIntoOneByOneMatchesTheBatch(t *testing.T) {
	dir := mergedFixture(t)
	ctx := context.Background()
	tips := []string{"squashed", "open", "rebased", "local-only", "merged"}

	batch, err := mergedTrees(ctx, dir, "origin/main", tips)
	if err != nil {
		t.Fatal(err)
	}
	single, err := mergedTreesOneByOne(ctx, dir, "origin/main", tips)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(batch, single) {
		t.Fatalf("batch %v != one by one %v", batch, single)
	}
	merged, err := MergedInto(ctx, dir, "origin/main", tips)
	if err != nil || !reflect.DeepEqual(merged, []bool{true, false, true, false, true}) {
		t.Fatalf("merged = %v, %v", merged, err)
	}
}
