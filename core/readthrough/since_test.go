package readthrough

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
	"github.com/gspeager/current-client/core/snapshot"
)

const feat = "refs/heads/feat"

func lines(from, to int) string {
	var text strings.Builder
	for i := from; i <= to; i++ {
		fmt.Fprintf(&text, "line %d\n", i)
	}
	return text.String()
}

// A branch that edits the end of long.txt and adds new.txt, all of it read.
func readRepo(t *testing.T) string {
	t.Helper()
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "long.txt", lines(1, 40), "base")
	gittest.Run(t, dir, "switch", "-q", "-c", "feat")
	gittest.CommitFile(t, dir, "long.txt", lines(1, 39)+"feat 40\n", "edit the end")
	gittest.CommitFile(t, dir, "new.txt", "new\n", "add")
	markAll(t, dir, rangeOf(t, dir, false))
	return dir
}

func markAll(t *testing.T, dir string, r Range) {
	t.Helper()
	files, _, err := Files(ctx, dir, r)
	if err != nil {
		t.Fatal(err)
	}
	if err := SetRead(ctx, dir, feat, r, files, true); err != nil {
		t.Fatal(err)
	}
}

// describe is each file's state in a word or two, for what isn't simply read.
func describe(t *testing.T, dir string, r Range) string {
	t.Helper()
	var all []string
	for _, file := range readAll(t, dir, feat, r).Files {
		state := "unread"
		switch {
		case file.Read:
			continue
		case file.Again:
			state = fmt.Sprintf("again +%d -%d", file.Since.Added, file.Since.Removed)
		}
		if file.Gone {
			state += " gone"
		}
		if file.Since.Conflict {
			state += " conflict"
		}
		all = append(all, file.Path+" "+state)
	}
	return strings.Join(all, ", ")
}

func expect(t *testing.T, dir string, r Range, want string) {
	t.Helper()
	if got := describe(t, dir, r); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func sinceLines(t *testing.T, dir string, r Range, path string) string {
	t.Helper()
	fd, err := SinceDiff(ctx, dir, feat, r, path, false)
	if err != nil {
		t.Fatal(err)
	}
	var changed []string
	for _, hunk := range fd.Hunks {
		for _, line := range strings.Split(hunk.Raw, "\n")[1:] {
			if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
				changed = append(changed, line)
			}
		}
	}
	return strings.Join(changed, " ")
}

func TestNothingIsNewUntilSomethingChanges(t *testing.T) {
	dir := readRepo(t)
	r := rangeOf(t, dir, false)
	expect(t, dir, r, "")
	if reading := readAll(t, dir, feat, r); reading.Marked != 2 || len(reading.Files) != 2 {
		t.Errorf("reading = %+v", reading)
	}
	if reading := readAll(t, dir, "refs/heads/other", r); reading.Marked != 0 {
		t.Errorf("another branch = %+v", reading)
	}
	if _, err := SinceDiff(ctx, dir, feat, r, "new.txt", false); err == nil {
		t.Error("a file that hasn't changed has no diff")
	}
}

func TestANewCommit(t *testing.T) {
	dir := readRepo(t)
	gittest.CommitFile(t, dir, "long.txt", "first\n"+lines(1, 39)+"feat 40\n", "edit again")
	gittest.CommitFile(t, dir, "another.txt", "x\n", "another")
	r := rangeOf(t, dir, false)
	expect(t, dir, r, "another.txt unread, long.txt again +1 -0")
	if got := sinceLines(t, dir, r, "long.txt"); got != "+first" {
		t.Errorf("since = %q", got)
	}
}

func TestAnAmend(t *testing.T) {
	dir := readRepo(t)
	gittest.Run(t, dir, "commit", "-q", "--amend", "-m", "only the message")
	expect(t, dir, rangeOf(t, dir, false), "")
	gittest.WriteFile(t, dir, "new.txt", "newer\n")
	gittest.Run(t, dir, "commit", "-q", "-a", "--amend", "--no-edit")
	gittest.Run(t, dir, "reflog", "expire", "--expire=now", "--all")
	gittest.Run(t, dir, "gc", "-q", "--prune=now")
	r := rangeOf(t, dir, false)
	expect(t, dir, r, "new.txt again +1 -1")
	if got := sinceLines(t, dir, r, "new.txt"); got != "-new +newer" {
		t.Errorf("since = %q", got)
	}
}

