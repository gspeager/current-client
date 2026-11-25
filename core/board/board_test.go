package board

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
	"github.com/gspeager/current-client/core/overlap"
)

// boardRepo is main with worktrees one and two, each on a branch of its name.
func boardRepo(t *testing.T) (main, one, two string) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	main = filepath.Join(root, "main")
	gittest.Run(t, root, "init", "-q", "-b", "main", main)
	for _, setting := range [][2]string{{"user.email", "test@example.com"}, {"user.name", "Test"}, {"core.autocrlf", "false"}} {
		gittest.Run(t, main, "config", setting[0], setting[1])
	}
	gittest.CommitFile(t, main, "shared.txt", "base\n", "base")
	gittest.CommitFile(t, main, "other.txt", "base\n", "other")
	one, two = filepath.Join(root, "one"), filepath.Join(root, "two")
	gittest.Run(t, main, "worktree", "add", "-q", "-b", "one", one)
	gittest.Run(t, main, "worktree", "add", "-q", "-b", "two", two)
	return main, one, two
}

func read(t *testing.T, r *Reader, dir string) Board {
	t.Helper()
	board, err := r.Read(context.Background(), dir)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return board
}

// "one: M shared.txt +1 -1, ? new.txt"
func columns(board Board) []string {
	var all []string
	for _, c := range board.Columns {
		var files []string
		for _, f := range c.Files {
			files = append(files, fmt.Sprintf("%s %s +%d -%d", f.Status[:1], f.Path, f.Added, f.Removed))
		}
		all = append(all, fmt.Sprintf("%s ↑%d ↓%d: %s%s", filepath.Base(c.Path), c.Ahead, c.Behind, strings.Join(files, ", "), c.Problem))
	}
	return all
}

func collisions(board Board) []string {
	var all []string
	for _, c := range board.Collisions {
		all = append(all, fmt.Sprintf("%s %s: %s", filepath.Base(board.Columns[c.A].Path), filepath.Base(board.Columns[c.B].Path), strings.Join(c.Files, " ")))
	}
	return all
}

func expect(t *testing.T, what string, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s = %q, want %q", what, got, want)
	}
}

func TestReadAChangeInEach(t *testing.T) {
	main, one, two := boardRepo(t)
	var r Reader
	expect(t, "clean columns", columns(read(t, &r, main)), []string{"main ↑0 ↓0: ", "one ↑0 ↓0: ", "two ↑0 ↓0: "})

	before := gittest.Run(t, main, "count-objects", "-v")
	index, err := os.ReadFile(filepath.Join(main, ".git", "worktrees", "one", "index"))
	if err != nil {
		t.Fatal(err)
	}
	gittest.WriteFile(t, main, "shared.txt", "from main\n")
	gittest.WriteFile(t, one, "shared.txt", "from one\n")
	gittest.WriteFile(t, one, "new.txt", "new\nlines\n")
	if err := os.Remove(filepath.Join(two, "other.txt")); err != nil {
		t.Fatal(err)
	}
	board := read(t, &r, main)
	expect(t, "columns", columns(board), []string{
		"main ↑0 ↓0: M shared.txt +1 -1",
		"one ↑0 ↓0: A new.txt +2 -0, M shared.txt +1 -1",
		"two ↑0 ↓0: D other.txt +0 -1",
	})
	expect(t, "collisions", collisions(board), []string{"main one: shared.txt"})
	if after := gittest.Run(t, main, "count-objects", "-v"); after != before {
		t.Errorf("objects after = %s, before = %s", after, before)
	}
	if after, _ := os.ReadFile(filepath.Join(main, ".git", "worktrees", "one", "index")); string(after) != string(index) {
		t.Error("reading the board changed a worktree's index")
	}

	// The same file again, at the same size and status.
	gittest.WriteFile(t, two, "shared.txt", "from two\n")
	expect(t, "collisions", collisions(read(t, &r, main)), []string{"main one: shared.txt", "main two: shared.txt", "one two: shared.txt"})
	gittest.WriteFile(t, two, "shared.txt", "from one\n")
	expect(t, "collisions once two agrees with one", collisions(read(t, &r, main)), []string{"main one: shared.txt", "main two: shared.txt"})
	gittest.Run(t, one, "checkout", "-q", "--", "shared.txt")
	expect(t, "collisions once one is undone", collisions(read(t, &r, main)), []string{"main two: shared.txt"})
}

func TestReadCountsCommits(t *testing.T) {
	main, one, _ := boardRepo(t)
	var r Reader
	read(t, &r, main)
	gittest.CommitFile(t, one, "shared.txt", "from one\n", "one")
	gittest.CommitFile(t, one, "a.txt", "a\n", "a")
	gittest.CommitFile(t, main, "shared.txt", "from main\n", "main")
	board := read(t, &r, main)
	expect(t, "columns", columns(board), []string{"main ↑0 ↓0: ", "one ↑2 ↓1: ", "two ↑0 ↓1: "})
	expect(t, "collisions of committed work", collisions(board), []string{"main one: shared.txt"})
	if board.Base != "main" {
		t.Errorf("base = %q", board.Base)
	}
}

