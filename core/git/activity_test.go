package git

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func commitOnDate(t *testing.T, dir, file, date string) {
	t.Helper()
	gittest.WriteFile(t, dir, file, date)
	gittest.Run(t, dir, "add", file)
	cmd := exec.Command("git", "commit", "-m", "commit on "+date)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE="+date+"T12:00:00",
		"GIT_COMMITTER_DATE="+date+"T12:00:00",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit (backdated): %v\n%s", err, out)
	}
}

func TestCommitActivityCountsPerDayAndFillsGaps(t *testing.T) {
	dir := gittest.InitRepo(t)

	today := time.Now()
	twoDaysAgo := today.AddDate(0, 0, -2).Format("2006-01-02")
	todayStr := today.Format("2006-01-02")

	commitOnDate(t, dir, "a.txt", twoDaysAgo)
	commitOnDate(t, dir, "b.txt", todayStr)
	commitOnDate(t, dir, "c.txt", todayStr)

	got, err := CommitActivity(context.Background(), dir, 5)
	if err != nil {
		t.Fatalf("CommitActivity: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("got %d days, want 5", len(got))
	}
	if got[len(got)-1].Date != todayStr || got[len(got)-1].Commits != 2 {
		t.Fatalf("last day = %+v, want {%s 2}", got[len(got)-1], todayStr)
	}
	byDate := make(map[string]int, len(got))
	for _, d := range got {
		byDate[d.Date] = d.Commits
	}
	if byDate[twoDaysAgo] != 1 {
		t.Fatalf("got %d commits on %s, want 1", byDate[twoDaysAgo], twoDaysAgo)
	}
	// A day with no commits still appears, at zero, rather than being omitted.
	yesterday := today.AddDate(0, 0, -1).Format("2006-01-02")
	if count, ok := byDate[yesterday]; !ok || count != 0 {
		t.Fatalf("got %d, %v for %s, want 0, true", count, ok, yesterday)
	}
}

func TestCommitActivityIncludesAllBranches(t *testing.T) {
	dir := gittest.InitRepo(t)
	todayStr := time.Now().Format("2006-01-02")
	commitOnDate(t, dir, "main.txt", todayStr)
	gittest.Run(t, dir, "checkout", "-q", "-b", "feature")
	commitOnDate(t, dir, "feature.txt", todayStr)

	got, err := CommitActivity(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("CommitActivity: %v", err)
	}
	if len(got) != 1 || got[0].Commits != 2 {
		t.Fatalf("got %+v, want a single day with both branches' commits counted", got)
	}
}

func TestCommitActivityAndChurnLeaveOutStashes(t *testing.T) {
	dir := gittest.InitRepo(t)
	todayStr := time.Now().Format("2006-01-02")
	commitOnDate(t, dir, "main.txt", todayStr)
	gittest.WriteFile(t, dir, "main.txt", "work in progress")
	gittest.WriteFile(t, dir, "scratch.txt", "untracked")
	gittest.Run(t, dir, "stash", "push", "-u")

	activity, err := CommitActivity(context.Background(), dir, 1)
	if err != nil {
		t.Fatalf("CommitActivity: %v", err)
	}
	if len(activity) != 1 || activity[0].Commits != 1 {
		t.Fatalf("activity = %+v, want only the one real commit", activity)
	}

	churn, err := ComputeFileChurn(context.Background(), dir, 1, 10)
	if err != nil {
		t.Fatalf("ComputeFileChurn: %v", err)
	}
	for _, c := range churn {
		if c.Path == "scratch.txt" || c.Changes != 1 {
			t.Fatalf("churn = %+v, want main.txt once and nothing from the stash", churn)
		}
	}
}
