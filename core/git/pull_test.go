package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

func TestPullRebaseReplaysLocalCommitsOnUpstream(t *testing.T) {
	remoteDir := t.TempDir() + "/remote.git"
	gittest.Run(t, "", "init", "--bare", "-b", "main", remoteDir)

	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "remote", "add", "origin", remoteDir)
	// A merge-only setting must not win over an explicit --rebase.
	gittest.Run(t, dir, "config", "pull.rebase", "false")
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	gittest.Run(t, dir, "push", "-u", "origin", "main")

	clone := t.TempDir()
	gittest.Run(t, "", "clone", "-q", remoteDir, clone)
	gittest.Run(t, clone, "config", "user.email", "test@example.com")
	gittest.Run(t, clone, "config", "user.name", "Test")
	gittest.WriteFile(t, clone, "theirs.txt", "theirs")
	gittest.Run(t, clone, "add", "theirs.txt")
	gittest.Run(t, clone, "commit", "-m", "from clone")
	gittest.Run(t, clone, "push", "-q", "origin", "main")
	upstreamSHA := gittest.Run(t, clone, "rev-parse", "HEAD")

	gittest.CommitFile(t, dir, "mine.txt", "mine", "local work")

	if err := PullRebase(context.Background(), dir); err != nil {
		t.Fatalf("PullRebase: %v", err)
	}

	if parents := gittest.Run(t, dir, "rev-list", "--parents", "-n", "1", "HEAD"); len(strings.Fields(parents)) != 2 {
		t.Fatalf("HEAD has parents %q, want a single parent (no merge commit)", parents)
	}
	if parent := gittest.Run(t, dir, "rev-parse", "HEAD^"); parent != upstreamSHA {
		t.Fatalf("HEAD^ = %q, want the upstream commit %q", parent, upstreamSHA)
	}
	if subject := gittest.Run(t, dir, "log", "-1", "--format=%s"); subject != "local work" {
		t.Fatalf("HEAD subject = %q, want the local commit on top", subject)
	}
}
