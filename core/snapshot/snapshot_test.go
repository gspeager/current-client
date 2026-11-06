package snapshot

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/internal/gittest"
)

func newRepo(t *testing.T) string {
	t.Helper()
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "branch", "-M", "main")
	gittest.CommitFile(t, dir, "tracked.txt", "committed\n", "initial")
	return dir
}

func read(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func take(t *testing.T, dir, action string) bool {
	t.Helper()
	taken, err := Take(context.Background(), dir, action, 0)
	if err != nil {
		t.Fatal(err)
	}
	return taken
}

func newest(t *testing.T, dir string) Snapshot {
	t.Helper()
	snapshots, err := List(context.Background(), dir)
	if err != nil || len(snapshots) == 0 {
		t.Fatalf("List = %+v, %v", snapshots, err)
	}
	return snapshots[0]
}

func TestRestoresADiscardedEdit(t *testing.T) {
	dir := newRepo(t)
	gittest.WriteFile(t, dir, "tracked.txt", "my edit\n")
	if !take(t, dir, "discard") {
		t.Fatal("no snapshot taken")
	}
	gittest.Run(t, dir, "checkout", "--", "tracked.txt")
	if read(t, dir, "tracked.txt") != "committed\n" {
		t.Fatal("the discard didn't happen")
	}

	snapshot := newest(t, dir)
	if snapshot.Action != "discard" || snapshot.Branch != "main" || snapshot.Kind != KindTree {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	if err := Restore(context.Background(), dir, snapshot.ID, nil, 0); err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, "tracked.txt"); got != "my edit\n" {
		t.Fatalf("restored %q", got)
	}
}

func TestRestoresAnUntrackedFileLostToACheckout(t *testing.T) {
	dir := newRepo(t)
	gittest.Run(t, dir, "switch", "-q", "-c", "other")
	gittest.CommitFile(t, dir, "new.txt", "theirs\n", "adds new.txt")
	gittest.Run(t, dir, "switch", "-q", "main")
	gittest.WriteFile(t, dir, "new.txt", "my untracked work\n")
	if !take(t, dir, "checkout") {
		t.Fatal("no snapshot taken")
	}
	if err := os.Remove(filepath.Join(dir, "new.txt")); err != nil {
		t.Fatal(err)
	}
	gittest.Run(t, dir, "switch", "-q", "other")
	gittest.Run(t, dir, "switch", "-q", "main")

	if err := Restore(context.Background(), dir, newest(t, dir).ID, nil, 0); err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, "new.txt"); got != "my untracked work\n" {
		t.Fatalf("restored %q", got)
	}
}

func TestRestoresAfterItsBranchIsDeleted(t *testing.T) {
	dir := newRepo(t)
	gittest.Run(t, dir, "switch", "-q", "-c", "doomed")
	gittest.CommitFile(t, dir, "doomed.txt", "committed\n", "only on doomed")
	gittest.WriteFile(t, dir, "doomed.txt", "edited\n")
	take(t, dir, "reset")
	snapshot := newest(t, dir)
	gittest.Run(t, dir, "checkout", "--", "doomed.txt")
	gittest.Run(t, dir, "switch", "-q", "main")
	gittest.Run(t, dir, "branch", "-q", "-D", "doomed")
	gittest.Run(t, dir, "gc", "-q", "--prune=now")

	if err := Restore(context.Background(), dir, snapshot.ID, nil, 0); err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, "doomed.txt"); got != "edited\n" {
		t.Fatalf("restored %q", got)
	}
}

func TestRestoreOneFile(t *testing.T) {
	dir := newRepo(t)
	gittest.WriteFile(t, dir, "tracked.txt", "edit\n")
	gittest.WriteFile(t, dir, "other.txt", "other\n")
	take(t, dir, "discard")
	snapshot := newest(t, dir)
	gittest.Run(t, dir, "checkout", "--", "tracked.txt")
	if err := os.Remove(filepath.Join(dir, "other.txt")); err != nil {
		t.Fatal(err)
	}

	if err := Restore(context.Background(), dir, snapshot.ID, []string{"other.txt"}, 0); err != nil {
		t.Fatal(err)
	}
	if read(t, dir, "other.txt") != "other\n" || read(t, dir, "tracked.txt") != "committed\n" {
		t.Fatal("restored the wrong files")
	}
}

func TestRestoreBringsBackADeletionAndKeepsWhatItOverwrites(t *testing.T) {
	dir := newRepo(t)
	gittest.Run(t, dir, "rm", "-q", "tracked.txt")
	take(t, dir, "discard")
	snapshot := newest(t, dir)
	gittest.Run(t, dir, "reset", "-q", "--hard")
	gittest.WriteFile(t, dir, "tracked.txt", "newer work\n")

	if err := Restore(context.Background(), dir, snapshot.ID, nil, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "tracked.txt")); !os.IsNotExist(err) {
		t.Fatalf("the deletion wasn't restored: %v", err)
	}
	if got := newest(t, dir); got.Action != "restore" {
		t.Fatalf("what the restore overwrote should be saved: %+v", got)
	}
}

func TestSkipsNothingToSaveAndARepeat(t *testing.T) {
	dir := newRepo(t)
	if take(t, dir, "checkout") {
		t.Fatal("a clean tree needs no snapshot")
	}
	gittest.WriteFile(t, dir, "tracked.txt", "edit\n")
	if !take(t, dir, "checkout") {
		t.Fatal("first snapshot")
	}
	if take(t, dir, "checkout") {
		t.Fatal("the same tree again needs no snapshot")
	}
	gittest.WriteFile(t, dir, "tracked.txt", "another\n")
	if !take(t, dir, "checkout") {
		t.Fatal("a changed tree needs one")
	}
}

