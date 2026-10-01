package git

import (
	"context"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

// initMailmapRepo commits as two identities for one person; .mailmap joins
// them under the canonical one.
func initMailmapRepo(t *testing.T) string {
	t.Helper()
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "config", "user.name", "ada")
	gittest.Run(t, dir, "config", "user.email", "ada@old.example")
	gittest.CommitFile(t, dir, "a.txt", "one\n", "feat: first")
	gittest.Run(t, dir, "config", "user.name", "Ada Lovelace")
	gittest.Run(t, dir, "config", "user.email", "ada@example.com")
	gittest.CommitFile(t, dir, "a.txt", "one\ntwo\n", "fix: second")
	gittest.CommitFile(t, dir, ".mailmap", "Ada Lovelace <ada@example.com> ada <ada@old.example>\n", "chore: add mailmap")
	return dir
}

func TestAuthorsFollowMailmap(t *testing.T) {
	dir := initMailmapRepo(t)
	ctx := context.Background()
	want := Author{Name: "Ada Lovelace", Email: "ada@example.com"}

	history, err := History(ctx, dir, 10, 0, HistoryFilter{})
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	ranged, err := RangeHistory(ctx, dir, "", "HEAD", "")
	if err != nil {
		t.Fatalf("RangeHistory: %v", err)
	}
	for _, e := range append(history, ranged...) {
		if got := (Author{Name: e.AuthorName, Email: e.AuthorEmail}); got != want {
			t.Errorf("%s by %+v, want %+v", e.Subject, got, want)
		}
	}

	recent, err := RecentAuthors(ctx, dir)
	if err != nil {
		t.Fatalf("RecentAuthors: %v", err)
	}
	if len(recent) != 1 || recent[0] != want {
		t.Errorf("RecentAuthors = %+v, want only %+v", recent, want)
	}

	contributors, err := Contributors(ctx, dir, "", "")
	if err != nil {
		t.Fatalf("Contributors: %v", err)
	}
	if len(contributors) != 1 || contributors[0].Name != want.Name || contributors[0].Email != want.Email || contributors[0].Commits != 3 {
		t.Errorf("Contributors = %+v, want one entry for %+v with 3 commits", contributors, want)
	}

	blame, err := Blame(ctx, dir, "a.txt")
	if err != nil {
		t.Fatalf("Blame: %v", err)
	}
	for _, line := range blame {
		if got := (Author{Name: line.AuthorName, Email: line.AuthorEmail}); got != want {
			t.Errorf("blame line %d by %+v, want %+v", line.LineNo, got, want)
		}
	}
}

func TestHistoryAuthorFilterFollowsMailmap(t *testing.T) {
	dir := initMailmapRepo(t)
	gittest.Run(t, dir, "config", "log.mailmap", "false")

	entries, err := History(context.Background(), dir, 10, 0, HistoryFilter{Author: "Ada Lovelace"})
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("filtering by the canonical name found %d commits, want 3", len(entries))
	}
}
