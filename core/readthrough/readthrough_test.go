package readthrough

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/filekind"
	"github.com/gspeager/current-client/core/internal/gittest"
)

var ctx = context.Background()

func repoWithBranch(t *testing.T) string {
	t.Helper()
	dir := gittest.InitRepo(t)
	gittest.WriteFile(t, dir, "renamed.txt", "one\ntwo\nthree\nfour\nfive\nsix\n")
	gittest.WriteFile(t, dir, "deleted.txt", "gone\n")
	gittest.Run(t, dir, "add", "-A")
	gittest.CommitFile(t, dir, "a.go", "package a\n", "base")
	gittest.Run(t, dir, "switch", "-q", "-c", "feat")
	gittest.CommitFile(t, dir, "a.go", "package a\n\nvar A = 1\n", "a")
	gittest.CommitFile(t, dir, "a_test.go", "package a\n", "test")
	gittest.Run(t, dir, "mv", "renamed.txt", "moved.txt")
	gittest.WriteFile(t, dir, "moved.txt", "one\ntwo\nthree\nfour\nfive\nsix\nseven\n")
	gittest.Run(t, dir, "rm", "-q", "deleted.txt")
	gittest.Run(t, dir, "add", "-A")
	gittest.Run(t, dir, "commit", "-q", "-m", "move and delete")
	// Main moves on, which isn't the branch's doing.
	gittest.Run(t, dir, "switch", "-q", "main")
	gittest.CommitFile(t, dir, "main.txt", "main\n", "main moves on")
	gittest.Run(t, dir, "switch", "-q", "feat")
	return dir
}

func rangeOf(t *testing.T, dir string, uncommitted bool) Range {
	t.Helper()
	base, err := Base(ctx, dir, "main", "HEAD", "")
	if err != nil {
		t.Fatal(err)
	}
	return Range{Base: base, Head: gittest.Run(t, dir, "rev-parse", "HEAD"), Uncommitted: uncommitted}
}

func filesOf(t *testing.T, dir string, r Range) map[string]File {
	t.Helper()
	files, _, err := Files(ctx, dir, r)
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]File{}
	for _, file := range files {
		byPath[file.Path] = file
	}
	return byPath
}

func list(files map[string]File) []File {
	var all []File
	for _, file := range files {
		all = append(all, file)
	}
	return all
}

func readPaths(t *testing.T, dir, key string, files map[string]File) map[string]bool {
	t.Helper()
	marks, err := Marks(ctx, dir, key)
	if err != nil {
		t.Fatal(err)
	}
	read := map[string]bool{}
	for path, file := range files {
		if IsRead(marks, file) {
			read[path] = true
		}
	}
	return read
}

func TestFilesAgainstTheMergeBase(t *testing.T) {
	dir := repoWithBranch(t)
	files := filesOf(t, dir, rangeOf(t, dir, false))
	if len(files) != 4 {
		t.Fatalf("files = %v", files)
	}
	if f := files["a.go"]; f.Status != "M" || f.Added != 2 || f.Kind != filekind.Code || f.Old.Blob == "" || f.New.Blob == "" {
		t.Errorf("a.go = %+v", f)
	}
	if f := files["a_test.go"]; f.Status != "A" || f.Kind != filekind.Test || f.Old.Blob != "" {
		t.Errorf("a_test.go = %+v", f)
	}
	if f := files["moved.txt"]; f.Status[0] != 'R' || f.OrigPath != "renamed.txt" || f.Added != 1 || f.Removed != 0 {
		t.Errorf("moved.txt = %+v", f)
	}
	if f := files["deleted.txt"]; f.Status != "D" || f.New.Blob != "" {
		t.Errorf("deleted.txt = %+v", f)
	}
}

