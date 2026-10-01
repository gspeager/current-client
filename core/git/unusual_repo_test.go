package git

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestEmptyRepositoryReadsAsEmptyRatherThanFailing(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.WriteFile(t, dir, "new.txt", "not committed yet")
	ctx := context.Background()

	if history, err := History(ctx, dir, 50, 0, HistoryFilter{}); err != nil || len(history) != 0 {
		t.Errorf("History = %v, %v; want empty, nil", history, err)
	}
	if found, err := SearchHistory(ctx, dir, "anything", 50); err != nil || len(found) != 0 {
		t.Errorf("SearchHistory = %v, %v; want empty, nil", found, err)
	}
	if history, err := FileHistory(ctx, dir, "new.txt", 50, 0); err != nil || len(history) != 0 {
		t.Errorf("FileHistory = %v, %v; want empty, nil", history, err)
	}
	if reflog, err := Reflog(ctx, dir, 50); err != nil || len(reflog) != 0 {
		t.Errorf("Reflog = %v, %v; want empty, nil", reflog, err)
	}
	if contributors, err := Contributors(ctx, dir, "", ""); err != nil || len(contributors) != 0 {
		t.Errorf("Contributors = %v, %v; want empty, nil", contributors, err)
	}
	status, err := CurrentBranchStatus(ctx, dir)
	if err != nil || status.Current != "main" {
		t.Errorf("CurrentBranchStatus = %+v, %v; want main, nil", status, err)
	}
	stats, err := ComputeRepoStats(ctx, dir)
	if err != nil || stats.CurrentBranch != "main" || stats.TotalCommits != 0 {
		t.Errorf("ComputeRepoStats = %+v, %v; want main with 0 commits", stats, err)
	}
}

func TestDetachedHeadReportsHEADAsTheBranch(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "f.txt", "1", "first")
	gittest.CommitFile(t, dir, "f.txt", "2", "second")
	gittest.Run(t, dir, "checkout", "-q", "--detach", "HEAD~1")
	ctx := context.Background()

	status, err := CurrentBranchStatus(ctx, dir)
	if err != nil || status.Current != "HEAD" {
		t.Fatalf("CurrentBranchStatus = %+v, %v; want HEAD", status, err)
	}
	history, err := History(ctx, dir, 50, 0, HistoryFilter{})
	if err != nil || len(history) != 1 || history[0].Subject != "first" {
		t.Fatalf("History = %+v, %v; want only the detached commit", history, err)
	}
}

func TestShallowCloneShowsOnlyItsHistory(t *testing.T) {
	source := gittest.InitRepo(t)
	for _, msg := range []string{"one", "two", "three"} {
		gittest.CommitFile(t, source, "f.txt", msg, msg)
	}
	clone := filepath.Join(t.TempDir(), "shallow")
	gittest.Run(t, "", "clone", "-q", "--depth", "1", "file://"+filepath.ToSlash(source), clone)
	ctx := context.Background()

	history, err := History(ctx, clone, 50, 0, HistoryFilter{})
	if err != nil || len(history) != 1 || history[0].Subject != "three" {
		t.Fatalf("History = %+v, %v; want just the shallow tip", history, err)
	}
	stats, err := ComputeRepoStats(ctx, clone)
	if err != nil || stats.TotalCommits != 1 {
		t.Fatalf("ComputeRepoStats = %+v, %v; want 1 commit", stats, err)
	}
}
