package git

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gspeager/current-client/internal/gittest"
)

func rangeLines(t *testing.T, dir, from, to string, since time.Time, most int) ([]CommitLines, bool) {
	t.Helper()
	commits, more, err := RangeCommitLines(context.Background(), dir, from, to, since, most)
	if err != nil {
		t.Fatal(err)
	}
	return commits, more
}

func TestRangeCommitLines(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "a.txt", "one\ntwo\n", "first")
	base := gittest.Run(t, dir, "rev-parse", "HEAD")
	gittest.WriteFile(t, dir, "a.txt", "one\nTWO\nthree\n")
	gittest.WriteFile(t, dir, "b.go", "package b\n")
	gittest.Run(t, dir, "add", "-A")
	gittest.Run(t, dir, "commit", "-q", "-m", "edit and add: a | b")
	gittest.Run(t, dir, "commit", "-q", "--allow-empty", "-m", "empty")

	commits, more := rangeLines(t, dir, base, "HEAD", time.Time{}, 10)
	if more || len(commits) != 2 {
		t.Fatalf("commits = %+v, more %v", commits, more)
	}
	if commits[0].Subject != "empty" || len(commits[0].Files) != 0 {
		t.Errorf("newest = %+v", commits[0])
	}
	edit := commits[1]
	if edit.Subject != "edit and add: a | b" || len(edit.SHA) != 40 || time.Since(edit.Time) > time.Hour {
		t.Errorf("edit = %+v", edit)
	}
	got := map[string]NumstatEntry{}
	for _, f := range edit.Files {
		got[f.Path] = f
	}
	if a := got["a.txt"]; a.Added != 2 || a.Removed != 1 {
		t.Errorf("a.txt = %+v", a)
	}
	if b := got["b.go"]; b.Added != 1 || b.Removed != 0 {
		t.Errorf("b.go = %+v", b)
	}

	if all, _ := rangeLines(t, dir, "", "HEAD", time.Time{}, 10); len(all) != 3 || all[2].Files[0].Added != 2 {
		t.Errorf("from the first commit = %+v", all)
	}
	if capped, more := rangeLines(t, dir, "", "HEAD", time.Time{}, 2); !more || len(capped) != 2 {
		t.Errorf("capped = %d, more %v", len(capped), more)
	}
}

func TestRangeCommitLinesRename(t *testing.T) {
	dir := gittest.InitRepo(t)
	content := strings.Repeat("line\n", 20)
	gittest.CommitFile(t, dir, "old.txt", content, "first")
	gittest.Run(t, dir, "mv", "old.txt", "new.txt")
	gittest.WriteFile(t, dir, "new.txt", content+"more\n")
	gittest.Run(t, dir, "add", "-A")
	gittest.Run(t, dir, "commit", "-q", "-m", "move")

	commits, _ := rangeLines(t, dir, "HEAD~1", "HEAD", time.Time{}, 10)
	if len(commits) != 1 || len(commits[0].Files) != 1 {
		t.Fatalf("commits = %+v", commits)
	}
	if f := commits[0].Files[0]; f.Path != "new.txt" || f.OrigPath != "old.txt" || f.Added != 1 || f.Removed != 0 {
		t.Errorf("renamed = %+v", f)
	}
}

func TestRangeCommitLinesFirstParentAndSince(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitAt(t, dir, "a.txt", "a\n", "old", gittest.DaysAgo(10))
	gittest.Run(t, dir, "checkout", "-q", "-b", "side")
	gittest.CommitFile(t, dir, "side.txt", "s\n", "on the side")
	gittest.Run(t, dir, "checkout", "-q", "main")
	gittest.CommitFile(t, dir, "b.txt", "b\n", "new")
	gittest.Run(t, dir, "merge", "-q", "--no-ff", "-m", "merge side", "side")

	commits, _ := rangeLines(t, dir, "", "HEAD", time.Now().Add(-24*time.Hour), 10)
	if len(commits) != 1 || commits[0].Subject != "new" {
		t.Errorf("since a day ago = %+v", commits)
	}
}

func TestRangeCommitLinesEmpty(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "a.txt", "a\n", "first")

	if commits, more := rangeLines(t, dir, "HEAD", "HEAD", time.Time{}, 10); len(commits) != 0 || more {
		t.Errorf("empty range = %+v, more %v", commits, more)
	}
	if _, _, err := RangeCommitLines(context.Background(), dir, "nope", "HEAD", time.Time{}, 10); err == nil {
		t.Error("an unknown base should fail")
	}
}
