package git

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestParseWorktrees(t *testing.T) {
	output := "worktree /repo\nHEAD aaa\nbranch refs/heads/main\n\n" +
		"worktree /wt/det\nHEAD bbb\ndetached\nlocked usb drive\n\n" +
		"worktree /wt/gone\nHEAD ccc\nbranch refs/heads/feature/x\nprunable gitdir file points to non-existent location\n"
	want := []Worktree{
		{Path: filepath.FromSlash("/repo"), Branch: "main", Head: "aaa", Main: true},
		{Path: filepath.FromSlash("/wt/det"), Head: "bbb", Locked: true},
		{Path: filepath.FromSlash("/wt/gone"), Branch: "feature/x", Head: "ccc", Missing: true},
	}
	if got := parseWorktrees(output); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestAddListAndRemoveWorktrees(t *testing.T) {
	ctx := context.Background()
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	gittest.Run(t, dir, "branch", "existing")
	parent := t.TempDir()
	existingPath := filepath.Join(parent, "existing")
	newPath := filepath.Join(parent, "new")

	if err := AddWorktree(ctx, dir, existingPath, "existing", false, ""); err != nil {
		t.Fatalf("AddWorktree existing: %v", err)
	}
	if err := AddWorktree(ctx, dir, newPath, "feature/new", true, ""); err != nil {
		t.Fatalf("AddWorktree new: %v", err)
	}
	if err := AddWorktree(ctx, dir, newPath, "other", true, ""); err == nil || err.Error() != "That folder already exists." {
		t.Fatalf("adding into an existing folder: err = %v", err)
	}
	if err := AddWorktree(ctx, dir, filepath.Join(parent, "again"), "existing", false, ""); err == nil ||
		err.Error() != "That branch is checked out in another worktree." {
		t.Fatalf("checking out a branch twice: err = %v", err)
	}

	worktrees, err := ListWorktrees(ctx, existingPath)
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	if len(worktrees) != 3 {
		t.Fatalf("got %d worktrees, want 3: %+v", len(worktrees), worktrees)
	}
	if !worktrees[0].Main || worktrees[0].Branch != "main" || worktrees[0].Current {
		t.Fatalf("main worktree = %+v", worktrees[0])
	}
	byBranch := map[string]Worktree{}
	for _, w := range worktrees {
		byBranch[w.Branch] = w
	}
	if w := byBranch["existing"]; !w.Current || !samePath(w.Path, existingPath) {
		t.Fatalf("existing worktree = %+v, want the current one at %s", w, existingPath)
	}
	if w := byBranch["feature/new"]; w.Current || !samePath(w.Path, newPath) {
		t.Fatalf("new worktree = %+v", w)
	}

	gittest.WriteFile(t, newPath, "dirty.txt", "x")
	if err := RemoveWorktree(ctx, dir, newPath, false); err == nil || err.Error() != "This worktree has changes." {
		t.Fatalf("removing a worktree with changes: err = %v", err)
	}
	if err := RemoveWorktree(ctx, dir, newPath, true); err != nil {
		t.Fatalf("RemoveWorktree --force: %v", err)
	}
	if _, err := os.Stat(newPath); !os.IsNotExist(err) {
		t.Fatal("the removed worktree's folder is still there")
	}
}

func TestMissingWorktreeIsReportedAndRemovable(t *testing.T) {
	ctx := context.Background()
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	gone := filepath.Join(t.TempDir(), "gone")
	if err := AddWorktree(ctx, dir, gone, "gone", true, ""); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}

	worktrees, _ := ListWorktrees(ctx, dir)
	if len(worktrees) != 2 || !worktrees[1].Missing {
		t.Fatalf("worktrees = %+v, want the second marked missing", worktrees)
	}
	if err := RemoveWorktree(ctx, dir, worktrees[1].Path, false); err != nil {
		t.Fatalf("RemoveWorktree of a missing worktree: %v", err)
	}
	if worktrees, _ := ListWorktrees(ctx, dir); len(worktrees) != 1 {
		t.Fatalf("still listed after removal: %+v", worktrees)
	}
}

func TestListBranchesReportsTheirWorktree(t *testing.T) {
	ctx := context.Background()
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	elsewhere := filepath.Join(t.TempDir(), "elsewhere")
	if err := AddWorktree(ctx, dir, elsewhere, "feature", true, ""); err != nil {
		t.Fatalf("AddWorktree: %v", err)
	}

	branches, err := ListBranches(ctx, dir)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	for _, b := range branches {
		switch b.Name {
		case "feature":
			if !samePath(b.WorktreePath, elsewhere) {
				t.Fatalf("feature's worktree = %q, want %q", b.WorktreePath, elsewhere)
			}
		case "main":
			if !samePath(b.WorktreePath, dir) {
				t.Fatalf("main's worktree = %q, want the main one %q", b.WorktreePath, dir)
			}
		}
	}
}
