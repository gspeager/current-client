package git

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestParseNumstatFixture(t *testing.T) {
	output := "3\t1\ta.txt\x00-\t-\tbin.dat\x001\t0\t\x00old.txt\x00new.txt\x00"
	got, err := parseNumstat(output)
	if err != nil {
		t.Fatalf("parseNumstat: %v", err)
	}
	want := []NumstatEntry{
		{Path: "a.txt", Added: 3, Removed: 1},
		{Path: "bin.dat", Binary: true},
		{Path: "new.txt", OrigPath: "old.txt", Added: 1, Removed: 0},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestParseNumstatEmpty(t *testing.T) {
	got, err := parseNumstat("")
	if err != nil {
		t.Fatalf("parseNumstat: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %+v, want empty", got)
	}
}

func initNumstatRepo(t *testing.T) (dir string) {
	t.Helper()
	dir = gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "line1\nline2\nline3\n", "initial")
	return dir
}

func TestWorkingTreeNumstatReflectsUnstagedEdit(t *testing.T) {
	dir := initNumstatRepo(t)
	gittest.WriteFile(t, dir, "file.txt", "line1\nline2\nline3\nline4\n")

	got, err := WorkingTreeNumstat(context.Background(), dir)
	if err != nil {
		t.Fatalf("WorkingTreeNumstat: %v", err)
	}
	want := []NumstatEntry{{Path: "file.txt", Added: 1, Removed: 0}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestIndexNumstatReflectsStagedEdit(t *testing.T) {
	dir := initNumstatRepo(t)
	gittest.WriteFile(t, dir, "file.txt", "line1\nline2\n")
	gittest.Run(t, dir, "add", "file.txt")

	got, err := IndexNumstat(context.Background(), dir)
	if err != nil {
		t.Fatalf("IndexNumstat: %v", err)
	}
	want := []NumstatEntry{{Path: "file.txt", Added: 0, Removed: 1}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestCommitNumstatForRegularCommit(t *testing.T) {
	dir := initNumstatRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "line1\nline2\nline3\nline4\nline5\n", "add two lines")
	sha := gittest.Run(t, dir, "rev-parse", "HEAD")

	got, err := CommitNumstat(context.Background(), dir, sha)
	if err != nil {
		t.Fatalf("CommitNumstat: %v", err)
	}
	want := []NumstatEntry{{Path: "file.txt", Added: 2, Removed: 0}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestCommitNumstatForRootCommit(t *testing.T) {
	dir := initNumstatRepo(t)
	sha := gittest.Run(t, dir, "rev-parse", "HEAD")

	got, err := CommitNumstat(context.Background(), dir, sha)
	if err != nil {
		t.Fatalf("CommitNumstat: %v", err)
	}
	want := []NumstatEntry{{Path: "file.txt", Added: 3, Removed: 0}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestRefRangeNumstatTwoBranches(t *testing.T) {
	dir := initNumstatRepo(t)
	gittest.Run(t, dir, "checkout", "-q", "-b", "feature")
	gittest.CommitFile(t, dir, "file.txt", "line1\nline2\nline3\nline4\n", "feature edit")
	gittest.Run(t, dir, "checkout", "-q", "main")

	got, err := RefRangeNumstat(context.Background(), dir, "main", "feature")
	if err != nil {
		t.Fatalf("RefRangeNumstat: %v", err)
	}
	want := []NumstatEntry{{Path: "file.txt", Added: 1, Removed: 0}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestCommitNumstatFollowsRename(t *testing.T) {
	dir := initNumstatRepo(t)
	gittest.Run(t, dir, "mv", "file.txt", "renamed.txt")
	gittest.CommitFile(t, dir, "renamed.txt", "line1\nline2\nline3\nline4\n", "rename and edit")
	sha := gittest.Run(t, dir, "rev-parse", "HEAD")

	got, err := CommitNumstat(context.Background(), dir, sha)
	if err != nil {
		t.Fatalf("CommitNumstat: %v", err)
	}
	want := []NumstatEntry{{Path: "renamed.txt", OrigPath: "file.txt", Added: 1, Removed: 0}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestNumstatMarksBinaryFilesWithoutCounts(t *testing.T) {
	dir := initNumstatRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "bin.dat"), []byte{0x00, 0x01, 0x02}, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	gittest.Run(t, dir, "add", "bin.dat")

	got, err := IndexNumstat(context.Background(), dir)
	if err != nil {
		t.Fatalf("IndexNumstat: %v", err)
	}
	want := []NumstatEntry{{Path: "bin.dat", Binary: true}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
