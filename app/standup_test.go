package app

import (
	"os/exec"
	"testing"
	"time"
)

func TestStandupListsOwnCommitsOnEveryBranchSince(t *testing.T) {
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q")
	gitIn(t, repo, "config", "user.email", "Ada@Example.com")
	commit := func(email, date, message string) {
		cmd := exec.Command("git", "-C", repo, "-c", "user.name=X", "-c", "user.email="+email, "commit", "-q", "--allow-empty", "-m", message)
		cmd.Env = append(cmd.Environ(), "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	commit("ada@example.com", "2026-10-20T09:00:00Z", "old work")
	commit("ada@example.com", "2026-10-25T09:00:00Z", "feat: forms")
	commit("bob@example.com", "2026-10-25T10:00:00Z", "bob's work")
	gitIn(t, repo, "switch", "-q", "-c", "side")
	commit("ada@example.com", "2026-10-25T11:00:00Z", "fix: on a side branch")

	var inbox InboxService
	repos, err := inbox.Standup([]string{repo}, time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC).Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	var subjects []string
	for _, c := range repos[0].Commits {
		subjects = append(subjects, c.Subject)
	}
	if len(subjects) != 2 || subjects[0] != "fix: on a side branch" || subjects[1] != "feat: forms" {
		t.Fatalf("commits = %v", subjects)
	}
	if repos[0].HostEnabled {
		t.Fatal("a repository without a host can't have host rows")
	}
	if _, err := inbox.Standup([]string{repo}, "yesterday"); err == nil {
		t.Fatal("since should be a time")
	}
}