func TestReadDoesNothingOverWhenNothingChanged(t *testing.T) {
	main, one, _ := boardRepo(t)
	gittest.WriteFile(t, one, "new.txt", "new\n")
	var r Reader
	first := read(t, &r, main)
	// What's returned is the board that was kept.
	if second := read(t, &r, main); &second.Columns[0] != &first.Columns[0] {
		t.Error("an unchanged board was read again")
	}
	if third := read(t, &r, one); third.Columns[1].Current != true || first.Columns[1].Current {
		t.Error("a board of another path isn't its own")
	}
}

func TestReadAWorktreeRemovedWhileOpen(t *testing.T) {
	main, one, two := boardRepo(t)
	var r Reader
	gittest.WriteFile(t, one, "shared.txt", "from one\n")
	gittest.WriteFile(t, two, "shared.txt", "from two\n")
	expect(t, "collisions", collisions(read(t, &r, main)), []string{"one two: shared.txt"})

	if err := os.RemoveAll(one); err != nil {
		t.Fatal(err)
	}
	board := read(t, &r, main)
	if len(board.Columns) != 3 || !board.Columns[1].Missing || board.Columns[1].Problem != "" || len(board.Collisions) != 0 {
		t.Errorf("with its folder gone: %+v", board)
	}
	gittest.Run(t, main, "worktree", "remove", "--force", two)
	gittest.Run(t, main, "worktree", "prune")
	expect(t, "columns", columns(read(t, &r, main)), []string{"main ↑0 ↓0: "})
}

func TestReadAWorktreeThatCantBeRead(t *testing.T) {
	main, one, two := boardRepo(t)
	gittest.WriteFile(t, two, "new.txt", "new\n")
	if err := os.WriteFile(filepath.Join(main, ".git", "worktrees", "one", "index"), []byte("not an index"), 0o600); err != nil {
		t.Fatal(err)
	}
	var r Reader
	board := read(t, &r, main)
	if board.Columns[1].Problem == "" || len(board.Columns[2].Files) != 1 {
		t.Errorf("columns = %q; want a problem in %s only", columns(board), one)
	}
}

func TestReadUntrackedFilesTooLargeToCount(t *testing.T) {
	main, one, _ := boardRepo(t)
	large, err := os.Create(filepath.Join(one, "large.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if err := large.Truncate(11 << 20); err != nil {
		t.Fatal(err)
	}
	large.Close()
	var r Reader
	if c := read(t, &r, main).Columns[1]; c.Skipped != 1 || len(c.Files) != 0 {
		t.Errorf("column = %+v", c)
	}
}

func TestReadDetachedAndWithoutCommits(t *testing.T) {
	main, one, _ := boardRepo(t)
	gittest.Run(t, one, "switch", "-q", "--detach")
	gittest.WriteFile(t, one, "shared.txt", "from one\n")
	var r Reader
	if c := read(t, &r, main).Columns[1]; c.Branch != "" || len(c.Files) != 1 {
		t.Errorf("detached: %+v", c)
	}

	gittest.Run(t, one, "switch", "-q", "--discard-changes", "--orphan", "fresh")
	gittest.WriteFile(t, one, "first.txt", "first\n")
	board := read(t, &r, main)
	expect(t, "columns", columns(board)[1:2], []string{"one ↑0 ↓0: A first.txt +1 -0"})

	empty := gittest.InitRepo(t)
	gittest.WriteFile(t, empty, "first.txt", "first\n")
	if board := read(t, &r, empty); len(board.Columns) != 1 || len(board.Columns[0].Files) != 1 {
		t.Errorf("a repository with no commits: %+v", board)
	}
}

func TestReadDuringAConflictedMerge(t *testing.T) {
	main, one, two := boardRepo(t)
	gittest.CommitFile(t, one, "shared.txt", "from one\n", "one")
	gittest.CommitFile(t, two, "shared.txt", "from two\n", "two")
	gittest.RunFails(t, one, "merge", "-q", "two")
	var r Reader
	board := read(t, &r, main)
	if c := board.Columns[1]; c.Problem != "" || len(c.Files) != 1 {
		t.Errorf("column = %+v", c)
	}
}

func TestReadAPartialClone(t *testing.T) {
	main, _, _ := boardRepo(t)
	gittest.Run(t, main, "config", "uploadpack.allowFilter", "true")
	clone := filepath.Join(t.TempDir(), "clone")
	gittest.Run(t, main, "clone", "-q", "--filter=blob:none", "file://"+filepath.ToSlash(main), clone)
	gittest.WriteFile(t, clone, "shared.txt", "edited\n")
	var r Reader
	board := read(t, &r, clone)
	if !errors.Is(board.Unpredicted, overlap.ErrPartialClone) || len(board.Columns[0].Files) != 1 {
		t.Errorf("board = %+v", board)
	}
}

func TestReadKeepsTheCurrentWorktreePastTheLimit(t *testing.T) {
	main, _, _ := boardRepo(t)
	var last string
	for i := range mostColumns {
		last = filepath.Join(filepath.Dir(main), fmt.Sprintf("more-%02d", i))
		gittest.Run(t, main, "worktree", "add", "-q", "--detach", last)
	}
	var r Reader
	board := read(t, &r, last)
	if len(board.Columns) != mostColumns || board.More != 3 || !board.Columns[mostColumns-1].Current {
		t.Errorf("%d columns, %d more, %q", len(board.Columns), board.More, columns(board))
	}
}
