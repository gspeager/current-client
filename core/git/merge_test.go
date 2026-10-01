package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestMergeBranchFastForward(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	gittest.Run(t, dir, "checkout", "-q", "-b", "feature")
	gittest.CommitFile(t, dir, "feature.txt", "v1", "feature commit")
	gittest.Run(t, dir, "checkout", "-q", "main")

	if err := MergeBranch(context.Background(), dir, "feature"); err != nil {
		t.Fatalf("MergeBranch: %v", err)
	}

	head := gittest.Run(t, dir, "rev-parse", "HEAD")
	featureHead := gittest.Run(t, dir, "rev-parse", "feature")
	if head != featureHead {
		t.Fatalf("HEAD = %q, want a fast-forward to feature's tip %q", head, featureHead)
	}
	if _, err := os.Stat(filepath.Join(dir, "feature.txt")); err != nil {
		t.Fatalf("feature.txt missing after fast-forward merge: %v", err)
	}
}

func TestMergeBranchCleanMergeCommit(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "base.txt", "v1", "initial")
	gittest.Run(t, dir, "checkout", "-q", "-b", "feature")
	gittest.CommitFile(t, dir, "feature.txt", "v1", "feature commit")
	gittest.Run(t, dir, "checkout", "-q", "main")
	gittest.CommitFile(t, dir, "main.txt", "v1", "main commit")

	if err := MergeBranch(context.Background(), dir, "feature"); err != nil {
		t.Fatalf("MergeBranch: %v", err)
	}

	parents := gittest.Run(t, dir, "log", "-1", "--pretty=%P")
	if len(parents) == 0 {
		t.Fatal("expected a merge commit with two parents")
	}
	for _, name := range []string{"base.txt", "feature.txt", "main.txt"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("%s missing after merge: %v", name, err)
		}
	}
}

func TestMergeBranchConflictLeavesConflictState(t *testing.T) {
	dir := initConflictRepo(t)

	if err := MergeBranch(context.Background(), dir, "theirs"); err == nil {
		t.Fatal("expected an error merging a conflicting branch")
	}

	state, err := DetectConflictState(context.Background(), dir)
	if err != nil {
		t.Fatalf("DetectConflictState: %v", err)
	}
	if state.Operation != ConflictMerge {
		t.Fatalf("Operation = %q, want %q", state.Operation, ConflictMerge)
	}
	if len(state.ConflictedPaths) != 1 || state.ConflictedPaths[0] != "file.txt" {
		t.Fatalf("ConflictedPaths = %v, want [file.txt]", state.ConflictedPaths)
	}
}
