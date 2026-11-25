package app

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
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
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-q", "-m", "files")
	gitIn(t, repo, "tag", "v2")
	gitIn(t, repo, "switch", "-q", "-c", "feat")
	writeIn(t, repo, "edited.go", "package a\n\nvar A = 2\nvar B = 3\n")
	writeIn(t, repo, "added.go", "new here\n")
	writeIn(t, repo, "clash.txt", "theirs\n")
	gitIn(t, repo, "mv", "moved.go", "elsewhere.go")
	gitIn(t, repo, "rm", "-q", "gone.go")
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-q", "-m", "feat")
	gitIn(t, repo, "switch", "-q", "main")
	writeIn(t, repo, "clash.txt", "ours\n")
	gitIn(t, repo, "commit", "-q", "-a", "-m", "clash")
	return repo
}

func changesOf(t *testing.T, repo, kind, ref string) MapChangesInfo {
	t.Helper()
	changes, err := new(ExploreService).Changes(repo, ExploreComparison{Kind: kind, Ref: ref})
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
	files, err := s.Files(repo, "HEAD")
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
	if files, err := s.Files(repo, "v1"); err != nil || len(files) != 1 {
		t.Errorf("at v1 = %v, %v", files, err)
	}
	activity, err := s.Activity(repo)
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
	gitIn(t, repo, "rm", "-q", "gone.go")
	gitIn(t, repo, "mv", "moved.go", "elsewhere.go")
	want := []string{"A untracked.go +1 -0", "D gone.go +0 -1", "M edited.go +0 -2", "R elsewhere.go +0 -0"}
	if got := describeChanges(changesOf(t, repo, "working", "")); !slices.Equal(got, want) {
		t.Errorf("working = %v", got)
	}
	if got := describeChanges(changesOf(t, repo, "release", "v2")); !slices.Equal(got, []string{"M clash.txt +1 -1"}) {
		t.Errorf("release = %v", got)
	}
	if _, err := new(ExploreService).Changes(repo, ExploreComparison{Kind: "planet"}); err == nil {
		t.Error("an unknown comparison should be refused")
	}
}

func TestExploreWithNothingToCompareWith(t *testing.T) {
	useTempSettings(t)
	repo := filepath.Join(t.TempDir(), "r")
	gitIn(t, t.TempDir(), "init", "-q", "-b", "odd", repo)
	gitIn(t, repo, "config", "user.name", "T")
	gitIn(t, repo, "config", "user.email", "t@example.com")
	gitIn(t, repo, "commit", "-q", "--allow-empty", "-m", "x")
	if changes := changesOf(t, repo, "branch", "odd"); changes.Target != "" || len(changes.Files) != 0 {
		t.Errorf("changes = %+v", changes)
	}
}
