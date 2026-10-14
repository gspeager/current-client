package git

import (
	"context"
	"testing"
	"time"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestRepoStats(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init", "-b", "main")
	gittest.Run(t, dir, "config", "user.email", "alice@example.com")
	gittest.Run(t, dir, "config", "user.name", "Alice")
	gittest.CommitFile(t, dir, "main.go", "v1", "initial")
	gittest.CommitFile(t, dir, "util.go", "v1", "add util")
	gittest.Run(t, dir, "tag", "v1.0.0")
	gittest.Run(t, dir, "branch", "feature")
	gittest.WriteFile(t, dir, "main.go", "v2")
	gittest.Run(t, dir, "stash", "push")

	stats, err := ComputeRepoStats(context.Background(), dir)
	if err != nil {
		t.Fatalf("ComputeRepoStats: %v", err)
	}

	if stats.TotalCommits != 2 {
		t.Errorf("TotalCommits = %d, want 2", stats.TotalCommits)
	}
	if stats.ContributorCount != 1 {
		t.Errorf("ContributorCount = %d, want 1", stats.ContributorCount)
	}
	if stats.FileCount != 2 {
		t.Errorf("FileCount = %d, want 2", stats.FileCount)
	}
	if stats.RepoAgeDays != 0 {
		t.Errorf("RepoAgeDays = %d, want 0 for a repo created moments ago", stats.RepoAgeDays)
	}
	if stats.BranchCount != 2 {
		t.Errorf("BranchCount = %d, want 2 (main, feature)", stats.BranchCount)
	}
	if stats.TagCount != 1 {
		t.Errorf("TagCount = %d, want 1", stats.TagCount)
	}
	if stats.StashCount != 1 {
		t.Errorf("StashCount = %d, want 1", stats.StashCount)
	}
	if stats.CurrentBranch != "main" {
		t.Errorf("CurrentBranch = %q, want %q", stats.CurrentBranch, "main")
	}
}

func TestRepoAgeIsFromTheOldestRootCommit(t *testing.T) {
	dir := gittest.InitRepo(t)
	at := func(daysAgo int) {
		date := time.Now().AddDate(0, 0, -daysAgo).Format(time.RFC3339)
		t.Setenv("GIT_AUTHOR_DATE", date)
		t.Setenv("GIT_COMMITTER_DATE", date)
	}
	at(20)
	gittest.CommitFile(t, dir, "a.txt", "v1", "first on main")
	at(2)
	gittest.CommitFile(t, dir, "a.txt", "v2", "newest on main")
	// A second history, merged in later, began earlier.
	at(30)
	gittest.Run(t, dir, "switch", "-q", "--orphan", "imported")
	gittest.CommitFile(t, dir, "b.txt", "v1", "first imported")
	gittest.Run(t, dir, "switch", "-q", "main")
	at(0)
	gittest.Run(t, dir, "merge", "-q", "--allow-unrelated-histories", "-m", "merge imported", "imported")
	// A stash of untracked files is a root commit too, and isn't history.
	at(100)
	gittest.WriteFile(t, dir, "untracked.txt", "x")
	gittest.Run(t, dir, "stash", "push", "-u")

	stats, err := ComputeRepoStats(context.Background(), dir)
	if err != nil {
		t.Fatalf("ComputeRepoStats: %v", err)
	}
	if stats.RepoAgeDays != 30 {
		t.Errorf("RepoAgeDays = %d, want 30", stats.RepoAgeDays)
	}
}
