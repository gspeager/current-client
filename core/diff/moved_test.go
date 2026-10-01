package diff

import (
	"context"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestParseUnifiedDiffMarksMovedBlock(t *testing.T) {
	const fixture = `diff --git a/file.txt b/file.txt
index 83db48f..bf269c4 100644
--- a/file.txt
+++ b/file.txt
@@ -1,5 +1,4 @@
 context1
-func Foo() {
-	return 1
-}
 context2
@@ -10,3 +9,6 @@
 context4
+func Foo() {
+	return 1
+}
 context5
`

	got, err := ParseUnifiedDiff(fixture)
	if err != nil {
		t.Fatalf("ParseUnifiedDiff: %v", err)
	}

	firstHunk := got.Hunks[0].Lines
	for _, l := range firstHunk[1:4] {
		if l.Kind != LineRemoved || !l.Moved {
			t.Fatalf("got %+v, want a moved removed line", l)
		}
	}
	if firstHunk[0].Moved || firstHunk[4].Moved {
		t.Fatalf("got context lines marked moved: %+v", firstHunk)
	}

	secondHunk := got.Hunks[1].Lines
	for _, l := range secondHunk[1:4] {
		if l.Kind != LineAdded || !l.Moved {
			t.Fatalf("got %+v, want a moved added line", l)
		}
	}
	if secondHunk[0].Moved || secondHunk[4].Moved {
		t.Fatalf("got context lines marked moved: %+v", secondHunk)
	}
}

func TestParseUnifiedDiffDoesNotMarkSingleLineMatchAsMoved(t *testing.T) {
	const fixture = `diff --git a/file.txt b/file.txt
index 83db48f..bf269c4 100644
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,3 @@
 context1
-shared line
+shared line
 context2
`
	got, err := ParseUnifiedDiff(fixture)
	if err != nil {
		t.Fatalf("ParseUnifiedDiff: %v", err)
	}
	for _, l := range got.Hunks[0].Lines {
		if l.Moved {
			t.Fatalf("got a single-line match marked moved, want the minimum-block-length threshold to exclude it: %+v", got.Hunks[0].Lines)
		}
	}
}

func TestParseUnifiedDiffDoesNotMarkOrdinaryEditsAsMoved(t *testing.T) {
	const fixture = `diff --git a/file.txt b/file.txt
index 83db48f..bf269c4 100644
--- a/file.txt
+++ b/file.txt
@@ -1,4 +1,5 @@
 unchanged line
-removed line
+added line
+another added line
 unchanged line 2
`
	got, err := ParseUnifiedDiff(fixture)
	if err != nil {
		t.Fatalf("ParseUnifiedDiff: %v", err)
	}
	for _, l := range got.Hunks[0].Lines {
		if l.Moved {
			t.Fatalf("got an ordinary add/remove pair marked moved: %+v", got.Hunks[0].Lines)
		}
	}
}

func TestGetWorkingTreeDiffMarksRelocatedFunctionAsMoved(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	gittest.Run(t, dir, "config", "core.autocrlf", "false")

	original := "package main\n\nfunc Foo() int {\n\treturn 1\n}\n\nfunc Bar() int {\n\treturn 2\n}\n"
	gittest.WriteFile(t, dir, "file.go", original)
	gittest.Run(t, dir, "add", "file.go")
	gittest.Run(t, dir, "commit", "-m", "initial")

	// Foo and Bar swap places, byte-for-byte, elsewhere unchanged.
	moved := "package main\n\nfunc Bar() int {\n\treturn 2\n}\n\nfunc Foo() int {\n\treturn 1\n}\n"
	gittest.WriteFile(t, dir, "file.go", moved)

	fd, err := GetWorkingTreeDiff(context.Background(), dir, "file.go", false, false)
	if err != nil {
		t.Fatalf("GetWorkingTreeDiff: %v", err)
	}

	var movedCount int
	for _, h := range fd.Hunks {
		for _, l := range h.Lines {
			if l.Moved {
				movedCount++
			}
		}
	}
	if movedCount == 0 {
		t.Fatalf("got no moved lines, want the relocated Foo/Bar bodies marked moved: %+v", fd)
	}
}
