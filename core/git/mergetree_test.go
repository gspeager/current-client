package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/gspeager/current-client/core/gitexec"
	"github.com/gspeager/current-client/internal/gittest"
)

func TestMergeConflicts(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "a.txt", "a\n", "base a")
	gittest.CommitFile(t, dir, "b.txt", "b\n", "base b")
	gittest.Run(t, dir, "switch", "-c", "clean")
	gittest.CommitFile(t, dir, "c.txt", "c\n", "unrelated file")
	gittest.Run(t, dir, "switch", "-c", "conflicting", "main")
	gittest.CommitFile(t, dir, "a.txt", "theirs a\n", "their a")
	gittest.CommitFile(t, dir, "b.txt", "theirs b\n", "their b")
	gittest.Run(t, dir, "switch", "main")
	gittest.CommitFile(t, dir, "a.txt", "ours a\n", "our a")
	gittest.CommitFile(t, dir, "b.txt", "ours b\n", "our b")
	ctx := context.Background()

	files, err := MergeConflicts(ctx, dir, "main", "conflicting")
	if err != nil || !reflect.DeepEqual(files, []string{"a.txt", "b.txt"}) {
		t.Errorf("conflicting = %v, %v; want [a.txt b.txt]", files, err)
	}
	if files, err := MergeConflicts(ctx, dir, "main", "clean"); err != nil || files != nil {
		t.Errorf("clean = %v, %v; want no conflicts", files, err)
	}
	if status := gittest.Run(t, dir, "status", "--porcelain"); status != "" {
		t.Errorf("working tree changed: %q", status)
	}
}

func TestMergeConflictsManyMatchesOneAtATime(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "a.txt", "a\n", "base a")
	gittest.CommitFile(t, dir, "b.txt", "b\n", "base b")
	gittest.Run(t, dir, "branch", "same")
	gittest.Run(t, dir, "switch", "-c", "clean")
	gittest.CommitFile(t, dir, "c.txt", "c\n", "unrelated file")
	gittest.Run(t, dir, "switch", "-c", "one", "main")
	gittest.CommitFile(t, dir, "b.txt", "theirs b\n", "their b")
	gittest.Run(t, dir, "switch", "-c", "two", "main")
	gittest.CommitFile(t, dir, "a.txt", "theirs a\n", "their a")
	gittest.CommitFile(t, dir, "b.txt", "theirs b too\n", "their b")
	gittest.Run(t, dir, "switch", "main")
	gittest.CommitFile(t, dir, "a.txt", "ours a\n", "our a")
	gittest.CommitFile(t, dir, "b.txt", "ours b\n", "our b")
	ctx := context.Background()

	tips := []string{"two", "clean", "one", "same", "two"}
	got, err := MergeConflictsMany(ctx, dir, "main", tips)
	if err != nil {
		t.Fatalf("MergeConflictsMany: %v", err)
	}
	for i, tip := range tips {
		want, err := MergeConflicts(ctx, dir, "main", tip)
		if err != nil {
			t.Fatalf("MergeConflicts %s: %v", tip, err)
		}
		if !reflect.DeepEqual(got[i], want) {
			t.Errorf("%s: batched %v, one at a time %v", tip, got[i], want)
		}
	}
	if got[0] == nil || got[1] != nil {
		t.Errorf("got %v, want two to conflict and clean not to", got)
	}
}

// Git before 2.38 rejects --write-tree with a usage error, exit 129, and Git
// 2.38 and 2.39 reject --stdin the same way.
func TestMergeConflictsOnOldGit(t *testing.T) {
	stub := t.TempDir()
	path, content := filepath.Join(stub, "git"), "#!/bin/sh\necho usage: git merge-tree >&2\nexit 129\n"
	if runtime.GOOS == "windows" {
		path, content = filepath.Join(stub, "git.bat"), "@echo usage: git merge-tree 1>&2\r\n@exit /b 129\r\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	gitexec.SetDefaultBinary(path)
	t.Cleanup(func() { gitexec.SetDefaultBinary("") })

	if _, err := MergeConflicts(context.Background(), t.TempDir(), "a", "b"); !errors.Is(err, ErrMergeTreeUnsupported) {
		t.Errorf("err = %v, want ErrMergeTreeUnsupported", err)
	}
	if _, err := MergeConflictsMany(context.Background(), t.TempDir(), "a", []string{"b"}); !errors.Is(err, ErrMergeTreeStdinUnsupported) {
		t.Errorf("batched err = %v, want ErrMergeTreeStdinUnsupported", err)
	}
}

func TestMergeConflictsWithUnrelatedHistory(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "a.txt", "a\n", "base")
	gittest.Run(t, dir, "switch", "-q", "--orphan", "gh-pages")
	gittest.CommitFile(t, dir, "index.html", "site\n", "site")
	gittest.Run(t, dir, "switch", "-q", "-c", "related", "main")
	gittest.CommitFile(t, dir, "b.txt", "b\n", "related")
	gittest.Run(t, dir, "switch", "-q", "main")
	ctx := context.Background()

	if _, err := MergeConflictsMany(ctx, dir, "main", []string{"related", "gh-pages"}); !errors.Is(err, ErrUnrelatedHistories) {
		t.Errorf("batched err = %v, want ErrUnrelatedHistories", err)
	}
	if _, err := MergeConflicts(ctx, dir, "main", "gh-pages"); !errors.Is(err, ErrUnrelatedHistories) {
		t.Errorf("single err = %v, want ErrUnrelatedHistories", err)
	}
	related, err := SharingHistory(ctx, dir, "main")
	if err != nil {
		t.Fatalf("SharingHistory: %v", err)
	}
	sha := func(rev string) string { return gittest.Run(t, dir, "rev-parse", rev) }
	if !related[sha("related")] || !related[sha("main")] || related[sha("gh-pages")] {
		t.Errorf("SharingHistory = %v, want related and main but not gh-pages", related)
	}
}

