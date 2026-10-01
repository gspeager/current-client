package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestPullUpdatesHeadAndWorkingTree(t *testing.T) {
	remoteDir := t.TempDir() + "/remote.git"
	gittest.Run(t, "", "init", "--bare", "-b", "main", remoteDir)

	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "remote", "add", "origin", remoteDir)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	gittest.Run(t, dir, "push", "-u", "origin", "main")

	clone := t.TempDir()
	gittest.Run(t, "", "clone", "-q", remoteDir, clone)
	gittest.Run(t, clone, "config", "user.email", "test@example.com")
	gittest.Run(t, clone, "config", "user.name", "Test")
	gittest.WriteFile(t, clone, "file.txt", "v2")
	gittest.Run(t, clone, "add", "file.txt")
	gittest.Run(t, clone, "commit", "-m", "from clone")
	gittest.Run(t, clone, "push", "-q", "origin", "main")
	newSHA := gittest.Run(t, clone, "rev-parse", "HEAD")

	if err := Pull(context.Background(), dir); err != nil {
		t.Fatalf("Pull: %v", err)
	}

	head := gittest.Run(t, dir, "rev-parse", "HEAD")
	if head != newSHA {
		t.Fatalf("HEAD = %q after pull, want %q", head, newSHA)
	}
	content, err := os.ReadFile(filepath.Join(dir, "file.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "v2" {
		t.Fatalf("working tree content = %q, want %q", content, "v2")
	}
}

func TestPullRejectsNoUpstream(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")

	if err := Pull(context.Background(), dir); err == nil {
		t.Fatal("expected an error pulling with no upstream configured")
	}
}
