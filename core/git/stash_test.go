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

func TestStashChangedFilesListsTrackedAndUntracked(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "tracked.txt", "one\n", "initial")
	gittest.WriteFile(t, dir, "tracked.txt", "one\ntwo\n")
	gittest.WriteFile(t, dir, "new.txt", "a\nb\nc\n")
	ctx := context.Background()
	if err := StashSave(ctx, dir, "wip", true); err != nil {
		t.Fatalf("StashSave: %v", err)
	}

	files, err := StashChangedFiles(ctx, dir, 0)
	if err != nil {
		t.Fatalf("StashChangedFiles: %v", err)
	}
	want := []ChangedFile{{Status: "M", Path: "tracked.txt"}, {Status: "?", Path: "new.txt"}}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("got %+v, want %+v", files, want)
	}

	stats, err := StashNumstat(ctx, dir, 0)
	if err != nil {
		t.Fatalf("StashNumstat: %v", err)
	}
	added := map[string]int{}
	for _, s := range stats {
		added[s.Path] = s.Added
	}
	if added["tracked.txt"] != 1 || added["new.txt"] != 3 {
		t.Fatalf("added lines = %v, want tracked.txt 1 and new.txt 3", added)
	}
}

func TestStashChangedFilesWithoutUntracked(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "tracked.txt", "one\n", "initial")
	gittest.WriteFile(t, dir, "tracked.txt", "two\n")
	ctx := context.Background()
	if err := StashSave(ctx, dir, "", false); err != nil {
		t.Fatalf("StashSave: %v", err)
	}

	ref, err := stashUntrackedRef(ctx, dir, 0)
	if err != nil || ref != "" {
		t.Fatalf("stashUntrackedRef = %q, %v; want no untracked commit", ref, err)
	}
	files, err := StashChangedFiles(ctx, dir, 0)
	if err != nil {
		t.Fatalf("StashChangedFiles: %v", err)
	}
	if want := []ChangedFile{{Status: "M", Path: "tracked.txt"}}; !reflect.DeepEqual(files, want) {
		t.Fatalf("got %+v, want %+v", files, want)
	}
}

func TestStashPushOnlySomePaths(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "a.txt", "a1\n", "initial")
	gittest.CommitFile(t, dir, "b.txt", "b1\n", "add b")
	gittest.WriteFile(t, dir, "a.txt", "a2\n")
	gittest.WriteFile(t, dir, "b.txt", "b2\n")
	gittest.WriteFile(t, dir, "new.txt", "new\n")
	gittest.WriteFile(t, dir, "other.txt", "other\n")

	opts := StashOptions{Message: "part", IncludeUntracked: true, Paths: []string{"a.txt", "new.txt"}}
	if err := StashPush(context.Background(), dir, opts); err != nil {
		t.Fatalf("StashPush: %v", err)
	}

	if got := gittest.Run(t, dir, "status", "--porcelain"); got != "M b.txt\n?? other.txt" {
		t.Fatalf("status after stashing a.txt and new.txt = %q, want only b.txt and other.txt left", got)
	}
	files, err := StashChangedFiles(context.Background(), dir, 0)
	if err != nil {
		t.Fatalf("StashChangedFiles: %v", err)
	}
	if want := []ChangedFile{{Status: "M", Path: "a.txt"}, {Status: "?", Path: "new.txt"}}; !reflect.DeepEqual(files, want) {
		t.Fatalf("stash holds %+v, want %+v", files, want)
	}
}

func TestStashPushSomePathsLeavesOtherStagedFilesOut(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "notes.md", "one\n", "initial")
	gittest.WriteFile(t, dir, "notes.md", "two\n")
	gittest.WriteFile(t, dir, "staged.txt", "staged\n")
	gittest.Run(t, dir, "add", "staged.txt")
	ctx := context.Background()

	if err := StashPush(ctx, dir, StashOptions{Paths: []string{"notes.md"}}); err != nil {
		t.Fatalf("StashPush: %v", err)
	}

	if got := gittest.Run(t, dir, "status", "--porcelain"); got != "A  staged.txt" {
		t.Fatalf("status = %q, want only the staged staged.txt left", got)
	}
	files, err := StashChangedFiles(ctx, dir, 0)
	if err != nil {
		t.Fatalf("StashChangedFiles: %v", err)
	}
	if want := []ChangedFile{{Status: "M", Path: "notes.md"}}; !reflect.DeepEqual(files, want) {
		t.Fatalf("stash holds %+v, want %+v", files, want)
	}

	// Popping used to fail once a file stashed by mistake had changed.
	gittest.Run(t, dir, "rm", "-q", "--cached", "staged.txt")
	gittest.WriteFile(t, dir, "staged.txt", "changed\n")
	if err := StashPop(ctx, dir, 0); err != nil {
		t.Fatalf("StashPop: %v", err)
	}
	if got := gittest.Run(t, dir, "status", "--porcelain"); got != "M notes.md\n?? staged.txt" {
		t.Fatalf("status after pop = %q", got)
	}
}

func TestStashPushSomePathsTakesTheirStagedChanges(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "a.txt", "a1\n", "initial")
	gittest.WriteFile(t, dir, "a.txt", "a2\n")
	gittest.WriteFile(t, dir, "new.txt", "new\n")
	gittest.Run(t, dir, "add", "a.txt", "new.txt")
	gittest.WriteFile(t, dir, "a.txt", "a3\n")
	ctx := context.Background()

	if err := StashPush(ctx, dir, StashOptions{Paths: []string{"a.txt", "new.txt"}}); err != nil {
		t.Fatalf("StashPush: %v", err)
	}

	if got := gittest.Run(t, dir, "status", "--porcelain"); got != "" {
		t.Fatalf("status = %q, want everything stashed", got)
	}
	if got := gittest.Run(t, dir, "diff", "--name-status", "stash@{0}^1", "stash@{0}^2"); got != "M\ta.txt\nA\tnew.txt" {
		t.Fatalf("stash's staged changes = %q", got)
	}
	if got := gittest.Run(t, dir, "show", "stash@{0}:a.txt"); got != "a3" {
		t.Fatalf("stashed a.txt = %q, want the working tree's a3", got)
	}
}

func TestStashPushKeepIndexLeavesStagedChanges(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "a.txt", "a1\n", "initial")
	gittest.CommitFile(t, dir, "b.txt", "b1\n", "add b")
	gittest.WriteFile(t, dir, "a.txt", "a2\n")
	gittest.Run(t, dir, "add", "a.txt")
	gittest.WriteFile(t, dir, "b.txt", "b2\n")

	if err := StashPush(context.Background(), dir, StashOptions{KeepIndex: true}); err != nil {
		t.Fatalf("StashPush: %v", err)
	}

	if got := gittest.Run(t, dir, "status", "--porcelain"); got != "M  a.txt" {
		t.Fatalf("status = %q, want only the staged a.txt left", got)
	}
}
