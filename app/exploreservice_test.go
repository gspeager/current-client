package app

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gspeager/current-client/internal/gittest"
)

func exploreRepo(t *testing.T) string {
	t.Helper()
	repo := notesRepo(t)
	for name, content := range map[string]string{
		"edited.go": "package a\n\nvar A = 1\n", "gone.go": "package a\n", "moved.go": strings.Repeat("a line\n", 20),
		"clash.txt": "one\n", "go.sum": "x\n", "edited_test.go": "package a\n",
	} {
		writeIn(t, repo, name, content)
	}
	gittest.Run(t, repo, "add", "-A")
	gittest.Run(t, repo, "commit", "-q", "-m", "files")
	gittest.Run(t, repo, "tag", "v2")
	gittest.Run(t, repo, "switch", "-q", "-c", "feat")
	writeIn(t, repo, "edited.go", "package a\n\nvar A = 2\nvar B = 3\n")
	writeIn(t, repo, "added.go", "new here\n")
	writeIn(t, repo, "clash.txt", "theirs\n")
	gittest.Run(t, repo, "mv", "moved.go", "elsewhere.go")
	gittest.Run(t, repo, "rm", "-q", "gone.go")
	gittest.Run(t, repo, "add", "-A")
	gittest.Run(t, repo, "commit", "-q", "-m", "feat")
	gittest.Run(t, repo, "switch", "-q", "main")
	writeIn(t, repo, "clash.txt", "ours\n")
	gittest.Run(t, repo, "commit", "-q", "-a", "-m", "clash")
	return repo
}

func changesOf(t *testing.T, repo, kind, ref string) MapChangesInfo {
	t.Helper()
	changes, err := new(ExploreService).Changes(context.Background(), repo, ExploreComparison{Kind: kind, Ref: ref})
	if err != nil {
		t.Fatal(err)
	}
	return changes
}

func describeChanges(changes MapChangesInfo) []string {
	all := mapSlice(changes.Files, func(f MapChangeInfo) string {
		text := fmt.Sprintf("%s %s +%d -%d", f.Status, f.Path, f.Added, f.Removed)
		if f.Conflict {
			text += " conflict"
		}
		return text
	})
	slices.Sort(all)
	return all
}

func TestExploreFilesAndActivity(t *testing.T) {
	repo := exploreRepo(t)
	s := new(ExploreService)
	files, err := s.Files(context.Background(), repo, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]string{}
	for _, f := range files {
		kinds[f.Path] = fmt.Sprintf("%d %s", f.Lines, f.Kind)
	}
	if kinds["moved.go"] != "20 " || kinds["go.sum"] != "1 lockfile" || kinds["edited_test.go"] != "1 test" {
		t.Errorf("files = %v", kinds)
	}
	if files, err := s.Files(context.Background(), repo, "v1"); err != nil || len(files) != 1 {
		t.Errorf("at v1 = %v, %v", files, err)
	}
	activity, err := s.Activity(context.Background(), repo)
	if err != nil || activity.Limit != 0 {
		t.Fatal(activity, err)
	}
	if clash := activity.Files["clash.txt"]; !slices.Equal(clash.Commits, []int{2, 2, 2, 2}) || clash.Last == "" {
		t.Errorf("clash.txt = %+v", clash)
	}
}

func TestExploreABranchAndItsBase(t *testing.T) {
	repo := exploreRepo(t)
	changes := changesOf(t, repo, "branch", "feat")
	want := []string{
		"A added.go +1 -0", "D gone.go +0 -1", "M clash.txt +1 -1 conflict", "M edited.go +2 -1", "R elsewhere.go +0 -0",
	}
	if got := describeChanges(changes); !slices.Equal(got, want) || changes.Target != "main" || !changes.Predicted {
		t.Errorf("changes = %v, %+v", got, changes)
	}
}

func TestExploreTheWorkingTreeAndARelease(t *testing.T) {
	repo := exploreRepo(t)
	writeIn(t, repo, "edited.go", "package a\n")
	writeIn(t, repo, "untracked.go", "not tracked\n")
	gittest.Run(t, repo, "rm", "-q", "gone.go")
	gittest.Run(t, repo, "mv", "moved.go", "elsewhere.go")
	want := []string{"A untracked.go +1 -0", "D gone.go +0 -1", "M edited.go +0 -2", "R elsewhere.go +0 -0"}
	if got := describeChanges(changesOf(t, repo, "working", "")); !slices.Equal(got, want) {
		t.Errorf("working = %v", got)
	}
	if got := describeChanges(changesOf(t, repo, "release", "v2")); !slices.Equal(got, []string{"M clash.txt +1 -1"}) {
		t.Errorf("release = %v", got)
	}
	if _, err := new(ExploreService).Changes(context.Background(), repo, ExploreComparison{Kind: "planet"}); err == nil {
		t.Error("an unknown comparison should be refused")
	}
}

