package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

func TestMergeBranchModeNoFastForwardRecordsMergeCommit(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	gittest.Run(t, dir, "checkout", "-q", "-b", "feature")
	gittest.CommitFile(t, dir, "feature.txt", "v1", "feature commit")
	gittest.Run(t, dir, "checkout", "-q", "main")

	if err := MergeBranchMode(context.Background(), dir, "feature", MergeCommit); err != nil {
		t.Fatalf("MergeBranchMode: %v", err)
	}

	parents := strings.Fields(gittest.Run(t, dir, "log", "-1", "--pretty=%P"))
	featureHead := gittest.Run(t, dir, "rev-parse", "feature")
	if len(parents) != 2 || parents[1] != featureHead {
		t.Fatalf("HEAD parents = %v, want a merge commit whose second parent is feature %q", parents, featureHead)
	}
}

func TestMergeBranchModeSquashStagesWithoutCommitting(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	gittest.Run(t, dir, "checkout", "-q", "-b", "feature")
	gittest.CommitFile(t, dir, "a.txt", "a", "first")
	gittest.CommitFile(t, dir, "b.txt", "b", "second")
	gittest.Run(t, dir, "checkout", "-q", "main")
	before := gittest.Run(t, dir, "rev-parse", "HEAD")

	if err := MergeBranchMode(context.Background(), dir, "feature", MergeSquash); err != nil {
		t.Fatalf("MergeBranchMode: %v", err)
	}

	if after := gittest.Run(t, dir, "rev-parse", "HEAD"); after != before {
		t.Fatalf("HEAD moved to %q; a squash merge must leave the commit to the user", after)
	}
	if staged := gittest.Run(t, dir, "diff", "--cached", "--name-only"); staged != "a.txt\nb.txt" {
		t.Fatalf("staged = %q, want both of feature's files", staged)
	}
}
