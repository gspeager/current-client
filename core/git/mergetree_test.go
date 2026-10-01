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

// Git before 2.38 rejects --write-tree with a usage error, exit 129.
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
}
