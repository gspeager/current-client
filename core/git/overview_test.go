package git

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func commitAt(t *testing.T, dir, message, date string) {
	t.Helper()
	cmd := exec.Command("git", "commit", "-q", "--allow-empty", "-m", message)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
}

func dayStrings(days []OverviewDay) []string {
	var out []string
	for _, day := range days {
		out = append(out, day.Day.Format("2006-01-02"))
	}
	return out
}

func TestOverviewFollowsTheFilterAndGivesRows(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "branch", "-M", "main")
	commitAt(t, dir, "one", "2026-01-10T12:00:00")
	commitAt(t, dir, "two", "2026-01-10T13:00:00")
	gittest.Run(t, dir, "tag", "-a", "v1", "-m", "v1")
	gittest.Run(t, dir, "switch", "-q", "-c", "side")
	commitAt(t, dir, "side", "2026-02-01T12:00:00")
	gittest.Run(t, dir, "switch", "-q", "main")
	commitAt(t, dir, "three", "2026-03-01T12:00:00")
	gittest.Run(t, dir, "merge", "-q", "--no-ff", "-m", "merge", "side")
	ctx := context.Background()

	all, err := Overview(ctx, dir, HistoryFilter{Ref: "--all"})
	if err != nil {
		t.Fatal(err)
	}
	history, err := History(ctx, dir, 0, 0, HistoryFilter{Ref: "--all"})
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, day := range all.Days {
		total += day.Commits
		first := history[day.FirstRow]
		if first.Date.Local().Format("2006-01-02") != day.Day.Format("2006-01-02") {
			t.Fatalf("row %d is dated %v, not %v", day.FirstRow, first.Date, day.Day)
		}
	}
	if total != len(history) {
		t.Fatalf("days count %d commits, History lists %d", total, len(history))
	}
	if got := dayStrings(all.Days); len(got) < 3 || got[0] != "2026-01-10" {
		t.Fatalf("days = %v", got)
	}
	kinds := map[string]bool{}
	for _, marker := range all.Markers {
		kinds[marker.Kind+":"+marker.Name] = true
	}
	for _, want := range []string{"head:HEAD", "branch:main", "branch:side", "tag:v1"} {
		if !kinds[want] {
			t.Errorf("no marker %s in %v", want, kinds)
		}
	}

	firstParent, err := Overview(ctx, dir, HistoryFilter{Ref: "main", FirstParent: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, day := range firstParent.Days {
		if day.Day.Format("2006-01") == "2026-02" {
			t.Fatalf("first parent only should skip the side branch's day: %v", dayStrings(firstParent.Days))
		}
	}
	for _, marker := range firstParent.Markers {
		if marker.Name == "side" {
			t.Fatalf("a branch off the listed line has a marker: %+v", marker)
		}
	}
}

func TestOverviewOfTypedHistoryAndNoCommits(t *testing.T) {
	dir := gittest.InitRepo(t)
	empty, err := Overview(context.Background(), dir, HistoryFilter{})
	if err != nil || len(empty.Days) != 0 {
		t.Fatalf("no commits = %+v, %v", empty, err)
	}
	commitAt(t, dir, "feat: wanted", "2026-01-10T12:00:00")
	commitAt(t, dir, "chore: not wanted", "2026-01-11T12:00:00")
	typed, err := Overview(context.Background(), dir, HistoryFilter{Type: "feat"})
	if err != nil {
		t.Fatal(err)
	}
	if got := dayStrings(typed.Days); len(got) != 1 || got[0] != "2026-01-10" {
		t.Fatalf("typed days = %v", got)
	}
}

func TestOverviewMarksAStash(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "a.txt", "one\n", "initial")
	gittest.WriteFile(t, dir, "a.txt", "two\n")
	gittest.Run(t, dir, "stash")
	overview, err := Overview(context.Background(), dir, HistoryFilter{Ref: "--all"})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, marker := range overview.Markers {
		found = found || (marker.Kind == OverviewStash && time.Since(marker.At) < time.Hour)
	}
	if !found {
		t.Fatalf("markers = %+v", overview.Markers)
	}
}