func TestFilesWithAWorktreesUncommittedChanges(t *testing.T) {
	dir := repoWithBranch(t)
	worktree := filepath.Join(t.TempDir(), "linked")
	gittest.Run(t, dir, "worktree", "add", "-q", "--detach", worktree, "feat")
	gittest.WriteFile(t, worktree, "a.go", "package a\n\nvar A = 2\n")
	gittest.WriteFile(t, worktree, "untracked.go", "package a\n")
	gittest.WriteFile(t, worktree, "staged.go", "package a\n")
	gittest.Run(t, worktree, "add", "staged.go")
	if err := os.Remove(filepath.Join(worktree, "a_test.go")); err != nil {
		t.Fatal(err)
	}
	objects := gittest.Run(t, dir, "count-objects")
	status := gittest.Run(t, worktree, "status", "--porcelain")

	r := rangeOf(t, worktree, true)
	files := filesOf(t, worktree, r)
	for _, path := range []string{"a.go", "untracked.go", "staged.go", "moved.txt", "deleted.txt"} {
		if _, ok := files[path]; !ok {
			t.Errorf("%s isn't listed", path)
		}
	}
	// Added by the branch, then deleted again.
	if _, ok := files["a_test.go"]; ok || len(files) != 5 {
		t.Errorf("files = %v", files)
	}
	if after := gittest.Run(t, dir, "count-objects"); after != objects {
		t.Errorf("listing left objects behind: %s, then %s", objects, after)
	}
	if after := gittest.Run(t, worktree, "status", "--porcelain"); after != status {
		t.Errorf("status went from %q to %q", status, after)
	}
	if committed := filesOf(t, worktree, Range{Base: r.Base, Head: r.Head}); len(committed) != 4 {
		t.Errorf("committed only = %v", committed)
	}

	for path, want := range map[string]string{"untracked.go": "package a", "staged.go": "package a", "a.go": "var A = 2"} {
		diff, err := FileDiff(ctx, worktree, r, "", path, false)
		if err != nil || len(diff.Hunks) != 1 || !strings.Contains(diff.Hunks[0].Raw, "+"+want) {
			t.Errorf("diff of %s = %+v, %v", path, diff, err)
		}
	}
}

func TestDiffOfAFileRenamedInTheWorkingTree(t *testing.T) {
	dir := repoWithBranch(t)
	if err := os.Rename(filepath.Join(dir, "a.go"), filepath.Join(dir, "b.go")); err != nil {
		t.Fatal(err)
	}
	gittest.WriteFile(t, dir, "b.go", "package a\n\nvar A = 1\nvar B = 2\n")
	r := Range{Base: gittest.Run(t, dir, "rev-parse", "HEAD"), Head: gittest.Run(t, dir, "rev-parse", "HEAD"), Uncommitted: true}
	file := filesOf(t, dir, r)["b.go"]
	if file.OrigPath != "a.go" {
		t.Fatalf("file = %+v, want a rename", file)
	}
	diff, err := FileDiff(ctx, dir, r, file.OrigPath, file.Path, false)
	if err != nil || len(diff.Hunks) != 1 || len(diff.Hunks[0].Lines) != 4 {
		t.Errorf("diff = %+v, %v", diff, err)
	}
}

func TestDiffOfACommittedRenameShowsOnlyTheEdit(t *testing.T) {
	dir := repoWithBranch(t)
	diff, err := FileDiff(ctx, dir, rangeOf(t, dir, false), "renamed.txt", "moved.txt", false)
	if err != nil || len(diff.Hunks) != 1 || strings.Count(diff.Hunks[0].Raw, "\n+") != 1 {
		t.Errorf("diff = %+v, %v", diff, err)
	}
}

func TestALayerIsComparedWithWhatItWasBuiltOn(t *testing.T) {
	dir := repoWithBranch(t)
	built := gittest.Run(t, dir, "rev-parse", "HEAD")
	gittest.Run(t, dir, "switch", "-q", "-c", "layer")
	gittest.CommitFile(t, dir, "layer.go", "package a\n", "layer")
	gittest.Run(t, dir, "switch", "-q", "feat")
	gittest.Run(t, dir, "commit", "-q", "--amend", "-m", "amended")
	if base, err := Base(ctx, dir, "feat", "layer", built); err != nil || base != built {
		t.Errorf("base = %s, %v, want %s", base, err, built)
	}
	if base, err := Base(ctx, dir, "feat", "layer", ""); err != nil || base == built {
		t.Errorf("without a record, base = %s, %v", base, err)
	}
	// Merging the parent in since makes the merge base the later of the two.
	gittest.Run(t, dir, "switch", "-q", "layer")
	gittest.Run(t, dir, "merge", "-q", "-X", "ours", "-m", "merge", "feat")
	if base, err := Base(ctx, dir, "feat", "layer", gittest.Run(t, dir, "rev-parse", "main~1")); err != nil || base != gittest.Run(t, dir, "rev-parse", "feat") {
		t.Errorf("after a merge, base = %s, %v", base, err)
	}
}

