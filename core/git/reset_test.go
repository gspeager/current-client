package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func initResetRepo(t *testing.T) (dir, v1SHA string) {
	t.Helper()
	dir = gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1\n", "v1")
	v1SHA = gittest.Run(t, dir, "rev-parse", "HEAD")
	gittest.CommitFile(t, dir, "file.txt", "v2\n", "v2")
	return dir, v1SHA
}

func TestResetSoftKeepsIndexAndWorkingTree(t *testing.T) {
	dir, v1SHA := initResetRepo(t)

	if err := Reset(context.Background(), dir, v1SHA, ResetSoft); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	if head := gittest.Run(t, dir, "rev-parse", "HEAD"); head != v1SHA {
		t.Fatalf("HEAD = %q, want %q", head, v1SHA)
	}
	// v2's change is still staged (index untouched by --soft).
	staged := gittest.Run(t, dir, "diff", "--cached", "--name-only")
	if staged != "file.txt" {
		t.Fatalf("staged files = %q, want file.txt still staged", staged)
	}
	content, err := os.ReadFile(filepath.Join(dir, "file.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "v2\n" {
		t.Fatalf("file.txt = %q, want the working tree untouched at v2", content)
	}
}

func TestResetMixedUnstagesButKeepsWorkingTree(t *testing.T) {
	dir, v1SHA := initResetRepo(t)

	if err := Reset(context.Background(), dir, v1SHA, ResetMixed); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	if head := gittest.Run(t, dir, "rev-parse", "HEAD"); head != v1SHA {
		t.Fatalf("HEAD = %q, want %q", head, v1SHA)
	}
	staged := gittest.Run(t, dir, "diff", "--cached", "--name-only")
	if staged != "" {
		t.Fatalf("staged files = %q, want nothing staged after --mixed", staged)
	}
	unstaged := gittest.Run(t, dir, "diff", "--name-only")
	if unstaged != "file.txt" {
		t.Fatalf("unstaged files = %q, want file.txt unstaged-modified", unstaged)
	}
	content, err := os.ReadFile(filepath.Join(dir, "file.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "v2\n" {
		t.Fatalf("file.txt = %q, want the working tree untouched at v2", content)
	}
}

func TestResetHardDiscardsWorkingTree(t *testing.T) {
	dir, v1SHA := initResetRepo(t)

	if err := Reset(context.Background(), dir, v1SHA, ResetHard); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	if head := gittest.Run(t, dir, "rev-parse", "HEAD"); head != v1SHA {
		t.Fatalf("HEAD = %q, want %q", head, v1SHA)
	}
	statuses, err := GetStatus(context.Background(), dir)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if len(statuses) != 0 {
		t.Fatalf("got %+v, want a clean working tree after --hard", statuses)
	}
	content, err := os.ReadFile(filepath.Join(dir, "file.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "v1\n" {
		t.Fatalf("file.txt = %q, want the working tree reverted to v1", content)
	}
}