func TestLeavesIgnoredFilesOutAndNamesLargeOnes(t *testing.T) {
	dir := newRepo(t)
	gittest.CommitFile(t, dir, ".gitignore", "*.log\n", "ignore logs")
	gittest.WriteFile(t, dir, "debug.log", "ignored\n")
	gittest.WriteFile(t, dir, "small.txt", "small\n")
	gittest.WriteFile(t, dir, "big.bin", strings.Repeat("x", largestUntracked))
	take(t, dir, "discard")
	snapshot := newest(t, dir)
	if snapshot.Skipped != 1 {
		t.Fatalf("skipped = %d", snapshot.Skipped)
	}
	tree := gittest.Run(t, dir, "ls-tree", "--name-only", snapshot.SHA)
	if strings.Contains(tree, "debug.log") || strings.Contains(tree, "big.bin") || !strings.Contains(tree, "small.txt") {
		t.Fatalf("tree = %q", tree)
	}
}

func TestLimitsWhatUntrackedFilesAddUpTo(t *testing.T) {
	dir := newRepo(t)
	chunk := strings.Repeat("x", 6<<20)
	for i := range 20 {
		gittest.WriteFile(t, dir, "out"+string(rune('a'+i))+".bin", chunk)
	}
	take(t, dir, "checkout")
	snapshot := newest(t, dir)
	if snapshot.Skipped != 4 {
		t.Fatalf("skipped = %d", snapshot.Skipped)
	}
}

func TestPrunesToTheLimit(t *testing.T) {
	dir := newRepo(t)
	ctx := context.Background()
	for _, content := range []string{"a", "b", "c", "d"} {
		gittest.WriteFile(t, dir, "tracked.txt", content+"\n")
		if _, err := Take(ctx, dir, content, 2); err != nil {
			t.Fatal(err)
		}
	}
	snapshots, err := List(ctx, dir)
	if err != nil || len(snapshots) != 2 || snapshots[0].Action != "d" || snapshots[1].Action != "c" {
		t.Fatalf("snapshots = %+v, %v", snapshots, err)
	}
}

func TestStaysOutOfBranchesHistoryAndTheRealIndex(t *testing.T) {
	dir := newRepo(t)
	gittest.WriteFile(t, dir, "tracked.txt", "edit\n")
	gittest.WriteFile(t, dir, "untracked.txt", "u\n")
	before := gittest.Run(t, dir, "status", "--porcelain")
	take(t, dir, "discard")
	if after := gittest.Run(t, dir, "status", "--porcelain"); after != before {
		t.Fatalf("status changed: %q -> %q", before, after)
	}
	ctx := context.Background()
	history, err := git.History(ctx, dir, 100, 0, git.HistoryFilter{Ref: "--all"})
	if err != nil || len(history) != 1 {
		t.Fatalf("history = %+v, %v", history, err)
	}
	if branches := gittest.Run(t, dir, "branch", "-a"); strings.Contains(branches, "snapshot") {
		t.Fatalf("branches = %q", branches)
	}
	if err := git.DeleteHostRefs(ctx, dir, ""); err != nil {
		t.Fatal(err)
	}
	if snapshots, _ := List(ctx, dir); len(snapshots) != 1 {
		t.Fatal("clearing host heads took a snapshot")
	}
}

func TestTipKeepsCommitsAfterTheBranchGoes(t *testing.T) {
	dir := newRepo(t)
	gittest.Run(t, dir, "switch", "-q", "-c", "doomed")
	gittest.CommitFile(t, dir, "doomed.txt", "x\n", "only here")
	tip := gittest.Run(t, dir, "rev-parse", "HEAD")
	gittest.Run(t, dir, "switch", "-q", "main")
	ctx := context.Background()
	if err := TakeTip(ctx, dir, "branch delete", "doomed", tip, 0); err != nil {
		t.Fatal(err)
	}
	gittest.Run(t, dir, "branch", "-q", "-D", "doomed")
	gittest.Run(t, dir, "reflog", "expire", "--expire=now", "--all")
	gittest.Run(t, dir, "gc", "-q", "--prune=now")

	snapshot := newest(t, dir)
	if snapshot.Kind != KindTip || snapshot.Parent != tip || snapshot.Branch != "doomed" {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	if got := gittest.Run(t, dir, "log", "-1", "--format=%s", tip); got != "only here" {
		t.Fatalf("the commit is gone: %q", got)
	}
}

func TestDeleteAll(t *testing.T) {
	dir := newRepo(t)
	gittest.WriteFile(t, dir, "tracked.txt", "edit\n")
	take(t, dir, "discard")
	if err := DeleteAll(context.Background(), dir); err != nil {
		t.Fatal(err)
	}
	if got := gittest.Run(t, dir, "for-each-ref", "refs/current-client"); got != "" {
		t.Fatalf("refs = %q", got)
	}
}

func TestRestoreOfAPrunedSnapshot(t *testing.T) {
	if err := Restore(context.Background(), newRepo(t), "123", nil, 0); err != ErrNotFound {
		t.Fatalf("err = %v", err)
	}
}

func TestWorksBeforeTheFirstCommit(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.WriteFile(t, dir, "first.txt", "first\n")
	if !take(t, dir, "discard") {
		t.Fatal("no snapshot taken")
	}
	snapshot := newest(t, dir)
	if snapshot.Parent != "" {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	if err := os.Remove(filepath.Join(dir, "first.txt")); err != nil {
		t.Fatal(err)
	}
	if err := Restore(context.Background(), dir, snapshot.ID, nil, 0); err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, "first.txt"); got != "first\n" {
		t.Fatalf("restored %q", got)
	}
}