func TestReadingAFileAndItChangingAgain(t *testing.T) {
	dir := repoWithBranch(t)
	const key = "refs/heads/feat"
	r := rangeOf(t, dir, false)
	files := filesOf(t, dir, r)
	if err := SetRead(ctx, dir, key, r, list(files), true); err != nil {
		t.Fatal(err)
	}
	if read := readPaths(t, dir, key, files); len(read) != 4 {
		t.Fatalf("read = %v", read)
	}
	if err := SetRead(ctx, dir, key, r, []File{files["moved.txt"]}, false); err != nil {
		t.Fatal(err)
	}
	if read := readPaths(t, dir, key, files); len(read) != 3 || read["moved.txt"] {
		t.Fatalf("after unmarking one, read = %v", read)
	}
	if read := readPaths(t, dir, "refs/heads/other", files); len(read) != 0 {
		t.Errorf("another branch has %v", read)
	}

	gittest.CommitFile(t, dir, "a.go", "package a\n\nvar A = 3\n", "change again")
	r = rangeOf(t, dir, false)
	if read := readPaths(t, dir, key, filesOf(t, dir, r)); read["a.go"] || !read["deleted.txt"] || !read["a_test.go"] {
		t.Errorf("after a change, read = %v", read)
	}

	// A rebase that leaves a file as it was leaves its mark.
	gittest.Run(t, dir, "rebase", "-q", "main")
	r = rangeOf(t, dir, false)
	if read := readPaths(t, dir, key, filesOf(t, dir, r)); read["a.go"] || !read["deleted.txt"] || !read["a_test.go"] {
		t.Errorf("after a rebase, read = %v", read)
	}
}

func TestReadingUncommittedChanges(t *testing.T) {
	dir := repoWithBranch(t)
	const key = "refs/heads/feat"
	gittest.WriteFile(t, dir, "a.go", "package a\n\nvar A = 2\n")
	gittest.WriteFile(t, dir, "new.go", "package a\n")
	r := rangeOf(t, dir, true)
	files := filesOf(t, dir, r)
	// Changed after it was listed, so what was read isn't what's there.
	gittest.WriteFile(t, dir, "new.go", "package a\n\nvar B = 1\n")
	if err := SetRead(ctx, dir, key, r, list(files), true); err != nil {
		t.Fatal(err)
	}
	if read := readPaths(t, dir, key, filesOf(t, dir, r)); read["new.go"] || !read["a.go"] {
		t.Errorf("read = %v", read)
	}
	marks, err := Marks(ctx, dir, key)
	if err != nil {
		t.Fatal(err)
	}
	gittest.Run(t, dir, "checkout", "-q", "a.go")
	gittest.Run(t, dir, "gc", "-q", "--prune=now")
	if content := gittest.Run(t, dir, "cat-file", "blob", marks["a.go"].Read.Blob); !strings.Contains(content, "var A = 2") {
		t.Errorf("what was read is %q", content)
	}
	if gittest.Run(t, dir, "cat-file", "blob", marks["a.go"].Base.Blob) != "package a" {
		t.Error("its base wasn't kept")
	}
}

func TestMarksOfAFileWhoseFolderWasOneOfThem(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "base.txt", "base\n", "base")
	gittest.Run(t, dir, "switch", "-q", "-c", "feat")
	gittest.CommitFile(t, dir, "x", "a file\n", "file")
	r := rangeOf(t, dir, false)
	if err := SetRead(ctx, dir, "k", r, list(filesOf(t, dir, r)), true); err != nil {
		t.Fatal(err)
	}
	gittest.Run(t, dir, "rm", "-q", "x")
	if err := os.Mkdir(filepath.Join(dir, "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	gittest.WriteFile(t, dir, "x/ünï q.txt", "a folder now\n")
	gittest.Run(t, dir, "add", "-A")
	gittest.Run(t, dir, "commit", "-q", "-m", "folder")
	r = rangeOf(t, dir, false)
	files := filesOf(t, dir, r)
	if err := SetRead(ctx, dir, "k", r, list(files), true); err != nil {
		t.Fatal(err)
	}
	if read := readPaths(t, dir, "k", files); len(read) != 1 {
		t.Errorf("read = %v of %v", read, files)
	}
}

func TestForgettingAndRenamingMarks(t *testing.T) {
	dir := repoWithBranch(t)
	r := rangeOf(t, dir, false)
	files := filesOf(t, dir, r)
	for _, key := range []string{"refs/heads/feat", "refs/heads/gone"} {
		if err := SetRead(ctx, dir, key, r, list(files), true); err != nil {
			t.Fatal(err)
		}
	}
	if err := Rename(ctx, dir, "refs/heads/feat", "refs/heads/feat/nested"); err != nil {
		t.Fatal(err)
	}
	if err := Forget(ctx, dir, func(key string) bool { return key == "refs/heads/gone" }); err != nil {
		t.Fatal(err)
	}
	marks, err := allMarks(ctx, dir)
	if err != nil || len(marks) != 1 || marks[markRef("refs/heads/feat/nested")] != "refs/heads/feat/nested" {
		t.Errorf("marks = %v, %v", marks, err)
	}
	if read := readPaths(t, dir, "refs/heads/feat/nested", files); len(read) != 4 {
		t.Errorf("read = %v", read)
	}
}
