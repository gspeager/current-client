package diff

import (
	"context"
	"os/exec"
	"reflect"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

// Captured from `git diff` on a real merge conflict (git 2.x, 2026-09-24).
const conflictDiff = `diff --cc f
index 5742e7d,0c02ccc..0000000
--- a/f
+++ b/f
@@@ -1,3 -1,3 +1,7 @@@
  a
++<<<<<<< HEAD
 +M
++=======
+ X
++>>>>>>> x
  c
`

func TestParseCombinedDiffClassifiesConflictSections(t *testing.T) {
	got, err := ParseUnifiedDiff(conflictDiff)
	if err != nil {
		t.Fatalf("ParseUnifiedDiff: %v", err)
	}
	if !got.Conflicted {
		t.Fatal("Conflicted = false, want true for a diff --cc")
	}
	if got.OldPath != "f" || got.NewPath != "f" {
		t.Fatalf("paths = %q/%q, want f/f", got.OldPath, got.NewPath)
	}
	if len(got.Hunks) != 1 {
		t.Fatalf("got %d hunks, want 1", len(got.Hunks))
	}
	want := []Line{
		{Kind: LineContext, OldLine: 1, NewLine: 1, Content: "a"},
		{Kind: LineConflictMarker, NewLine: 2, Content: "<<<<<<< HEAD"},
		{Kind: LineConflictOurs, OldLine: 2, NewLine: 3, Content: "M"},
		{Kind: LineConflictMarker, NewLine: 4, Content: "======="},
		{Kind: LineConflictTheirs, NewLine: 5, Content: "X"},
		{Kind: LineConflictMarker, NewLine: 6, Content: ">>>>>>> x"},
		{Kind: LineContext, OldLine: 3, NewLine: 7, Content: "c"},
	}
	if !reflect.DeepEqual(got.Hunks[0].Lines, want) {
		t.Fatalf("lines =\n%+v\nwant\n%+v", got.Hunks[0].Lines, want)
	}
	if got.Hunks[0].OldStart != 1 || got.Hunks[0].NewStart != 1 || got.Hunks[0].NewLines != 7 {
		t.Fatalf("hunk ranges = %+v", got.Hunks[0])
	}
}

func TestParseCombinedDiffTreatsDiff3BaseAsMarkerSection(t *testing.T) {
	diff3 := `diff --cc f
--- a/f
+++ b/f
@@@ -1,1 -1,1 +1,7 @@@
++<<<<<<< HEAD
 +ours
++||||||| base
++original
++=======
+ theirs
++>>>>>>> x
`
	got, err := ParseUnifiedDiff(diff3)
	if err != nil {
		t.Fatalf("ParseUnifiedDiff: %v", err)
	}
	kinds := make([]LineKind, len(got.Hunks[0].Lines))
	for i, l := range got.Hunks[0].Lines {
		kinds[i] = l.Kind
	}
	want := []LineKind{LineConflictMarker, LineConflictOurs, LineConflictMarker, LineConflictMarker, LineConflictMarker, LineConflictTheirs, LineConflictMarker}
	if !reflect.DeepEqual(kinds, want) {
		t.Fatalf("kinds = %v, want %v", kinds, want)
	}
}

func TestGetWorkingTreeDiffShowsAMergeConflict(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init", "-b", "main")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	gittest.Run(t, dir, "config", "core.autocrlf", "false")
	gittest.WriteFile(t, dir, "f.txt", "a\nb\nc\n")
	gittest.Run(t, dir, "add", "f.txt")
	gittest.Run(t, dir, "commit", "-m", "base")
	gittest.Run(t, dir, "checkout", "-q", "-b", "theirs")
	gittest.WriteFile(t, dir, "f.txt", "a\nTHEIRS\nc\n")
	gittest.Run(t, dir, "commit", "-qam", "theirs")
	gittest.Run(t, dir, "checkout", "-q", "main")
	gittest.WriteFile(t, dir, "f.txt", "a\nOURS\nc\n")
	gittest.Run(t, dir, "commit", "-qam", "ours")
	merge := exec.Command("git", "merge", "theirs")
	merge.Dir = dir
	if err := merge.Run(); err == nil {
		t.Fatal("expected the merge to conflict")
	}

	got, err := GetWorkingTreeDiff(context.Background(), dir, "f.txt", false, false)
	if err != nil {
		t.Fatalf("GetWorkingTreeDiff: %v", err)
	}
	if !got.Conflicted || len(got.Hunks) != 1 {
		t.Fatalf("got %+v, want one conflicted hunk", got)
	}
	var ours, theirs string
	for _, l := range got.Hunks[0].Lines {
		switch l.Kind {
		case LineConflictOurs:
			ours = l.Content
		case LineConflictTheirs:
			theirs = l.Content
		}
	}
	if ours != "OURS" || theirs != "THEIRS" {
		t.Fatalf("ours/theirs = %q/%q, want OURS/THEIRS", ours, theirs)
	}
}
