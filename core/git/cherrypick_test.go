package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestCherryPickAppliesCleanCommit(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "base.txt", "v1", "base")
	gittest.Run(t, dir, "checkout", "-q", "-b", "feature")
	gittest.CommitFile(t, dir, "feature.txt", "v1", "feature commit")
	featureSHA := gittest.Run(t, dir, "rev-parse", "feature")
	gittest.Run(t, dir, "checkout", "-q", "main")

	if err := CherryPick(context.Background(), dir, featureSHA); err != nil {
		t.Fatalf("CherryPick: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "feature.txt")); err != nil {
		t.Fatalf("feature.txt missing after cherry-pick: %v", err)
	}
	subject := gittest.Run(t, dir, "log", "-1", "--pretty=%s")
	if subject != "feature commit" {
		t.Fatalf("last commit subject = %q, want %q", subject, "feature commit")
	}
}

func TestCherryPickConflictLeavesConflictState(t *testing.T) {
	dir := initConflictRepo(t)
	theirsSHA := gittest.Run(t, dir, "rev-parse", "theirs")

	if err := CherryPick(context.Background(), dir, theirsSHA); err == nil {
		t.Fatal("expected an error cherry-picking a conflicting commit")
	}

	state, err := DetectConflictState(context.Background(), dir)
	if err != nil {
		t.Fatalf("DetectConflictState: %v", err)
	}
	if state.Operation != ConflictCherryPick {
		t.Fatalf("Operation = %q, want %q", state.Operation, ConflictCherryPick)
	}
}

func TestRevertAppliesCleanCommit(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "add file")
	toRevert := gittest.Run(t, dir, "rev-parse", "HEAD")

	if err := Revert(context.Background(), dir, toRevert); err != nil {
		t.Fatalf("Revert: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "file.txt")); !os.IsNotExist(err) {
		t.Fatalf("file.txt still present after reverting its add commit")
	}
	subject := gittest.Run(t, dir, "log", "-1", "--pretty=%s")
	if subject != `Revert "add file"` {
		t.Fatalf("last commit subject = %q, want a Revert commit", subject)
	}
}

func TestRevertConflictLeavesConflictState(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1\n", "v1")
	gittest.CommitFile(t, dir, "file.txt", "v2\n", "v2")
	gittest.CommitFile(t, dir, "file.txt", "v3\n", "v3")
	v2SHA := gittest.Run(t, dir, "rev-parse", "HEAD~1")

	if err := Revert(context.Background(), dir, v2SHA); err == nil {
		t.Fatal("expected an error reverting a commit that conflicts with a later edit")
	}

	state, err := DetectConflictState(context.Background(), dir)
	if err != nil {
		t.Fatalf("DetectConflictState: %v", err)
	}
	if state.Operation != ConflictRevert {
		t.Fatalf("Operation = %q, want %q", state.Operation, ConflictRevert)
	}
}