func TestExploreWithNothingToCompareWith(t *testing.T) {
	useTempSettings(t)
	repo := filepath.Join(t.TempDir(), "r")
	gittest.Run(t, t.TempDir(), "init", "-q", "-b", "odd", repo)
	gittest.Run(t, repo, "config", "user.name", "T")
	gittest.Run(t, repo, "config", "user.email", "t@example.com")
	s := new(ExploreService)
	if files, err := s.Files(context.Background(), repo, "HEAD"); err != nil || len(files) != 0 {
		t.Errorf("before the first commit, files = %v, %v", files, err)
	}
	if activity, err := s.Activity(context.Background(), repo); err != nil || activity.Files == nil {
		t.Errorf("before the first commit, activity = %v, %v", activity, err)
	}
	if changes := changesOf(t, repo, "working", ""); len(changes.Files) != 0 {
		t.Errorf("before the first commit, changes = %+v", changes)
	}
	gittest.Run(t, repo, "commit", "-q", "--allow-empty", "-m", "x")
	if changes := changesOf(t, repo, "branch", "odd"); changes.Target != "" || len(changes.Files) != 0 {
		t.Errorf("changes = %+v", changes)
	}
	if timeline := timelineOf(t, repo, "branch", "odd"); timeline.Target != "" || len(timeline.Commits) != 0 {
		t.Errorf("timeline = %+v", timeline)
	}
}

func timelineOf(t *testing.T, repo, kind, ref string) TimelineInfo {
	t.Helper()
	timeline, err := new(ExploreService).Timeline(context.Background(), repo, ExploreComparison{Kind: kind, Ref: ref})
	if err != nil {
		t.Fatal(err)
	}
	return timeline
}

func describeTimeline(timeline TimelineInfo) []string {
	return mapSlice(timeline.Commits, func(c TimelineCommitInfo) string {
		files := mapSlice(c.Files, func(f TimelineFileInfo) string { return fmt.Sprintf("%s %d", f.Path, f.Lines) })
		slices.Sort(files)
		return c.Subject + ": " + strings.Join(files, ", ")
	})
}

func TestExploreTimelineOfABranch(t *testing.T) {
	repo := exploreRepo(t)
	timeline := timelineOf(t, repo, "branch", "feat")
	want := []string{"feat: added.go 1, clash.txt 2, edited.go 3, elsewhere.go 0, gone.go 1"}
	if got := describeTimeline(timeline); !slices.Equal(got, want) || timeline.Target != "main" || timeline.Limit != 0 {
		t.Errorf("timeline = %v, %+v", got, timeline)
	}
	if timeline.Now || len(timeline.Working) != 0 {
		t.Errorf("a branch that isn't checked out has no uncommitted changes: %+v", timeline.Working)
	}
}

func TestExploreTimelineToNow(t *testing.T) {
	repo := exploreRepo(t)
	writeIn(t, repo, "edited.go", "package a\n")
	writeIn(t, repo, "untracked.go", "not tracked\n")

	release := timelineOf(t, repo, "release", "v2")
	if got := describeTimeline(release); !slices.Equal(got, []string{"clash: clash.txt 2"}) {
		t.Errorf("since v2 = %v", got)
	}
	working := mapSlice(release.Working, func(f TimelineFileInfo) string { return fmt.Sprintf("%s %d", f.Path, f.Lines) })
	slices.Sort(working)
	if !slices.Equal(working, []string{"edited.go 2", "untracked.go 1"}) {
		t.Errorf("uncommitted = %v", working)
	}

	all := describeTimeline(timelineOf(t, repo, "since", time.Now().Add(-time.Hour).Format(time.RFC3339)))
	if len(all) < 2 || all[0] != "clash: clash.txt 2" || !strings.HasPrefix(all[1], "files: ") {
		t.Errorf("in the last hour = %v", all)
	}
	if later := timelineOf(t, repo, "since", time.Now().Add(time.Hour).Format(time.RFC3339)); len(later.Commits) != 0 || len(later.Working) != 2 {
		t.Errorf("a range with no commits = %+v", later)
	}
	if only := timelineOf(t, repo, "working", ""); len(only.Commits) != 0 || !only.Now || len(only.Working) != 2 {
		t.Errorf("uncommitted alone = %+v", only)
	}
	if _, err := new(ExploreService).Timeline(context.Background(), repo, ExploreComparison{Kind: "since", Ref: "today"}); err == nil {
		t.Error("a time that isn't one should be refused")
	}
}
