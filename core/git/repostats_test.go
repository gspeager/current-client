package git

import (
	"context"
	"testing"

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
