package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

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

	got, err := (&RepositoryService{}).CloneRepository(context.Background(), source, parent, "widgets")
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

	if _, err := (&RepositoryService{}).CloneRepository(context.Background(), source, parent, "widgets"); err == nil {
		t.Error("expected cloning into an existing, non-empty folder to fail")
	}
}

func TestCloneRepositoryRejectsFolderNamesThatLeaveParent(t *testing.T) {
	for _, name := range []string{"", ".", "..", "a/b", `a\b`} {
		if _, err := (&RepositoryService{}).CloneRepository(context.Background(), "https://example.invalid/r.git", t.TempDir(), name); err == nil {
			t.Errorf("folder name %q: expected an error", name)
		}
	}
}
