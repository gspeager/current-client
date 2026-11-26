package rework

import (
	"context"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/internal/gittest"
)

var ctx = context.Background()

func lines(n int, word string) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		b.WriteString(word + " " + strings.Repeat("x", i) + "\n")
	}
	return b.String()
}

// repo returns a repository whose base commit holds old.go, with HEAD at base.
func repo(t *testing.T) (dir, base string) {
	t.Helper()
	dir = gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "old.go", lines(4, "before"), "base")
	return dir, strings.TrimSpace(gittest.Output(t, dir, "rev-parse", "HEAD"))
}

func report(t *testing.T, dir, base string) Report {
	t.Helper()
	r, err := Range(ctx, dir, base, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestRewriteCountsLinesWrittenEarlierInTheRange(t *testing.T) {
	dir, base := repo(t)
	gittest.CommitFile(t, dir, "a.go", lines(5, "first"), "write")
	gittest.CommitFile(t, dir, "a.go", lines(2, "first")+lines(3, "second"), "rewrite three")
	r := report(t, dir, base)
	if r.Written != 8 || r.Rewritten != 3 {
		t.Fatalf("written %d rewritten %d, want 8 and 3", r.Written, r.Rewritten)
	}
	if len(r.Commits) != 2 || r.Commits[0].Rewritten != 0 || r.Commits[1].Rewritten != 3 {
		t.Fatalf("commits %+v", r.Commits)
	}
	if len(r.Files) != 1 || r.Files[0] != (File{Path: "a.go", Written: 8, Rewritten: 3, Commits: 2}) {
		t.Fatalf("files %+v", r.Files)
	}
}

func TestLinesFromBeforeTheRangeArentRework(t *testing.T) {
	dir, base := repo(t)
	gittest.CommitFile(t, dir, "old.go", lines(2, "new"), "replace what the base wrote")
	if r := report(t, dir, base); r.Rewritten != 0 || len(r.Files) != 0 {
		t.Fatalf("got %+v", r)
	}
}

func TestMovedLinesArentRework(t *testing.T) {
	dir, base := repo(t)
	block := "func movedAround() error {\n\treturn errors.New(\"distinctive enough to follow\")\n}\n"
	gittest.CommitFile(t, dir, "a.go", block, "write")
	gittest.WriteFile(t, dir, "a.go", "")
	gittest.WriteFile(t, dir, "b.go", "\t"+strings.ReplaceAll(block, "\n", "\n\t"))
	gittest.Run(t, dir, "add", "-A")
	gittest.Run(t, dir, "commit", "-q", "-m", "move, indented")
	if r := report(t, dir, base); r.Rewritten != 0 {
		t.Fatalf("rewritten %d, want 0", r.Rewritten)
	}
}

func TestADeletedFileIsReworkUnderItsLastName(t *testing.T) {
	dir, base := repo(t)
	gittest.CommitFile(t, dir, "gone.go", lines(3, "short-lived"), "write")
	gittest.Run(t, dir, "rm", "-q", "gone.go")
	gittest.Run(t, dir, "commit", "-q", "-m", "delete")
	r := report(t, dir, base)
	if r.Rewritten != 3 || len(r.Files) != 1 || r.Files[0].Path != "gone.go" {
		t.Fatalf("got %+v", r)
	}
}

func TestARenamedFileKeepsItsReworkUnderItsNewName(t *testing.T) {
	dir, base := repo(t)
	gittest.CommitFile(t, dir, "a.go", lines(6, "kept"), "write")
	gittest.Run(t, dir, "mv", "a.go", "b.go")
	gittest.Run(t, dir, "commit", "-q", "-m", "rename")
	gittest.CommitFile(t, dir, "b.go", lines(5, "kept")+"changed\n", "rewrite one")
	r := report(t, dir, base)
	if r.Rewritten != 1 || len(r.Files) != 1 || r.Files[0].Path != "b.go" {
		t.Fatalf("got %+v", r)
	}
}

func TestBlankLinesDontCount(t *testing.T) {
	dir, base := repo(t)
	gittest.CommitFile(t, dir, "a.go", "one\n\n\ntwo\n", "write")
	gittest.CommitFile(t, dir, "a.go", "one\ntwo\n", "close up")
	r := report(t, dir, base)
	if r.Written != 2 || r.Rewritten != 0 {
		t.Fatalf("written %d rewritten %d, want 2 and 0", r.Written, r.Rewritten)
	}
}

func TestAnEmptyRange(t *testing.T) {
	dir, base := repo(t)
	r := report(t, dir, base)
	if r.Written != 0 || len(r.Commits) != 0 || len(r.Files) != 0 || r.Capped {
		t.Fatalf("got %+v", r)
	}
}

func TestDeletedSpan(t *testing.T) {
	for header, want := range map[string][2]int{
		"@@ -3,2 +3,0 @@":      {3, 2},
		"@@ -7 +7 @@ func a()": {7, 1},
		"@@ -0,0 +1,4 @@":      {0, 0},
	} {
		start, count, ok := deletedSpan(header)
		if !ok || start != want[0] || count != want[1] {
			t.Errorf("%q: got %d,%d,%v", header, start, count, ok)
		}
	}
}

func TestShortLinesAreMovedOnlyBesideADistinctiveMovedLine(t *testing.T) {
	line := func(n int, text string) git.BlameLine { return git.BlameLine{SHA: "w", LineNo: n, Content: text} }
	added := func() map[string]int {
		return map[string]int{"return errors.New(\"distinctive enough\")": 1, "}": 2, "return nil": 1}
	}
	read := map[string]bool{"w": true}
	moved := []git.BlameLine{line(4, "\treturn errors.New(\"distinctive enough\")"), line(5, "}")}
	if n := countRewritten(moved, added(), read); n != 0 {
		t.Errorf("a brace beside a moved line: rewritten %d, want 0", n)
	}
	alone := []git.BlameLine{line(9, "\treturn nil"), line(10, "}")}
	if n := countRewritten(alone, added(), read); n != 2 {
		t.Errorf("short lines on their own: rewritten %d, want 2", n)
	}
}

func TestAPathWithASpace(t *testing.T) {
	dir, base := repo(t)
	gittest.CommitFile(t, dir, "release notes.md", lines(3, "note"), "write")
	gittest.CommitFile(t, dir, "release notes.md", lines(1, "note"), "trim")
	r := report(t, dir, base)
	if r.Rewritten != 2 || len(r.Files) != 1 || r.Files[0].Path != "release notes.md" {
		t.Fatalf("got %+v", r)
	}
}

func TestANewFileUnderARenamedFilesOldName(t *testing.T) {
	dir, base := repo(t)
	gittest.CommitFile(t, dir, "util.go", lines(6, "util"), "write")
	gittest.Run(t, dir, "mv", "util.go", "text.go")
	gittest.Run(t, dir, "commit", "-q", "-m", "rename")
	gittest.CommitFile(t, dir, "util.go", lines(3, "fresh"), "a new util")
	gittest.CommitFile(t, dir, "util.go", lines(1, "fresh"), "trim it")
	r := report(t, dir, base)
	if len(r.Files) != 1 || r.Files[0].Path != "util.go" || r.Files[0].Rewritten != 2 {
		t.Fatalf("files %+v", r.Files)
	}
}

func TestOnlyTheNewestCommitsAreRead(t *testing.T) {
	readLimit = 2
	t.Cleanup(func() { readLimit = MostCommits })
	dir, base := repo(t)
	gittest.CommitFile(t, dir, "a.go", lines(4, "older"), "write, unread")
	gittest.CommitFile(t, dir, "b.go", "b\n", "read")
	gittest.CommitFile(t, dir, "a.go", lines(1, "older"), "rewrite what an unread commit wrote")
	r := report(t, dir, base)
	if !r.Capped || len(r.Commits) != 2 || r.Rewritten != 0 {
		t.Fatalf("got %+v", r)
	}
}

func TestManyHunksBlameTheWholeFile(t *testing.T) {
	dir, base := repo(t)
	var before, after strings.Builder
	for i := 0; i < 2*mostRanges+10; i++ {
		line := strings.Repeat("y", i+1)
		before.WriteString(line + "\n")
		if i%2 == 0 {
			after.WriteString(line + "\n")
		}
	}
	gittest.CommitFile(t, dir, "many.txt", before.String(), "write")
	gittest.CommitFile(t, dir, "many.txt", after.String(), "drop every other line")
	if r := report(t, dir, base); r.Rewritten != mostRanges+5 {
		t.Fatalf("rewritten %d, want %d", r.Rewritten, mostRanges+5)
	}
}
