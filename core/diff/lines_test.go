package diff

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

const before = "one\ntwo\nthree\nfour\n"

// "two" becomes "TWO" and "five" is added; the tests stage, unstage or
// discard only some of those changes.
const after = "one\nTWO\nthree\nfour\nfive\n"

func initLinesRepo(t *testing.T) string {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "f.txt", before, "initial")
	gittest.WriteFile(t, dir, "f.txt", after)
	return dir
}

func onlyHunk(t *testing.T, fd FileDiff) string {
	t.Helper()
	if len(fd.Hunks) != 1 {
		t.Fatalf("want one hunk, got %d", len(fd.Hunks))
	}
	return fd.Hunks[0].Raw
}

func readFile(t *testing.T, dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "f.txt"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestStageLinesStagesOnlyTheSelection(t *testing.T) {
	ctx := context.Background()
	dir := initLinesRepo(t)
	fd, _ := GetWorkingTreeDiff(ctx, dir, "f.txt", false, false)

	// Only the added "five" (new line 5).
	if err := StageLines(ctx, dir, "f.txt", onlyHunk(t, fd), LineSelection{Added: []int{5}}); err != nil {
		t.Fatalf("StageLines: %v", err)
	}
	if got := gittest.Run(t, dir, "show", ":f.txt"); got != "one\ntwo\nthree\nfour\nfive" {
		t.Fatalf("index = %q, want only five added", got)
	}

	// Then only the removal of "two" (old line 2), without its replacement.
	fd, _ = GetWorkingTreeDiff(ctx, dir, "f.txt", false, false)
	if err := StageLines(ctx, dir, "f.txt", onlyHunk(t, fd), LineSelection{Removed: []int{2}}); err != nil {
		t.Fatalf("StageLines: %v", err)
	}
	if got := gittest.Run(t, dir, "show", ":f.txt"); got != "one\nthree\nfour\nfive" {
		t.Fatalf("index = %q, want two removed as well", got)
	}
	if readFile(t, dir) != after {
		t.Fatal("staging changed the working tree")
	}
}

func TestUnstageLinesUnstagesOnlyTheSelection(t *testing.T) {
	ctx := context.Background()
	dir := initLinesRepo(t)
	gittest.Run(t, dir, "add", "f.txt")
	fd, _ := GetIndexDiff(ctx, dir, "f.txt", false)

	// Unstage the change to "two" (old line 2, new line 2), keep "five" staged.
	if err := UnstageLines(ctx, dir, "f.txt", onlyHunk(t, fd), LineSelection{Removed: []int{2}, Added: []int{2}}); err != nil {
		t.Fatalf("UnstageLines: %v", err)
	}
	if got := gittest.Run(t, dir, "show", ":f.txt"); got != "one\ntwo\nthree\nfour\nfive" {
		t.Fatalf("index = %q, want two restored and five still staged", got)
	}
}

func TestUnstageHunkOfANewFileUnstagesTheFile(t *testing.T) {
	ctx := context.Background()
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "other.txt", "x\n", "initial")
	gittest.WriteFile(t, dir, "f.txt", "a\nb\n")
	gittest.Run(t, dir, "add", "f.txt")
	fd, _ := GetIndexDiff(ctx, dir, "f.txt", false)

	if err := UnstageHunk(ctx, dir, "f.txt", onlyHunk(t, fd)); err != nil {
		t.Fatalf("UnstageHunk: %v", err)
	}
	if got := gittest.Run(t, dir, "status", "--porcelain", "f.txt"); got != "?? f.txt" {
		t.Fatalf("status = %q, want the file untracked again, not an empty file staged", got)
	}
}

func TestUnstageSomeLinesOfANewFileKeepsTheRestStaged(t *testing.T) {
	ctx := context.Background()
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "other.txt", "x\n", "initial")
	gittest.WriteFile(t, dir, "f.txt", "a\nb\n")
	gittest.Run(t, dir, "add", "f.txt")
	fd, _ := GetIndexDiff(ctx, dir, "f.txt", false)

	if err := UnstageLines(ctx, dir, "f.txt", onlyHunk(t, fd), LineSelection{Added: []int{2}}); err != nil {
		t.Fatalf("UnstageLines: %v", err)
	}
	if got := gittest.Run(t, dir, "show", ":f.txt"); got != "a" {
		t.Fatalf("index = %q, want only the first line still staged", got)
	}
}

func TestStageHunkOfADeletedFileStagesTheDeletion(t *testing.T) {
	ctx := context.Background()
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "f.txt", "a\nb\n", "initial")
	if err := os.Remove(filepath.Join(dir, "f.txt")); err != nil {
		t.Fatal(err)
	}
	fd, _ := GetWorkingTreeDiff(ctx, dir, "f.txt", false, false)

	if err := StageHunk(ctx, dir, "f.txt", onlyHunk(t, fd)); err != nil {
		t.Fatalf("StageHunk: %v", err)
	}
	if got := gittest.Run(t, dir, "status", "--porcelain", "f.txt"); got != "D  f.txt" {
		t.Fatalf("status = %q, want the deletion staged, not an empty file", got)
	}
}

func TestStageHunkIntoAnExistingEmptyFile(t *testing.T) {
	ctx := context.Background()
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "f.txt", "", "initial")
	gittest.WriteFile(t, dir, "f.txt", "a\n")
	fd, _ := GetWorkingTreeDiff(ctx, dir, "f.txt", false, false)

	if err := StageHunk(ctx, dir, "f.txt", onlyHunk(t, fd)); err != nil {
		t.Fatalf("StageHunk: %v", err)
	}
	if got := gittest.Run(t, dir, "status", "--porcelain", "f.txt"); got != "M  f.txt" {
		t.Fatalf("status = %q, want the edit staged", got)
	}
}

func TestDiscardLinesRevertsOnlyTheSelectionInTheWorkingTree(t *testing.T) {
	ctx := context.Background()
	dir := initLinesRepo(t)
	fd, _ := GetWorkingTreeDiff(ctx, dir, "f.txt", false, false)

	if err := DiscardLines(ctx, dir, "f.txt", onlyHunk(t, fd), LineSelection{Added: []int{5}}); err != nil {
		t.Fatalf("DiscardLines: %v", err)
	}
	if got := readFile(t, dir); got != "one\nTWO\nthree\nfour\n" {
		t.Fatalf("working tree = %q, want only five discarded", got)
	}
}

func TestLinesNeedAChangedLineSelected(t *testing.T) {
	ctx := context.Background()
	dir := initLinesRepo(t)
	fd, _ := GetWorkingTreeDiff(ctx, dir, "f.txt", false, false)

	err := StageLines(ctx, dir, "f.txt", onlyHunk(t, fd), LineSelection{Added: []int{1}})
	if err == nil || err.Error() != "Select a changed line in this hunk." {
		t.Fatalf("err = %v, want the select-a-line message", err)
	}
}

func TestStageLinesAtEndOfFileWithoutNewline(t *testing.T) {
	ctx := context.Background()
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "f.txt", "a\nb", "initial")
	gittest.WriteFile(t, dir, "f.txt", "a\nB\nc")
	fd, _ := GetWorkingTreeDiff(ctx, dir, "f.txt", false, false)

	// Stage b -> B but not the added c.
	if err := StageLines(ctx, dir, "f.txt", onlyHunk(t, fd), LineSelection{Removed: []int{2}, Added: []int{2}}); err != nil {
		t.Fatalf("StageLines: %v", err)
	}
	if got := gittest.Run(t, dir, "show", ":f.txt"); got != "a\nB" {
		t.Fatalf("index = %q, want a, B", got)
	}
}