func rebaseOnto(t *testing.T, dir string, change func()) {
	t.Helper()
	gittest.Run(t, dir, "switch", "-q", "main")
	change()
	gittest.Run(t, dir, "switch", "-q", "feat")
	gittest.Run(t, dir, "rebase", "-q", "main")
}

func TestARebaseThatChangesNothing(t *testing.T) {
	dir := readRepo(t)
	rebaseOnto(t, dir, func() { gittest.CommitFile(t, dir, "elsewhere.txt", "x\n", "elsewhere") })
	expect(t, dir, rangeOf(t, dir, false), "")
}

func TestARebaseOntoABaseThatChangedTheSameFile(t *testing.T) {
	dir := readRepo(t)
	before := gittest.Run(t, dir, "count-objects")
	rebaseOnto(t, dir, func() { gittest.CommitFile(t, dir, "long.txt", "base 1\n"+lines(2, 40), "edit the start") })
	r := rangeOf(t, dir, false)
	expect(t, dir, r, "")

	objects := gittest.Run(t, dir, "count-objects")
	readAll(t, dir, feat, r)
	if after := gittest.Run(t, dir, "count-objects"); after != objects || objects == before {
		t.Errorf("reading left objects behind: %s, then %s", objects, after)
	}

	gittest.CommitFile(t, dir, "long.txt", "base 1\n"+lines(2, 39)+"feat 40\nfeat 41\n", "and the branch")
	r = rangeOf(t, dir, false)
	expect(t, dir, r, "long.txt again +1 -0")
	if got := sinceLines(t, dir, r, "long.txt"); got != "+feat 41" {
		t.Errorf("since = %q", got)
	}
	// Read again, its new base is what's kept.
	markAll(t, dir, r)
	expect(t, dir, r, "")
}

func TestMergingTheTargetIn(t *testing.T) {
	dir := readRepo(t)
	gittest.Run(t, dir, "switch", "-q", "main")
	gittest.CommitFile(t, dir, "long.txt", "base 1\n"+lines(2, 40), "edit the start")
	gittest.Run(t, dir, "switch", "-q", "feat")
	gittest.Run(t, dir, "merge", "-q", "--no-edit", "main")
	expect(t, dir, rangeOf(t, dir, false), "")
}

func TestARebaseThatConflicted(t *testing.T) {
	dir := readRepo(t)
	gittest.Run(t, dir, "switch", "-q", "main")
	gittest.CommitFile(t, dir, "long.txt", lines(1, 39)+"base 40\n", "edit the end too")
	gittest.Run(t, dir, "switch", "-q", "feat")
	gittest.Run(t, dir, "rebase", "-q", "-X", "theirs", "main")
	gittest.CommitFile(t, dir, "long.txt", lines(1, 39)+"both 40\n", "settle it")
	r := rangeOf(t, dir, false)
	expect(t, dir, r, "long.txt again +1 -1 conflict")
	if got := sinceLines(t, dir, r, "long.txt"); got != "-feat 40 +both 40" {
		t.Errorf("since = %q", got)
	}
}

func TestUncommittedEditsSince(t *testing.T) {
	dir := readRepo(t)
	gittest.WriteFile(t, dir, "new.txt", "new\nmore\n")
	gittest.WriteFile(t, dir, "untracked.txt", "x\n")
	r := rangeOf(t, dir, true)
	expect(t, dir, r, "new.txt again +1 -0, untracked.txt unread")
	if got := sinceLines(t, dir, r, "new.txt"); got != "+more" {
		t.Errorf("since = %q", got)
	}
	markAll(t, dir, r)
	expect(t, dir, r, "")
	// What was read was never committed, and is still there to compare with.
	gittest.WriteFile(t, dir, "new.txt", "new\nless\n")
	gittest.Run(t, dir, "gc", "-q", "--prune=now")
	if got := sinceLines(t, dir, r, "new.txt"); got != "-more +less" {
		t.Errorf("since = %q", got)
	}
}