func TestMergePairsOneByOneMatchesTheBatch(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "a.txt", "a\n", "base")
	gittest.Run(t, dir, "switch", "-q", "-c", "one")
	gittest.CommitFile(t, dir, "a.txt", "one\n", "one")
	gittest.Run(t, dir, "switch", "-q", "-c", "two", "main")
	gittest.CommitFile(t, dir, "a.txt", "two\n", "two")
	gittest.Run(t, dir, "switch", "-q", "-c", "three", "main")
	gittest.CommitFile(t, dir, "b.txt", "three\n", "three")
	gittest.Run(t, dir, "switch", "-q", "main")
	ctx := context.Background()
	pairs := [][2]string{{"one", "two"}, {"two", "one"}, {"one", "three"}, {"main", "main"}, {"three", "two"}}

	batch, err := mergeTreeBatch(ctx, dir, nil, pairs)
	if err != nil {
		t.Fatal(err)
	}
	each, err := mergePairsEach(ctx, dir, nil, pairs)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(batch, each) {
		t.Fatalf("batch %+v != one by one %+v", batch, each)
	}
	if !reflect.DeepEqual(batch[0].Files, []string{"a.txt"}) || !batch[2].Clean || batch[2].Tree == "" || !batch[3].Clean {
		t.Errorf("merges = %+v", batch)
	}
	conflicts, err := MergeConflictsPairs(ctx, dir, pairs)
	if err != nil || !reflect.DeepEqual(conflicts, [][]string{{"a.txt"}, {"a.txt"}, nil, nil, nil}) {
		t.Errorf("MergeConflictsPairs = %v, %v", conflicts, err)
	}
}

func TestMergePairsAnswersAroundUnrelatedHistory(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "a.txt", "a\n", "base")
	gittest.Run(t, dir, "switch", "-q", "--orphan", "gh-pages")
	gittest.CommitFile(t, dir, "index.html", "site\n", "site")
	gittest.Run(t, dir, "switch", "-q", "-c", "related", "main")
	gittest.CommitFile(t, dir, "a.txt", "b\n", "related")
	gittest.Run(t, dir, "switch", "-q", "main")
	gittest.CommitFile(t, dir, "a.txt", "c\n", "ours")

	merges, err := MergePairs(context.Background(), dir, [][2]string{{"main", "related"}, {"main", "gh-pages"}, {"gh-pages", "main"}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(merges[0].Files, []string{"a.txt"}) || merges[0].Unrelated {
		t.Errorf("related = %+v, want a.txt conflicting", merges[0])
	}
	if !merges[1].Unrelated || !merges[2].Unrelated || merges[1].Tree != "" {
		t.Errorf("unrelated = %+v %+v", merges[1], merges[2])
	}
}

// Reading which branches conflict shouldn't grow the repository.
func TestMergingInMemoryWritesNoObjects(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "a.txt", "a\n", "base")
	gittest.Run(t, dir, "switch", "-c", "theirs")
	gittest.CommitFile(t, dir, "a.txt", "theirs\n", "their a")
	gittest.Run(t, dir, "switch", "main")
	gittest.CommitFile(t, dir, "a.txt", "ours\n", "our a")
	worktree := filepath.Join(t.TempDir(), "linked: worktree")
	gittest.Run(t, dir, "worktree", "add", "-q", "--detach", worktree)
	ctx := context.Background()
	before := gittest.Run(t, dir, "count-objects")

	for _, path := range []string{dir, worktree} {
		if files, err := MergeConflicts(ctx, path, "main", "theirs"); err != nil || len(files) != 1 {
			t.Errorf("MergeConflicts in %s = %v, %v", path, files, err)
		}
		if files, err := MergeConflictsMany(ctx, path, "main", []string{"theirs"}); err != nil || len(files[0]) != 1 {
			t.Errorf("MergeConflictsMany in %s = %v, %v", path, files, err)
		}
		if files, err := MergeConflictsPairs(ctx, path, [][2]string{{"main", "theirs"}}); err != nil || len(files[0]) != 1 {
			t.Errorf("MergeConflictsPairs in %s = %v, %v", path, files, err)
		}
		if merged, err := MergedInto(ctx, path, "main", []string{"theirs"}); err != nil || merged[0] {
			t.Errorf("MergedInto in %s = %v, %v", path, merged, err)
		}
	}
	if after := gittest.Run(t, dir, "count-objects"); after != before {
		t.Errorf("objects went from %s to %s", before, after)
	}

	// MergePairs' callers may read the tree it names.
	merges, err := MergePairs(ctx, dir, [][2]string{{"main", "theirs"}})
	if err != nil {
		t.Fatal(err)
	}
	gittest.Run(t, dir, "cat-file", "-e", merges[0].Tree)
}

