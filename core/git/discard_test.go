package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestDiscardFileRevertsUnstagedModification(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	gittest.CommitFile(t, dir, "f.txt", "committed", "initial")
	gittest.WriteFile(t, dir, "f.txt", "edited")

	if err := DiscardFile(context.Background(), dir, "f.txt"); err != nil {
		t.Fatalf("DiscardFile: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "f.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "committed" {
		t.Fatalf("content = %q, want %q", got, "committed")
	}
}

func TestDiscardFileDeletesUntrackedFile(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.WriteFile(t, dir, "new.txt", "hello")

	if err := DiscardFile(context.Background(), dir, "new.txt"); err != nil {
		t.Fatalf("DiscardFile: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "new.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected new.txt to be gone, stat err = %v", err)
	}
}

func TestDiscardFilesHandlesMixedTrackedAndUntrackedInOneCall(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	gittest.CommitFile(t, dir, "tracked.txt", "committed", "initial")
	gittest.WriteFile(t, dir, "tracked.txt", "edited")
	gittest.WriteFile(t, dir, "new.txt", "hello")

	if err := DiscardFiles(context.Background(), dir, []string{"tracked.txt", "new.txt"}); err != nil {
		t.Fatalf("DiscardFiles: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "tracked.txt"))
	if err != nil {
		t.Fatalf("ReadFile tracked.txt: %v", err)
	}
	if string(got) != "committed" {
		t.Fatalf("tracked.txt content = %q, want %q", got, "committed")
	}
	if _, err := os.Stat(filepath.Join(dir, "new.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected new.txt to be gone, stat err = %v", err)
	}
}

func TestDiscardFilesEmptyIsNoop(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	if err := DiscardFiles(context.Background(), dir, nil); err != nil {
		t.Fatalf("DiscardFiles(nil): %v", err)
	}
}

func TestDiscardFilePreservesStagedChanges(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	gittest.CommitFile(t, dir, "f.txt", "committed", "initial")

	gittest.WriteFile(t, dir, "f.txt", "staged version")
	gittest.Run(t, dir, "add", "f.txt")
	gittest.WriteFile(t, dir, "f.txt", "further unstaged edit")

	if err := DiscardFile(context.Background(), dir, "f.txt"); err != nil {
		t.Fatalf("DiscardFile: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "f.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "staged version" {
		t.Fatalf("content = %q, want the still-staged %q (discard should only revert the working tree)", got, "staged version")
	}
}
