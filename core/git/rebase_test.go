package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestRebaseOntoLinearReplay(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "base.txt", "v1", "initial")
	gittest.Run(t, dir, "checkout", "-q", "-b", "feature")
	gittest.CommitFile(t, dir, "feature.txt", "v1", "feature commit")
	gittest.Run(t, dir, "checkout", "-q", "main")
	gittest.CommitFile(t, dir, "main.txt", "v1", "main commit")
	gittest.Run(t, dir, "checkout", "-q", "feature")

	if err := RebaseOnto(context.Background(), dir, "main"); err != nil {
		t.Fatalf("RebaseOnto: %v", err)
	}

	mainHead := gittest.Run(t, dir, "rev-parse", "main")
	newParent := gittest.Run(t, dir, "log", "-1", "--pretty=%P")
	if newParent != mainHead {
		t.Fatalf("feature's new parent = %q, want a single-parent replay onto main's tip %q", newParent, mainHead)
	}
	for _, name := range []string{"base.txt", "main.txt", "feature.txt"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("%s missing after rebase: %v", name, err)
		}
	}
}

func TestRebaseOntoConflictLeavesConflictState(t *testing.T) {
	dir := initConflictRepo(t)

	if err := RebaseOnto(context.Background(), dir, "theirs"); err == nil {
		t.Fatal("expected an error rebasing onto a conflicting branch")
	}

	state, err := DetectConflictState(context.Background(), dir)
	if err != nil {
		t.Fatalf("DetectConflictState: %v", err)
	}
	if state.Operation != ConflictRebase {
		t.Fatalf("Operation = %q, want %q", state.Operation, ConflictRebase)
	}
	if len(state.ConflictedPaths) != 1 || state.ConflictedPaths[0] != "file.txt" {
		t.Fatalf("ConflictedPaths = %v, want [file.txt]", state.ConflictedPaths)
	}
}
