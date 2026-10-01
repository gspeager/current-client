package git

import (
	"context"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestParseShortlogFixture(t *testing.T) {
	const output = "     2\tAlice Smith <a@b.com>\n     1\tBob Jones <bob@b.com>\n"
	got, err := parseShortlog(output)
	if err != nil {
		t.Fatalf("parseShortlog: %v", err)
	}
	want := []ContributorStat{
		{Name: "Alice Smith", Email: "a@b.com", Commits: 2},
		{Name: "Bob Jones", Email: "bob@b.com", Commits: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestParseShortlogEmpty(t *testing.T) {
	got, err := parseShortlog("")
	if err != nil {
		t.Fatalf("parseShortlog: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %+v, want empty", got)
	}
}

func TestContributorsMatchesShortlogRealRepo(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init", "-b", "main")
	commitAs := func(name, email, file string) {
		gittest.Run(t, dir, "config", "user.email", email)
		gittest.Run(t, dir, "config", "user.name", name)
		gittest.WriteFile(t, dir, file, file)
		gittest.Run(t, dir, "add", file)
		gittest.Run(t, dir, "commit", "-m", file)
	}
	commitAs("Alice", "alice@example.com", "a.txt")
	commitAs("Bob", "bob@example.com", "b.txt")
	commitAs("Alice", "alice@example.com", "c.txt")
	commitAs("Carol", "carol@example.com", "d.txt")
	commitAs("Alice", "alice@example.com", "e.txt")

	got, err := Contributors(context.Background(), dir, "", "")
	if err != nil {
		t.Fatalf("Contributors: %v", err)
	}

	cmd := exec.Command("git", "shortlog", "-sn", "-e", "HEAD")
	cmd.Dir = dir
	rawOut, err := cmd.Output()
	if err != nil {
		t.Fatalf("git shortlog: %v", err)
	}
	want, err := parseShortlog(string(rawOut))
	if err != nil {
		t.Fatalf("parseShortlog(real shortlog output): %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v (from real `git shortlog -sn -e`)", got, want)
	}
	if len(got) != 3 || got[0].Name != "Alice" || got[0].Commits != 3 {
		t.Fatalf("got %+v, want Alice first with 3 commits", got)
	}
}

func TestContributorsRespectsSinceUntil(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.WriteFile(t, dir, "file.txt", "v1")
	gittest.Run(t, dir, "add", "file.txt")
	oldCommit := exec.Command("git", "commit", "-m", "old commit")
	oldCommit.Dir = dir
	oldCommit.Env = append(os.Environ(), "GIT_AUTHOR_DATE=2020-01-01T00:00:00", "GIT_COMMITTER_DATE=2020-01-01T00:00:00")
	if out, err := oldCommit.CombinedOutput(); err != nil {
		t.Fatalf("git commit (backdated): %v\n%s", err, out)
	}
	gittest.CommitFile(t, dir, "file.txt", "v2", "recent commit")

	got, err := Contributors(context.Background(), dir, "2024-01-01", "")
	if err != nil {
		t.Fatalf("Contributors: %v", err)
	}
	if len(got) != 1 || got[0].Commits != 1 {
		t.Fatalf("got %+v, want a single contributor with 1 commit after the --since cutoff", got)
	}
}
