package git

import (
	"context"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func reviewRow(t *testing.T, review BranchReview, name string) BranchReviewRow {
	t.Helper()
	for _, row := range review.Rows {
		if row.Name == name {
			return row
		}
	}
	t.Fatalf("no row for %s in %+v", name, review.Rows)
	return BranchReviewRow{}
}

func TestReviewBranches(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "branch", "-M", "main")
	gittest.CommitFile(t, dir, "file.txt", "one\n", "initial")

	gittest.Run(t, dir, "switch", "-c", "merged")
	gittest.CommitFile(t, dir, "merged.txt", "x\n", "merged work")
	gittest.Run(t, dir, "switch", "-c", "squashed", "main")
	gittest.CommitFile(t, dir, "squashed.txt", "a\n", "squashed one")
	gittest.CommitFile(t, dir, "squashed.txt", "a\nb\n", "squashed two")
	gittest.Run(t, dir, "switch", "-c", "unmerged", "main")
	gittest.CommitFile(t, dir, "unmerged.txt", "y\n", "unmerged one")
	gittest.CommitFile(t, dir, "unmerged.txt", "y\nz\n", "unmerged two")

	gittest.Run(t, dir, "switch", "main")
	gittest.Run(t, dir, "merge", "--no-ff", "-m", "merge", "merged")
	gittest.Run(t, dir, "merge", "--squash", "squashed")
	gittest.Run(t, dir, "commit", "-m", "squash")

	review, err := ReviewBranches(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if review.Default != "main" || review.Base != "main" {
		t.Fatalf("default %q base %q", review.Default, review.Base)
	}
	if row := reviewRow(t, review, "merged"); !row.Merged || row.AheadBase != 0 || row.BehindBase != 2 {
		t.Fatalf("merged = %+v", row)
	}
	if row := reviewRow(t, review, "squashed"); !row.Merged || row.AheadBase != 2 {
		t.Fatalf("squashed = %+v", row)
	}
	row := reviewRow(t, review, "unmerged")
	if row.Merged || row.AheadBase != 2 || row.BehindBase != 3 || row.AuthorName != "Test" || row.AuthorEmail != "test@example.com" {
		t.Fatalf("unmerged = %+v", row)
	}
	if row := reviewRow(t, review, "main"); !row.Current || row.AheadBase != 0 || row.BehindBase != 0 {
		t.Fatalf("main = %+v", row)
	}
}

func TestReviewBranchesComparesWithTheDefaultBranchesUpstream(t *testing.T) {
	remote := t.TempDir()
	gittest.Run(t, remote, "init", "--bare", "-b", "main")
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "branch", "-M", "main")
	gittest.CommitFile(t, dir, "file.txt", "one\n", "initial")
	gittest.Run(t, dir, "remote", "add", "origin", remote)
	gittest.Run(t, dir, "push", "-u", "origin", "main")
	gittest.Run(t, dir, "remote", "set-head", "origin", "main")
	gittest.CommitFile(t, dir, "file.txt", "two\n", "not pushed")

	review, err := ReviewBranches(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if review.Base != "origin/main" {
		t.Fatalf("base = %q", review.Base)
	}
	if row := reviewRow(t, review, "main"); row.AheadBase != 1 || row.Ahead != 1 || row.Upstream != "origin/main" {
		t.Fatalf("main = %+v", row)
	}
}

func TestParseCountPair(t *testing.T) {
	if got := parseCountPair("3 4\n"); got != [2]int{3, 4} {
		t.Fatalf("got %v", got)
	}
	if got := parseCountPair("nope"); got != [2]int{} {
		t.Fatalf("got %v", got)
	}
}
