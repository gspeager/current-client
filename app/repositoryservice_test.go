package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gspeager/current-client/core/repository"
	"github.com/gspeager/current-client/internal/config"
)

func TestRemoveRecentRepositoryDeletesTheAppsRefs(t *testing.T) {
	useTempSettings(t)
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	gitIn(t, repo, "-c", "user.name=X", "-c", "user.email=x@example.com", "commit", "-q", "--allow-empty", "-m", "initial")
	gitIn(t, repo, "update-ref", "refs/current-client/hosts/origin/mr/1", "HEAD")
	gitIn(t, repo, "update-ref", "refs/current-client/snapshots/1", "HEAD")
	if err := updateConfig(func(cfg *config.Config) { cfg.RecentRepositories = []string{repo} }); err != nil {
		t.Fatal(err)
	}

	var s RepositoryService
	if err := s.RemoveRecentRepository(repo); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(mustOutput(t, repo, "for-each-ref", "--format=%(refname)", "refs/current-client"))); got != "" {
		t.Fatalf("refs left = %q", got)
	}
	if err := s.RemoveRecentRepository(filepath.Join(repo, "gone")); err != nil {
		t.Fatalf("a folder that's gone should still be forgotten: %v", err)
	}
	if cfg := configOrDefault(); len(cfg.RecentRepositories) != 0 {
		t.Fatalf("recent = %v", cfg.RecentRepositories)
	}
}

func repoWithFetchedHead(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	gitIn(t, repo, "-c", "user.name=X", "-c", "user.email=x@example.com", "commit", "-q", "--allow-empty", "-m", "initial")
	gitIn(t, repo, "update-ref", "refs/current-client/hosts/origin/mr/1", "HEAD")
	return repo
}

func TestRemovingARepositoryOpenInATabKeepsItsHeads(t *testing.T) {
	useTempSettings(t)
	repo := repoWithFetchedHead(t)
	if err := updateConfig(func(cfg *config.Config) {
		cfg.RecentRepositories = []string{repo}
		cfg.OpenTabs = []string{repo}
	}); err != nil {
		t.Fatal(err)
	}

	var s RepositoryService
	if err := s.RemoveRecentRepository(repo); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(mustOutput(t, repo, "for-each-ref", "refs/current-client"))); got == "" {
		t.Fatal("the open repository lost its fetched heads")
	}
}

func TestDroppingOffTheRecentListDeletesFetchedHeads(t *testing.T) {
	useTempSettings(t)
	dropped := repoWithFetchedHead(t)
	recent := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", dropped}
	if err := updateConfig(func(cfg *config.Config) { cfg.RecentRepositories = recent }); err != nil {
		t.Fatal(err)
	}

	if _, err := opened(&repository.Repository{Path: t.TempDir()}, nil); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for strings.TrimSpace(string(mustOutput(t, dropped, "for-each-ref", "refs/current-client"))) != "" {
		if time.Now().After(deadline) {
			t.Fatal("the dropped repository kept its fetched heads")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestCloneRepositoryCreatesFolderInsideParent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APPDATA", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("HOME", home)

	source := filepath.Join(t.TempDir(), "widgets")
	git := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "user.name=T", "-c", "user.email=t@example.com"}, args...)...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git(".", "init", "-q", source)
	git(source, "commit", "-q", "--allow-empty", "-m", "first")

	parent := t.TempDir()
	if err := os.WriteFile(filepath.Join(parent, "unrelated.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := (&RepositoryService{}).CloneRepository(context.Background(), source, parent, "widgets", nil)
	if err != nil {
		t.Fatalf("CloneRepository: %v", err)
	}
	if want := filepath.Join(parent, "widgets"); got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(got, ".git")); err != nil {
		t.Errorf("clone has no .git: %v", err)
	}
	if _, err := os.Stat(filepath.Join(parent, ".git")); !os.IsNotExist(err) {
		t.Errorf("parent folder became a repository: %v", err)
	}

	if _, err := (&RepositoryService{}).CloneRepository(context.Background(), source, parent, "widgets", nil); err == nil {
		t.Error("expected cloning into an existing, non-empty folder to fail")
	}
}

func TestCloneRepositoryRejectsFolderNamesThatLeaveParent(t *testing.T) {
	for _, name := range []string{"", ".", "..", "a/b", `a\b`} {
		if _, err := (&RepositoryService{}).CloneRepository(context.Background(), "https://example.invalid/r.git", t.TempDir(), name, nil); err == nil {
			t.Errorf("folder name %q: expected an error", name)
		}
	}
}
