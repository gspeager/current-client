package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/gspeager/current-client/core/gitexec"
	"github.com/gspeager/current-client/core/internal/gittest"
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
