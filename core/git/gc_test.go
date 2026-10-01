package git

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestComputeRepoDiskUsage(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "a.txt", "v1", "initial")

	usage, err := ComputeRepoDiskUsage(context.Background(), dir)
	if err != nil {
		t.Fatalf("ComputeRepoDiskUsage: %v", err)
	}
	if usage.LooseObjectCount == 0 {
		t.Error("LooseObjectCount = 0, want at least the objects from one commit")
	}
	if usage.TotalSizeKB != usage.LooseSizeKB+usage.PackedSizeKB {
		t.Errorf("TotalSizeKB = %d, want LooseSizeKB(%d) + PackedSizeKB(%d)", usage.TotalSizeKB, usage.LooseSizeKB, usage.PackedSizeKB)
	}
}

func TestFsckCheckCleanRepoHasNoIssues(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "a.txt", "v1", "initial")

	issues, err := FsckCheck(context.Background(), dir)
	if err != nil {
		t.Fatalf("FsckCheck: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("issues = %v, want none for a healthy repo", issues)
	}
}

func TestFsckCheckDetectsCorruptedObject(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "a.txt", "v1", "initial")

	blobSHA := gittest.Run(t, dir, "rev-parse", "HEAD:a.txt")
	objPath := filepath.Join(dir, ".git", "objects", blobSHA[:2], blobSHA[2:])
	// Git writes loose objects read-only, which Windows enforces on overwrite.
	if err := os.Chmod(objPath, 0o644); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	if err := os.WriteFile(objPath, []byte("not a valid git object"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	issues, err := FsckCheck(context.Background(), dir)
	if err != nil {
		t.Fatalf("FsckCheck: %v", err)
	}
	if len(issues) == 0 {
		t.Fatal("expected fsck to report the corrupted object, got no issues")
	}
}

func TestRunGCReducesLooseObjectCount(t *testing.T) {
	dir := gittest.InitRepo(t)
	for i := 0; i < 5; i++ {
		gittest.CommitFile(t, dir, "a.txt", strconv.Itoa(i), "commit "+strconv.Itoa(i))
	}

	before := looseObjects(t, dir)
	if before == 0 {
		t.Fatal("expected some loose objects before gc")
	}

	if err := RunGC(context.Background(), dir); err != nil {
		t.Fatalf("RunGC: %v", err)
	}

	after := looseObjects(t, dir)
	if after >= before {
		t.Fatalf("loose object count after gc = %d, want less than before (%d)", after, before)
	}
}

func looseObjects(t *testing.T, dir string) int {
	t.Helper()
	usage, err := ComputeRepoDiskUsage(context.Background(), dir)
	if err != nil {
		t.Fatalf("ComputeRepoDiskUsage: %v", err)
	}
	return usage.LooseObjectCount
}
