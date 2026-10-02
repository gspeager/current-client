package repository

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gspeager/current-client/core/gitexec"
	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestOpenValidRepository(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, "", "init", dir)

	repo, err := Open(context.Background(), dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if repo.Path != dir {
		t.Fatalf("Path = %q, want %q", repo.Path, dir)
	}
}

func TestOpenRejectsNonRepositoryFolder(t *testing.T) {
	dir := t.TempDir()

	_, err := Open(context.Background(), dir)
	assertAppError(t, err)
}

func TestOpenRejectsMissingPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	_, err := Open(context.Background(), missing)
	assertAppError(t, err)
}

func TestInitCreatesRepository(t *testing.T) {
	dir := t.TempDir()

	repo, err := Init(context.Background(), dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if repo.Path != dir {
		t.Fatalf("Path = %q, want %q", repo.Path, dir)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		t.Fatalf(".git not found after Init: %v", err)
	}
}

func TestInitRejectsPathThatIsAFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-folder")
	if err := os.WriteFile(file, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := Init(context.Background(), file)
	assertAppError(t, err)
}

func TestCloneClonesRepository(t *testing.T) {
	bareDir := newBareRepoWithCommit(t)
	dest := filepath.Join(t.TempDir(), "clone-dest")

	repo, err := Clone(context.Background(), bareDir, dest)
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if repo.Path != dest {
		t.Fatalf("Path = %q, want %q", repo.Path, dest)
	}
	if _, err := os.Stat(filepath.Join(dest, "README.md")); err != nil {
		t.Fatalf("expected README.md to exist after clone: %v", err)
	}
}

func TestCloneRejectsInvalidSource(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "clone-dest")

	_, err := Clone(context.Background(), filepath.Join(t.TempDir(), "no-such-source"), dest)
	assertAppError(t, err)
}

func TestOpenWithCancelledContextReturnsFriendlyError(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, "", "init", dir)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Open(ctx, dir)
	assertAppError(t, err)
}

func TestInitWithCancelledContextReturnsFriendlyError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Init(ctx, t.TempDir())
	assertAppError(t, err)
}

func TestCloneWithCancelledContextReturnsFriendlyError(t *testing.T) {
	bareDir := newBareRepoWithCommit(t)
	dest := filepath.Join(t.TempDir(), "clone-dest")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Clone(ctx, bareDir, dest)
	assertAppError(t, err)
}

// newBareRepoWithCommit stands in for a remote without touching the network.
func newBareRepoWithCommit(t *testing.T) string {
	t.Helper()

	source := t.TempDir()
	gittest.Run(t, source, "init")
	gittest.Run(t, source, "config", "user.email", "test@example.com")
	gittest.Run(t, source, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	gittest.Run(t, source, "add", "README.md")
	gittest.Run(t, source, "commit", "-m", "initial commit")

	bareDir := filepath.Join(t.TempDir(), "bare.git")
	gittest.Run(t, "", "clone", "--bare", source, bareDir)
	return bareDir
}

func assertAppError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error")
	}
	var appErr *gitexec.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected a *gitexec.AppError, got %T: %v", err, err)
	}
	if appErr.Message == "" {
		t.Fatal("expected a friendly message")
	}
}

func TestOpenSubfolderOpensTheWholeRepository(t *testing.T) {
	// Resolved, since Git reports the real top-level path: macOS's temp dir is
	// under /var, a symlink to /private/var.
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	gittest.Run(t, "", "init", dir)
	sub := filepath.Join(dir, "src", "app")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	repo, err := Open(context.Background(), sub)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if repo.Path != dir {
		t.Fatalf("Path = %q, want the top of the working tree %q", repo.Path, dir)
	}
}

func TestOpenRejectsBareRepository(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, "", "init", "--bare", dir)

	_, err := Open(context.Background(), dir)
	var appErr *gitexec.AppError
	if !errors.As(err, &appErr) || appErr.Message != "This is a bare repository, which has no working files. Open a clone of it instead." {
		t.Fatalf("err = %v, want the bare repository message", err)
	}
}

func TestOpenRejectsGitDirectory(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, "", "init", dir)

	_, err := Open(context.Background(), filepath.Join(dir, ".git"))
	var appErr *gitexec.AppError
	if !errors.As(err, &appErr) || appErr.Message != "This folder isn't part of a repository's working files." {
		t.Fatalf("err = %v, want the not-a-working-tree message", err)
	}
}

func TestOpenPathWithSpacesAndNonASCII(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my repo ñé 日本")
	gittest.Run(t, "", "init", dir)

	repo, err := Open(context.Background(), dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if repo.Path != dir {
		t.Fatalf("Path = %q, want %q", repo.Path, dir)
	}
}

func TestOpenSubmoduleOpensTheSubmodule(t *testing.T) {
	lib := t.TempDir()
	gittest.Run(t, "", "init", "-b", "main", lib)
	gittest.Run(t, lib, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "--allow-empty", "-m", "lib")
	parent := t.TempDir()
	gittest.Run(t, "", "init", "-b", "main", parent)
	// Local file:// submodules are blocked by default since Git 2.38.1.
	gittest.Run(t, parent, "-c", "protocol.file.allow=always", "submodule", "add", "file://"+filepath.ToSlash(lib), "lib")

	repo, err := Open(context.Background(), filepath.Join(parent, "lib"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if repo.Path != filepath.Join(parent, "lib") {
		t.Fatalf("Path = %q, want the submodule's own folder", repo.Path)
	}
}