func TestAChangeThatWasUndone(t *testing.T) {
	dir := readRepo(t)
	gittest.CommitFile(t, dir, "long.txt", lines(1, 40), "undo")
	gittest.Run(t, dir, "rm", "-q", "new.txt")
	gittest.Run(t, dir, "commit", "-q", "-m", "and the other")
	r := rangeOf(t, dir, false)
	expect(t, dir, r, "long.txt again +1 -1 gone, new.txt again +0 -1 gone")
	if got := sinceLines(t, dir, r, "long.txt"); got != "-feat 40 +line 40" {
		t.Errorf("since = %q", got)
	}
	if files, _, _ := Files(ctx, dir, r); len(files) != 0 {
		t.Errorf("files = %+v", files)
	}
}

func TestAChangeThatTheBaseTook(t *testing.T) {
	dir := readRepo(t)
	gittest.Run(t, dir, "switch", "-q", "main")
	gittest.Run(t, dir, "checkout", "-q", "feat", "--", "new.txt")
	gittest.Run(t, dir, "commit", "-q", "-m", "take new.txt")
	gittest.Run(t, dir, "switch", "-q", "feat")
	gittest.Run(t, dir, "rebase", "-q", "main")
	expect(t, dir, rangeOf(t, dir, false), "")
}

func TestARenamedFileOnANewBase(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "old.txt", lines(1, 40), "base")
	gittest.Run(t, dir, "switch", "-q", "-c", "feat")
	gittest.Run(t, dir, "mv", "old.txt", "new.txt")
	gittest.CommitFile(t, dir, "new.txt", lines(1, 39)+"feat 40\n", "move and edit")
	gittest.Run(t, dir, "commit", "-q", "--allow-empty", "-a", "-m", "the old one")
	markAll(t, dir, rangeOf(t, dir, false))
	rebaseOnto(t, dir, func() { gittest.CommitFile(t, dir, "old.txt", "base 1\n"+lines(2, 40), "edit the start") })
	expect(t, dir, rangeOf(t, dir, false), "")
}

func TestAMarkOutlivesSnapshots(t *testing.T) {
	dir := readRepo(t)
	for i := range 4 {
		gittest.WriteFile(t, dir, "scratch.txt", fmt.Sprint(i))
		if _, err := snapshot.Take(ctx, dir, "discard", 2); err != nil {
			t.Fatal(err)
		}
	}
	if kept, err := snapshot.List(ctx, dir); err != nil || len(kept) != 2 {
		t.Fatalf("snapshots = %v, %v", kept, err)
	}
	expect(t, dir, rangeOf(t, dir, false), "")
}

func TestChangesOfAFileAndAFolderOfItsName(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "base.txt", "x\n", "base")
	gittest.Run(t, dir, "switch", "-q", "-c", "feat")
	gittest.CommitFile(t, dir, "x", "a file\n", "file")
	markAll(t, dir, rangeOf(t, dir, false))
	gittest.Run(t, dir, "rm", "-q", "x")
	gittest.Run(t, dir, "commit", "-q", "-m", "gone")
	expect(t, dir, rangeOf(t, dir, false), "x again +0 -1 gone")
}

func TestAFileRenamedSinceItWasRead(t *testing.T) {
	dir := readRepo(t)
	gittest.Run(t, dir, "mv", "long.txt", "moved.txt")
	gittest.Run(t, dir, "commit", "-q", "-m", "move")
	expect(t, dir, rangeOf(t, dir, false), "")
	gittest.CommitFile(t, dir, "moved.txt", lines(1, 39)+"feat 40\nfeat 41\n", "and edit")
	r := rangeOf(t, dir, false)
	expect(t, dir, r, "moved.txt again +1 -0")
	if got := sinceLines(t, dir, r, "moved.txt"); got != "+feat 41" {
		t.Errorf("since = %q", got)
	}
	// Read under its new name, and only that.
	markAll(t, dir, r)
	expect(t, dir, r, "")
	marks, err := Marks(ctx, dir, feat)
	if err != nil || len(marks) != 2 {
		t.Errorf("marks = %v, %v", marks, err)
	}
}
