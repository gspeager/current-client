package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/snapshot"
	"github.com/gspeager/current-client/internal/config"
)

func recoverRepo(t *testing.T) string {
	t.Helper()
	useTempSettings(t)
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	gitIn(t, repo, "config", "user.name", "T")
	gitIn(t, repo, "config", "user.email", "t@example.com")
	write(t, repo, "a.txt", "committed\n")
	gitIn(t, repo, "add", "a.txt")
	gitIn(t, repo, "commit", "-q", "-m", "first")
	return repo
}

func write(t *testing.T, repo, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func snapshots(t *testing.T, repo string) []SnapshotInfo {
	t.Helper()
	list, err := new(RecoverService).List(repo)
	if err != nil {
		t.Fatal(err)
	}
	return list
}

func TestDestructiveActionsTakeASnapshotFirst(t *testing.T) {
	repo := recoverRepo(t)
	actions := []struct {
		name string
		run  func() error
	}{
		{"discard", func() error { return new(StatusService).DiscardFile(repo, "a.txt") }},
		{"discard", func() error { return new(StatusService).DiscardFiles(repo, []string{"a.txt"}) }},
		{"reset", func() error { return new(CommitService).Reset(repo, "HEAD", "hard") }},
		{"stash", func() error { return new(StashService).StashSave(repo, StashOptions{}) }},
	}
	for i, action := range actions {
		write(t, repo, "a.txt", "edit "+string(rune('a'+i))+"\n")
		if err := action.run(); err != nil {
			t.Fatalf("%s: %v", action.name, err)
		}
		list := snapshots(t, repo)
		if len(list) != 1 || list[0].Action != action.name || list[0].Kind != "tree" {
			t.Fatalf("after %s: %+v", action.name, list)
		}
		if err := new(RecoverService).Restore(repo, list[0].ID, []string{"a.txt"}); err != nil {
			t.Fatal(err)
		}
		if got, _ := os.ReadFile(filepath.Join(repo, "a.txt")); !strings.HasPrefix(string(got), "edit") {
			t.Fatalf("restored %q", got)
		}
		gitIn(t, repo, "checkout", "--", "a.txt")
		if err := snapshot.DeleteAll(context.Background(), repo); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSoftResetAndCleanCheckoutTakeNoSnapshot(t *testing.T) {
	repo := recoverRepo(t)
	gitIn(t, repo, "branch", "side")
	if err := new(BranchService).CheckoutBranch(repo, "side"); err != nil {
		t.Fatal(err)
	}
	write(t, repo, "a.txt", "edit\n")
	if err := new(CommitService).Reset(repo, "HEAD", "mixed"); err != nil {
		t.Fatal(err)
	}
	if list := snapshots(t, repo); len(list) != 0 {
		t.Fatalf("snapshots = %+v", list)
	}
	if err := new(BranchService).CheckoutBranch(repo, "main"); err != nil {
		t.Fatal(err)
	}
	if list := snapshots(t, repo); len(list) != 1 || list[0].Action != "checkout" {
		t.Fatalf("a dirty checkout = %+v", list)
	}
}

func TestDeletingAnUnmergedBranchKeepsItsCommits(t *testing.T) {
	repo := recoverRepo(t)
	gitIn(t, repo, "switch", "-q", "-c", "unmerged")
	gitIn(t, repo, "commit", "-q", "--allow-empty", "-m", "only here")
	tip := strings.TrimSpace(string(mustOutput(t, repo, "rev-parse", "HEAD")))
	gitIn(t, repo, "switch", "-q", "main")
	gitIn(t, repo, "branch", "merged")

	var s BranchService
	if _, err := s.DeleteBranch(repo, "merged", false); err != nil {
		t.Fatal(err)
	}
	if list := snapshots(t, repo); len(list) != 0 {
		t.Fatalf("a branch held elsewhere needs no backup: %+v", list)
	}
	if _, err := s.DeleteBranch(repo, "unmerged", true); err != nil {
		t.Fatal(err)
	}
	list := snapshots(t, repo)
	if len(list) != 1 || list[0].Kind != "tip" || list[0].Tip != tip || list[0].Branch != "unmerged" {
		t.Fatalf("snapshots = %+v", list)
	}
}

func TestRemovingALinkedWorktreeKeepsTheSharedSnapshots(t *testing.T) {
	repo := recoverRepo(t)
	linked := filepath.Join(t.TempDir(), "linked")
	gitIn(t, repo, "worktree", "add", "-q", "-b", "side", linked)
	write(t, repo, "a.txt", "edit\n")
	snapshotBefore(repo, "discard")
	if err := new(RepositoryService).RemoveRecentRepository(linked); err != nil {
		t.Fatal(err)
	}
	if list := snapshots(t, repo); len(list) != 1 {
		t.Fatalf("snapshots = %+v", list)
	}
}

func TestSnapshotLimitAndRemoval(t *testing.T) {
	repo := recoverRepo(t)
	if err := updateConfig(func(cfg *config.Config) { cfg.SnapshotLimit = 1; cfg.RecentRepositories = []string{repo} }); err != nil {
		t.Fatal(err)
	}
	for _, content := range []string{"one\n", "two\n"} {
		write(t, repo, "a.txt", content)
		snapshotBefore(repo, "discard")
	}
	if list := snapshots(t, repo); len(list) != 1 {
		t.Fatalf("snapshots = %+v", list)
	}
	if err := new(RepositoryService).RemoveRecentRepository(repo); err != nil {
		t.Fatal(err)
	}
	if out := strings.TrimSpace(string(mustOutput(t, repo, "for-each-ref", "refs/current-client"))); out != "" {
		t.Fatalf("removal left refs: %s", out)
	}
}
