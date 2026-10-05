package git

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func initGrepRepo(t *testing.T) string {
	dir := gittest.InitRepo(t)
	if err := os.MkdirAll(filepath.Join(dir, "src dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	gittest.CommitFile(t, dir, "src dir/a.go", "func Foo() {}\nfoo := 1\n", "initial")
	gittest.WriteFile(t, dir, "untracked.txt", "food\n")
	gittest.WriteFile(t, dir, ".gitignore", "ignored.txt\n")
	gittest.WriteFile(t, dir, "ignored.txt", "foo\n")
	return dir
}

func TestGrepWorkingTreeIncludesUntrackedButNotIgnored(t *testing.T) {
	dir := initGrepRepo(t)

	got, truncated, err := Grep(context.Background(), dir, GrepOptions{Pattern: "foo", IgnoreCase: true})
	if err != nil || truncated {
		t.Fatalf("Grep = %v, %v", truncated, err)
	}
	want := []GrepMatch{
		{Path: "src dir/a.go", Line: 1, Text: "func Foo() {}"},
		{Path: "src dir/a.go", Line: 2, Text: "foo := 1"},
		{Path: "untracked.txt", Line: 1, Text: "food"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestGrepOptions(t *testing.T) {
	dir := initGrepRepo(t)
	ctx := context.Background()

	got, _, _ := Grep(ctx, dir, GrepOptions{Pattern: "foo", WholeWord: true})
	if len(got) != 1 || got[0].Text != "foo := 1" {
		t.Fatalf("whole word, case-sensitive = %+v, want only the foo := 1 line", got)
	}
	got, _, _ = Grep(ctx, dir, GrepOptions{Pattern: "F.o\\(", Regexp: true})
	if len(got) != 1 || got[0].Line != 1 {
		t.Fatalf("regexp = %+v, want func Foo()", got)
	}
	got, _, _ = Grep(ctx, dir, GrepOptions{Pattern: "F.o"})
	if len(got) != 0 {
		t.Fatalf("fixed text F.o matched %+v; want nothing", got)
	}
}

func TestGrepAtARevisionStripsItFromPaths(t *testing.T) {
	dir := initGrepRepo(t)
	first := gittest.Run(t, dir, "rev-parse", "HEAD")
	gittest.CommitFile(t, dir, "src dir/a.go", "nothing here\n", "remove foo")

	got, _, err := Grep(context.Background(), dir, GrepOptions{Pattern: "foo", Rev: first})
	if err != nil {
		t.Fatalf("Grep: %v", err)
	}
	if len(got) != 1 || got[0].Path != "src dir/a.go" || got[0].Line != 2 {
		t.Fatalf("got %+v, want line 2 of src dir/a.go at the first commit", got)
	}
}

func TestGrepNoMatchesTruncationAndBadPattern(t *testing.T) {
	dir := initGrepRepo(t)
	ctx := context.Background()

	if got, truncated, err := Grep(ctx, dir, GrepOptions{Pattern: "zzz"}); got != nil || truncated || err != nil {
		t.Fatalf("no matches = %v, %v, %v; want nothing and no error", got, truncated, err)
	}
	if got, truncated, _ := Grep(ctx, dir, GrepOptions{Pattern: "o", MaxMatches: 2}); len(got) != 2 || !truncated {
		t.Fatalf("capped at 2 = %d matches, truncated %v", len(got), truncated)
	}
	_, _, err := Grep(ctx, dir, GrepOptions{Pattern: "(", Regexp: true})
	if err == nil || !strings.HasPrefix(err.Error(), "Search failed: ") {
		t.Fatalf("bad pattern err = %v, want Git's reason", err)
	}
}
