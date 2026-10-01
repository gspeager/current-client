package undo

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/internal/gittest"
)

// commitFile returns the new commit.
func commitFile(t *testing.T, dir, name, content, message string) string {
	t.Helper()
	gittest.CommitFile(t, dir, name, content, message)
	return head(t, dir)
}

func head(t *testing.T, dir string) string {
	t.Helper()
	return gittest.Run(t, dir, "rev-parse", "HEAD")
}

// initRepo starts with one commit, so an operation under test always has
// something before it.
func initRepo(t *testing.T) string {
	t.Helper()
	dir := gittest.InitRepo(t)
	commitFile(t, dir, "a.txt", "one\n", "first")
	return dir
}

// undoOnce previews and applies, checking the operation it found.
func undoOnce(t *testing.T, dir, wantOp string) Plan {
	t.Helper()
	ctx := context.Background()
	plan, err := Preview(ctx, dir)
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if plan.Operation != wantOp {
		t.Fatalf("Operation = %q, want %q (plan %+v)", plan.Operation, wantOp, plan)
	}
	if err := Apply(ctx, dir, plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	return plan
}

func TestUndoCommitKeepsChangesStaged(t *testing.T) {
	dir := initRepo(t)
	before := head(t, dir)
	commitFile(t, dir, "b.txt", "new\n", "second")

	plan := undoOnce(t, dir, Commit)

	if head(t, dir) != before || plan.Removed != 1 || plan.Restored != 0 || plan.Branch != "main" {
		t.Errorf("plan %+v, HEAD %s, want HEAD %s with one commit removed", plan, head(t, dir), before)
	}
	if staged := gittest.Run(t, dir, "diff", "--cached", "--name-only"); staged != "b.txt" {
		t.Errorf("staged = %q, want b.txt", staged)
	}
}

func TestUndoAmendRestoresOriginalCommit(t *testing.T) {
	dir := initRepo(t)
	original := commitFile(t, dir, "b.txt", "one\n", "second")
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gittest.Run(t, dir, "commit", "-a", "--amend", "-m", "second, amended")

	undoOnce(t, dir, Amend)

	if head(t, dir) != original {
		t.Errorf("HEAD = %s, want the original commit %s", head(t, dir), original)
	}
	if staged := gittest.Run(t, dir, "diff", "--cached", "--name-only"); staged != "b.txt" {
		t.Errorf("staged = %q, want the amended change to b.txt", staged)
	}
}

func TestUndoMerge(t *testing.T) {
	for _, ff := range []bool{false, true} {
		t.Run(map[bool]string{false: "merge commit", true: "fast-forward"}[ff], func(t *testing.T) {
			dir := initRepo(t)
			gittest.Run(t, dir, "switch", "-c", "feature")
			commitFile(t, dir, "f.txt", "f\n", "feature work")
			gittest.Run(t, dir, "switch", "main")
			if !ff {
				commitFile(t, dir, "m.txt", "m\n", "main work")
			}
			before := head(t, dir)
			args := []string{"merge", "feature", "-m", "merge feature"}
			if !ff {
				args = append(args, "--no-ff")
			}
			gittest.Run(t, dir, args...)

			plan := undoOnce(t, dir, Merge)

			if head(t, dir) != before || plan.Mode != git.ResetKeep {
				t.Errorf("HEAD = %s (plan %+v), want %s with a keep reset", head(t, dir), plan, before)
			}
			if _, err := os.Stat(filepath.Join(dir, "f.txt")); !os.IsNotExist(err) {
				t.Error("f.txt still in the working tree after undoing the merge")
			}
		})
	}
}

func TestUndoRebaseReturnsToPreRebaseTip(t *testing.T) {
	dir := initRepo(t)
	gittest.Run(t, dir, "switch", "-c", "feature")
	commitFile(t, dir, "f1.txt", "1\n", "feature one")
	tip := commitFile(t, dir, "f2.txt", "2\n", "feature two")
	gittest.Run(t, dir, "switch", "main")
	commitFile(t, dir, "m.txt", "m\n", "main work")
	gittest.Run(t, dir, "switch", "feature")
	gittest.Run(t, dir, "rebase", "main")

	plan := undoOnce(t, dir, Rebase)

	if head(t, dir) != tip || plan.Removed != 3 || plan.Restored != 2 {
		t.Errorf("HEAD = %s (plan %+v), want the pre-rebase tip %s; 3 removed (2 rebased + main's commit), 2 restored", head(t, dir), plan, tip)
	}
}

func TestUndoHardReset(t *testing.T) {
	dir := initRepo(t)
	lost := commitFile(t, dir, "b.txt", "b\n", "second")
	gittest.Run(t, dir, "reset", "--hard", "HEAD~1")

	plan := undoOnce(t, dir, Reset)

	if head(t, dir) != lost || plan.Restored != 1 {
		t.Errorf("HEAD = %s (plan %+v), want the reset-away commit %s", head(t, dir), plan, lost)
	}
	if _, err := os.Stat(filepath.Join(dir, "b.txt")); err != nil {
		t.Errorf("b.txt not restored: %v", err)
	}
}

func TestUndoCherryPick(t *testing.T) {
	dir := initRepo(t)
	gittest.Run(t, dir, "switch", "-c", "feature")
	picked := commitFile(t, dir, "f.txt", "f\n", "feature work")
	gittest.Run(t, dir, "switch", "main")
	before := head(t, dir)
	gittest.Run(t, dir, "cherry-pick", picked)

	undoOnce(t, dir, CherryPick)

	if head(t, dir) != before {
		t.Errorf("HEAD = %s, want %s", head(t, dir), before)
	}
}

func TestUndoSwitchGoesBack(t *testing.T) {
	dir := initRepo(t)
	gittest.Run(t, dir, "branch", "feature")
	gittest.Run(t, dir, "checkout", "feature")

	plan := undoOnce(t, dir, Switch)

	if branch := gittest.Run(t, dir, "branch", "--show-current"); branch != "main" || plan.SwitchTo != "main" {
		t.Errorf("on %q (plan %+v), want back on main", branch, plan)
	}
}

func TestUndoTwiceRedoes(t *testing.T) {
	dir := initRepo(t)
	gittest.Run(t, dir, "switch", "-c", "feature")
	commitFile(t, dir, "f.txt", "f\n", "feature work")
	gittest.Run(t, dir, "switch", "main")
	gittest.Run(t, dir, "merge", "--no-ff", "feature", "-m", "merge feature")
	merged := head(t, dir)

	undoOnce(t, dir, Merge)
	undoOnce(t, dir, Reset)

	if head(t, dir) != merged {
		t.Errorf("HEAD = %s, want the merge %s back", head(t, dir), merged)
	}
}

func TestPreviewRefusesLocalChangesForKeepReset(t *testing.T) {
	dir := initRepo(t)
	gittest.Run(t, dir, "switch", "-c", "feature")
	commitFile(t, dir, "f.txt", "f\n", "feature work")
	gittest.Run(t, dir, "switch", "main")
	gittest.Run(t, dir, "merge", "--no-ff", "feature", "-m", "merge feature")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Preview(context.Background(), dir); !errors.Is(err, ErrLocalChanges) {
		t.Errorf("Preview error = %v, want ErrLocalChanges", err)
	}
}

func TestPreviewAllowsUntrackedFiles(t *testing.T) {
	dir := initRepo(t)
	commitFile(t, dir, "b.txt", "b\n", "second")
	gittest.Run(t, dir, "reset", "--hard", "HEAD~1")
	if err := os.WriteFile(filepath.Join(dir, "scratch.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Preview(context.Background(), dir); err != nil {
		t.Errorf("Preview with only an untracked file: %v", err)
	}
}

func TestApplyRefusesWhenHistoryMoved(t *testing.T) {
	dir := initRepo(t)
	commitFile(t, dir, "b.txt", "b\n", "second")
	plan, err := Preview(context.Background(), dir)
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	commitFile(t, dir, "c.txt", "c\n", "third")

	if err := Apply(context.Background(), dir, plan); !errors.Is(err, ErrHistoryChanged) {
		t.Errorf("Apply error = %v, want ErrHistoryChanged", err)
	}
}

func TestPreviewRefusesDuringConflict(t *testing.T) {
	dir := initRepo(t)
	gittest.Run(t, dir, "switch", "-c", "feature")
	commitFile(t, dir, "a.txt", "feature\n", "feature edit")
	gittest.Run(t, dir, "switch", "main")
	commitFile(t, dir, "a.txt", "main\n", "main edit")
	cmd := exec.Command("git", "merge", "feature")
	cmd.Dir = dir
	_ = cmd.Run()

	_, err := Preview(context.Background(), dir)
	if err == nil || !strings.Contains(err.Error(), "Finish or abort the merge") {
		t.Errorf("Preview error = %v, want a finish-or-abort message", err)
	}
}

func TestPreviewNothingToUndoAfterFirstCommit(t *testing.T) {
	dir := initRepo(t)
	if _, err := Preview(context.Background(), dir); !errors.Is(err, ErrNothingToUndo) {
		t.Errorf("Preview error = %v, want ErrNothingToUndo", err)
	}
}

func TestPreviewMarksPushedCommits(t *testing.T) {
	remote := t.TempDir()
	gittest.Run(t, remote, "init", "--bare", "-b", "main")
	dir := initRepo(t)
	gittest.Run(t, dir, "remote", "add", "origin", remote)
	commitFile(t, dir, "b.txt", "b\n", "second")
	gittest.Run(t, dir, "push", "-u", "origin", "main")

	plan, err := Preview(context.Background(), dir)
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if !plan.Pushed {
		t.Errorf("plan %+v, want Pushed for a commit already on origin/main", plan)
	}

	commitFile(t, dir, "c.txt", "c\n", "third")
	if plan, err = Preview(context.Background(), dir); err != nil || plan.Pushed {
		t.Errorf("plan %+v (err %v), want not Pushed for a local-only commit", plan, err)
	}
}

func TestUndoRevert(t *testing.T) {
	dir := initRepo(t)
	before := commitFile(t, dir, "b.txt", "b\n", "second")
	gittest.Run(t, dir, "revert", "--no-edit", "HEAD")

	undoOnce(t, dir, Revert)

	if head(t, dir) != before {
		t.Errorf("HEAD = %s, want %s", head(t, dir), before)
	}
}

func TestUndoPull(t *testing.T) {
	upstream := initRepo(t)
	dir := t.TempDir()
	gittest.Run(t, dir, "clone", upstream, ".")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	before := head(t, dir)
	commitFile(t, upstream, "b.txt", "b\n", "upstream work")
	gittest.Run(t, dir, "pull", "--ff-only")

	undoOnce(t, dir, Pull)

	if head(t, dir) != before {
		t.Errorf("HEAD = %s, want %s", head(t, dir), before)
	}
}