func TestMergingInMemoryInFoldersGitMustUnquote(t *testing.T) {
	folders := []string{"no\u00a0break", "\u65e5\u672c"}
	if runtime.GOOS != "windows" {
		folders = append(folders, "a:b", `"quoted"`, `back\slash`)
	}
	for _, folder := range folders {
		dir := filepath.Join(t.TempDir(), folder)
		gittest.Run(t, t.TempDir(), "init", "-q", "-b", "main", dir)
		gittest.Run(t, dir, "config", "user.name", "t")
		gittest.Run(t, dir, "config", "user.email", "t@example.com")
		gittest.CommitFile(t, dir, "a.txt", "a\n", "base")
		gittest.Run(t, dir, "switch", "-c", "theirs")
		gittest.CommitFile(t, dir, "a.txt", "theirs\n", "their a")
		gittest.Run(t, dir, "switch", "main")
		gittest.CommitFile(t, dir, "a.txt", "ours\n", "our a")
		if files, err := MergeConflicts(context.Background(), dir, "main", "theirs"); err != nil || len(files) != 1 {
			t.Errorf("MergeConflicts in %q = %v, %v", folder, files, err)
		}
	}
}

func TestTemporaryObjectsLeftBehindAreRemoved(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("TMPDIR", temp)
	t.Setenv("TMP", temp)
	old, recent, other := filepath.Join(temp, scratchPrefix+"1"), filepath.Join(temp, scratchPrefix+"2"), filepath.Join(temp, "other")
	for _, dir := range []string{old, recent, other} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	lastWeek := time.Now().Add(-7 * 24 * time.Hour)
	for _, dir := range []string{old, other} {
		if err := os.Chtimes(dir, lastWeek, lastWeek); err != nil {
			t.Fatal(err)
		}
	}
	removeScratchOlderThan(24 * time.Hour)
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Errorf("the old folder is still there: %v", err)
	}
	for _, dir := range []string{recent, other} {
		if _, err := os.Stat(dir); err != nil {
			t.Error(err)
		}
	}
}

func TestAPartialCloneKeepsWhatItMerges(t *testing.T) {
	origin := gittest.InitRepo(t)
	gittest.CommitFile(t, origin, "a.txt", "a\n", "base")
	gittest.Run(t, origin, "config", "uploadpack.allowFilter", "true")
	clone := filepath.Join(t.TempDir(), "clone")
	gittest.Run(t, t.TempDir(), "clone", "-q", "--filter=blob:none", "file://"+filepath.ToSlash(origin), clone)

	env, discard := ScratchObjects(context.Background(), clone)
	defer discard()
	if env != nil {
		t.Errorf("env = %v, want none", env)
	}
	if env, discard := ScratchObjects(context.Background(), origin); env == nil {
		t.Error("a full repository has no temporary folder")
	} else {
		discard()
	}
}

func TestAPartialCloneThatBorrowsItsPacks(t *testing.T) {
	ctx := context.Background()
	origin := gittest.InitRepo(t)
	gittest.CommitFile(t, origin, "a.txt", "a\n", "base")
	gittest.Run(t, origin, "config", "uploadpack.allowFilter", "true")
	root, url := t.TempDir(), "file://"+filepath.ToSlash(origin)
	gittest.Run(t, root, "clone", "-q", "--filter=blob:none", url, "partial")
	gittest.Run(t, root, "clone", "-q", "--filter=blob:none", "--reference", "partial", url, "borrows")
	gittest.Run(t, root, "clone", "-q", "--reference", origin, url, "whole")
	for name, want := range map[string]bool{"partial": true, "borrows": true, "whole": false} {
		if got := PartialClone(ctx, filepath.Join(root, name)); got != want {
			t.Errorf("PartialClone(%s) = %v, want %v", name, got, want)
		}
	}
	// A folder can name itself.
	objects := filepath.Join(root, "whole", ".git", "objects")
	if err := os.WriteFile(filepath.Join(objects, "info", "alternates"), []byte("# itself\n\n.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if PartialClone(ctx, filepath.Join(root, "whole")) {
		t.Error("a whole clone is partial")
	}
}
