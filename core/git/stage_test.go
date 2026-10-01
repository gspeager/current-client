package git

import (
	"context"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestStageFileStagesUntrackedFile(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	gittest.WriteFile(t, dir, "new.txt", "hello")

	if err := StageFile(context.Background(), dir, "new.txt"); err != nil {
		t.Fatalf("StageFile: %v", err)
	}

	statuses, err := GetStatus(context.Background(), dir)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if len(statuses) != 1 || statuses[0].Path != "new.txt" || statuses[0].IndexStatus != 'A' {
		t.Fatalf("got %+v, want a single staged addition of new.txt", statuses)
	}
}

func TestUnstageFileUnstagesFile(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	gittest.WriteFile(t, dir, "new.txt", "hello")
	gittest.Run(t, dir, "add", "new.txt")

	if err := UnstageFile(context.Background(), dir, "new.txt"); err != nil {
		t.Fatalf("UnstageFile: %v", err)
	}

	statuses, err := GetStatus(context.Background(), dir)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if len(statuses) != 1 || statuses[0].Path != "new.txt" || statuses[0].IndexStatus != '?' {
		t.Fatalf("got %+v, want new.txt back to untracked", statuses)
	}
}

func TestStageFileRejectsMissingPath(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")

	err := StageFile(context.Background(), dir, "does-not-exist.txt")
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestStageFilesStagesEveryPathInOneCall(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	gittest.WriteFile(t, dir, "a.txt", "a")
	gittest.WriteFile(t, dir, "b.txt", "b")

	if err := StageFiles(context.Background(), dir, []string{"a.txt", "b.txt"}); err != nil {
		t.Fatalf("StageFiles: %v", err)
	}

	statuses, err := GetStatus(context.Background(), dir)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if len(statuses) != 2 || statuses[0].IndexStatus != 'A' || statuses[1].IndexStatus != 'A' {
		t.Fatalf("got %+v, want both files staged as additions", statuses)
	}
}

func TestUnstageFilesUnstagesEveryPathInOneCall(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	gittest.WriteFile(t, dir, "a.txt", "a")
	gittest.WriteFile(t, dir, "b.txt", "b")
	gittest.Run(t, dir, "add", "a.txt", "b.txt")

	if err := UnstageFiles(context.Background(), dir, []string{"a.txt", "b.txt"}); err != nil {
		t.Fatalf("UnstageFiles: %v", err)
	}

	statuses, err := GetStatus(context.Background(), dir)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if len(statuses) != 2 || statuses[0].IndexStatus != '?' || statuses[1].IndexStatus != '?' {
		t.Fatalf("got %+v, want both files back to untracked", statuses)
	}
}

func TestStageFilesNoopOnEmptyPaths(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")

	if err := StageFiles(context.Background(), dir, nil); err != nil {
		t.Fatalf("StageFiles with no paths should not error: %v", err)
	}
}
