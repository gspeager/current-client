package git

import (
	"context"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestCommitCreatesCommitWithMessage(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	gittest.WriteFile(t, dir, "new.txt", "hello")
	gittest.Run(t, dir, "add", "new.txt")

	if err := Commit(context.Background(), dir, "add new.txt", CommitOptions{}); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	cmd := exec.Command("git", "log", "-1", "--pretty=%s")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "add new.txt" {
		t.Fatalf("last commit message = %q, want %q", got, "add new.txt")
	}
}

func TestCommitRejectsNothingStaged(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	gittest.CommitFile(t, dir, "committed.txt", "v1", "initial")

	if err := Commit(context.Background(), dir, "empty commit", CommitOptions{}); err == nil {
		t.Fatal("expected an error when nothing is staged")
	}
}

func TestCommitAllowsEmptyWhenRequested(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	gittest.CommitFile(t, dir, "committed.txt", "v1", "initial")

	countBefore := commitCount(t, dir)

	if err := Commit(context.Background(), dir, "empty commit", CommitOptions{AllowEmpty: true}); err != nil {
		t.Fatalf("Commit (allowEmpty): %v", err)
	}

	if got := commitCount(t, dir); got != countBefore+1 {
		t.Fatalf("commit count = %d, want %d", got, countBefore+1)
	}
}

func TestCommitAmendReplacesLastCommitWithoutAddingOne(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	gittest.CommitFile(t, dir, "file.txt", "v1", "first message")

	countBefore := commitCount(t, dir)

	gittest.WriteFile(t, dir, "file.txt", "v2")
	gittest.Run(t, dir, "add", "file.txt")

	if err := Commit(context.Background(), dir, "amended message", CommitOptions{Amend: true}); err != nil {
		t.Fatalf("Commit (amend): %v", err)
	}

	if got := commitCount(t, dir); got != countBefore {
		t.Fatalf("commit count = %d, want unchanged at %d", got, countBefore)
	}

	cmd := exec.Command("git", "log", "-1", "--pretty=%s")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git log: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "amended message" {
		t.Fatalf("last commit message = %q, want %q", got, "amended message")
	}
}

func TestLastCommitMessage(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	gittest.CommitFile(t, dir, "file.txt", "v1", "the last message")

	got, err := LastCommitMessage(context.Background(), dir)
	if err != nil {
		t.Fatalf("LastCommitMessage: %v", err)
	}
	if got != "the last message" {
		t.Fatalf("LastCommitMessage = %q, want %q", got, "the last message")
	}
}

func TestLastCommitMessageNoCommitsYet(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")

	if _, err := LastCommitMessage(context.Background(), dir); err == nil {
		t.Fatal("expected an error with no commits yet")
	}
}

func TestUndoLastCommitKeepsChangesStaged(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")

	gittest.CommitFile(t, dir, "file.txt", "v2", "to be undone")
	countBefore := commitCount(t, dir)

	if err := UndoLastCommit(context.Background(), dir); err != nil {
		t.Fatalf("UndoLastCommit: %v", err)
	}

	if got := commitCount(t, dir); got != countBefore-1 {
		t.Fatalf("commit count = %d, want %d", got, countBefore-1)
	}

	statuses, err := GetStatus(context.Background(), dir)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if len(statuses) != 1 || statuses[0].Path != "file.txt" || statuses[0].IndexStatus != 'M' {
		t.Fatalf("got %+v, want file.txt staged as modified", statuses)
	}
}

func TestRecentAuthorsReturnsDistinctAuthorsMostRecentFirst(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	commitAs := func(name, email, file string) {
		gittest.Run(t, dir, "config", "user.email", email)
		gittest.Run(t, dir, "config", "user.name", name)
		gittest.WriteFile(t, dir, file, file)
		gittest.Run(t, dir, "add", file)
		gittest.Run(t, dir, "commit", "-m", file)
	}
	commitAs("Alice", "alice@example.com", "a.txt")
	commitAs("Bob", "bob@example.com", "b.txt")
	commitAs("Alice", "alice@example.com", "c.txt") // repeats Alice — must not duplicate
	commitAs("Carol", "carol@example.com", "d.txt")

	got, err := RecentAuthors(context.Background(), dir)
	if err != nil {
		t.Fatalf("RecentAuthors: %v", err)
	}
	want := []Author{
		{Name: "Carol", Email: "carol@example.com"},
		{Name: "Alice", Email: "alice@example.com"},
		{Name: "Bob", Email: "bob@example.com"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestRecentAuthorsNoCommitsYet(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")

	if _, err := RecentAuthors(context.Background(), dir); err == nil {
		t.Fatal("expected an error with no commits yet")
	}
}

func TestUndoLastCommitFailsOnRootCommit(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	gittest.CommitFile(t, dir, "file.txt", "v1", "only commit")

	if err := UndoLastCommit(context.Background(), dir); err == nil {
		t.Fatal("expected an error undoing the repository's only commit")
	}
}

func TestIsLastCommitPushedNoUpstream(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")

	pushed, err := IsLastCommitPushed(context.Background(), dir)
	if err != nil {
		t.Fatalf("IsLastCommitPushed: %v", err)
	}
	if pushed {
		t.Fatal("IsLastCommitPushed = true, want false with no upstream configured")
	}
}

func TestIsLastCommitPushedTracksUpstream(t *testing.T) {
	remoteDir := filepath.Join(t.TempDir(), "remote.git")
	gittest.Run(t, "", "init", "--bare", "-b", "main", remoteDir)

	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "remote", "add", "origin", remoteDir)
	gittest.CommitFile(t, dir, "file.txt", "v1", "first")
	gittest.Run(t, dir, "push", "-u", "origin", "main")

	pushed, err := IsLastCommitPushed(context.Background(), dir)
	if err != nil {
		t.Fatalf("IsLastCommitPushed: %v", err)
	}
	if !pushed {
		t.Fatal("IsLastCommitPushed = false, want true right after pushing")
	}

	gittest.CommitFile(t, dir, "file.txt", "v2", "second, not pushed")

	pushed, err = IsLastCommitPushed(context.Background(), dir)
	if err != nil {
		t.Fatalf("IsLastCommitPushed: %v", err)
	}
	if pushed {
		t.Fatal("IsLastCommitPushed = true, want false for a commit not yet pushed")
	}
}

func TestCommitWithSignFailsWithoutAConfiguredKey(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	gittest.WriteFile(t, dir, "file.txt", "v1")
	gittest.Run(t, dir, "add", "file.txt")

	// No signing key exists, so this only fails if -S reaches git.
	if err := Commit(context.Background(), dir, "signed", CommitOptions{Sign: true}); err == nil {
		t.Fatal("expected an error committing with sign=true and no signing key configured")
	}
}

func TestCommitWithoutSignOverridesConfiguredGpgsign(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	// Would fail signing (no key configured) if this weren't overridden.
	gittest.Run(t, dir, "config", "commit.gpgsign", "true")
	gittest.WriteFile(t, dir, "file.txt", "v1")
	gittest.Run(t, dir, "add", "file.txt")

	if err := Commit(context.Background(), dir, "unsigned", CommitOptions{}); err != nil {
		t.Fatalf("Commit (sign=false): %v, want --no-gpg-sign to override commit.gpgsign=true", err)
	}
}

func TestGPGSignDefaultReflectsLocalConfig(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")

	if GPGSignDefault(context.Background(), dir) {
		t.Fatal("GPGSignDefault = true, want false when commit.gpgsign is unset")
	}

	gittest.Run(t, dir, "config", "commit.gpgsign", "true")
	if !GPGSignDefault(context.Background(), dir) {
		t.Fatal("GPGSignDefault = false, want true when commit.gpgsign is set to true")
	}
}

func commitCount(t *testing.T, dir string) int {
	t.Helper()
	cmd := exec.Command("git", "rev-list", "--count", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-list: %v", err)
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatalf("parse commit count: %v", err)
	}
	return n
}
