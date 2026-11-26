package git

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gspeager/current-client/internal/gittest"
)

func linesByPath(t *testing.T, dir, rev string) map[string]FileLines {
	t.Helper()
	files, err := LineCounts(context.Background(), dir, rev)
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]FileLines{}
	for _, file := range files {
		byPath[file.Path] = file
	}
	return byPath
}

func TestLineCounts(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.WriteFile(t, dir, "ünï \"q\".txt", "one\ntwo\nthree")
	gittest.WriteFile(t, dir, "empty.txt", "")
	gittest.WriteFile(t, dir, "image.bin", "a\x00b\n")
	gittest.WriteFile(t, dir, "huge.txt", strings.Repeat("x\n", largestCounted/2+1))
	gittest.Run(t, dir, "add", "-A")
	gittest.Run(t, dir, "commit", "-q", "-m", "first")
	gittest.Run(t, dir, "tag", "v1")
	gittest.CommitFile(t, dir, "later.txt", "x\n", "later")

	files := linesByPath(t, dir, "v1")
	if f := files["ünï \"q\".txt"]; f.Lines != 3 || f.Binary {
		t.Errorf("text = %+v", f)
	}
	if f, ok := files["empty.txt"]; !ok || f.Lines != 0 || f.Binary {
		t.Errorf("empty = %+v, %v", f, ok)
	}
	if f := files["image.bin"]; !f.Binary {
		t.Errorf("binary = %+v", f)
	}
	if f := files["huge.txt"]; !f.Binary || f.Lines != 0 {
		t.Errorf("over the limit = %+v", f)
	}
	if _, ok := files["later.txt"]; ok || len(files) != 4 {
		t.Errorf("at the tag = %v", files)
	}
	if now := linesByPath(t, dir, "HEAD"); len(now) != 5 || now["later.txt"].Lines != 1 {
		t.Errorf("at HEAD = %v", now)
	}
}

func commitDaysAgo(t *testing.T, dir string, daysAgo int, name string) {
	t.Helper()
	when := gittest.DaysAgo(daysAgo)
	gittest.CommitAt(t, dir, name, when+name, "change "+name, when)
}

func TestFileCommitsOverPeriods(t *testing.T) {
	dir := gittest.InitRepo(t)
	commitDaysAgo(t, dir, 400, "a.txt")
	commitDaysAgo(t, dir, 60, "a.txt")
	commitDaysAgo(t, dir, 60, "sub ü.txt")
	commitDaysAgo(t, dir, 5, "a.txt")
	gittest.Run(t, dir, "switch", "-q", "-c", "side", "HEAD~1")
	commitDaysAgo(t, dir, 4, "side.txt")
	gittest.Run(t, dir, "switch", "-q", "main")
	gittest.Run(t, dir, "merge", "-q", "--no-ff", "-m", "merge", "side")
	gittest.Run(t, dir, "config", "log.showRoot", "false")

	files, more, err := FileCommits(context.Background(), dir, "HEAD", []int{30, 90, 0}, 100)
	if err != nil || more {
		t.Fatal(err, more)
	}
	counts := map[string][]int{}
	for _, file := range files {
		counts[file.Path] = file.Commits
		if file.Path == "a.txt" && time.Since(file.Last) > 6*24*time.Hour {
			t.Errorf("a.txt last changed %v", file.Last)
		}
	}
	for path, want := range map[string][]int{"a.txt": {1, 2, 3}, "sub ü.txt": {0, 1, 1}, "side.txt": {1, 1, 1}} {
		if !slices.Equal(counts[path], want) {
			t.Errorf("%s = %v, want %v", path, counts[path], want)
		}
	}

	files, more, err = FileCommits(context.Background(), dir, "HEAD", []int{0}, 2)
	if err != nil || !more {
		t.Fatal(err, more)
	}
	total := 0
	for _, file := range files {
		total += file.Commits[0]
	}
	if total != 2 {
		t.Errorf("counted %d commits' files, want 2", total)
	}
}
