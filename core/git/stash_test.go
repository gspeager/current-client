package git

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestParseStashList(t *testing.T) {
	output := "stash@{0}\tOn main: wip feature\t1700000000\n" +
		"stash@{1}\tWIP on main: abc1234 message\t1699999000\n"
	got, err := parseStashList(output)
	if err != nil {
		t.Fatalf("parseStashList: %v", err)
	}
	want := []Stash{
		{Index: 0, Message: "On main: wip feature", Date: time.Unix(1700000000, 0)},
		{Index: 1, Message: "WIP on main: abc1234 message", Date: time.Unix(1699999000, 0)},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestParseStashListEmpty(t *testing.T) {
	got, err := parseStashList("")
	if err != nil {
		t.Fatalf("parseStashList: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %+v, want empty", got)
	}
}

func initStashRepo(t *testing.T) (dir string) {
	t.Helper()
	dir = gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1\n", "initial")
	return dir
}

func TestStashSaveAndListRealRepo(t *testing.T) {
	dir := initStashRepo(t)
	gittest.WriteFile(t, dir, "file.txt", "v2\n")

	if err := StashSave(context.Background(), dir, "my changes", false); err != nil {
		t.Fatalf("StashSave: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "file.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "v1\n" {
		t.Fatalf("file.txt = %q, want stash to restore the clean working tree", content)
	}

	stashes, err := ListStashes(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListStashes: %v", err)
	}
	if len(stashes) != 1 || stashes[0].Index != 0 {
		t.Fatalf("got %+v, want a single stash at index 0", stashes)
	}
	if want := "On main: my changes"; stashes[0].Message != want {
		t.Fatalf("Message = %q, want %q", stashes[0].Message, want)
	}
}

func TestStashSaveIncludesUntrackedWhenRequested(t *testing.T) {
	dir := initStashRepo(t)
	gittest.WriteFile(t, dir, "untracked.txt", "new\n")

	if err := StashSave(context.Background(), dir, "", true); err != nil {
		t.Fatalf("StashSave: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "untracked.txt")); !os.IsNotExist(err) {
		t.Fatalf("untracked.txt still present after stashing with includeUntracked, want it stashed away")
	}
}

func TestStashApplyRestoresChangesAndKeepsStash(t *testing.T) {
	dir := initStashRepo(t)
	gittest.WriteFile(t, dir, "file.txt", "v2\n")
	gittest.Run(t, dir, "stash", "push", "-m", "my changes")

	if err := StashApply(context.Background(), dir, 0); err != nil {
		t.Fatalf("StashApply: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "file.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "v2\n" {
		t.Fatalf("file.txt = %q, want the stashed change applied", content)
	}

	stashes, err := ListStashes(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListStashes: %v", err)
	}
	if len(stashes) != 1 {
		t.Fatalf("got %+v, want apply to keep the stash in the list", stashes)
	}
}

func TestStashPopRestoresChangesAndRemovesStash(t *testing.T) {
	dir := initStashRepo(t)
	gittest.WriteFile(t, dir, "file.txt", "v2\n")
	gittest.Run(t, dir, "stash", "push", "-m", "my changes")

	if err := StashPop(context.Background(), dir, 0); err != nil {
		t.Fatalf("StashPop: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "file.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "v2\n" {
		t.Fatalf("file.txt = %q, want the stashed change applied", content)
	}

	stashes, err := ListStashes(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListStashes: %v", err)
	}
	if len(stashes) != 0 {
		t.Fatalf("got %+v, want pop to remove the stash from the list", stashes)
	}
}

func TestStashDropRemovesStash(t *testing.T) {
	dir := initStashRepo(t)
	gittest.WriteFile(t, dir, "file.txt", "v2\n")
	gittest.Run(t, dir, "stash", "push", "-m", "first")
	gittest.WriteFile(t, dir, "file.txt", "v3\n")
	gittest.Run(t, dir, "stash", "push", "-m", "second")

	if err := StashDrop(context.Background(), dir, 0); err != nil {
		t.Fatalf("StashDrop: %v", err)
	}

	stashes, err := ListStashes(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListStashes: %v", err)
	}
	if len(stashes) != 1 || stashes[0].Message != "On main: first" {
		t.Fatalf("got %+v, want only the older stash (\"first\") to remain after dropping index 0", stashes)
	}
}
