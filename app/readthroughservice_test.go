package app

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func readThroughRepo(t *testing.T) string {
	t.Helper()
	repo := notesRepo(t)
	gitIn(t, repo, "switch", "-q", "side")
	writeIn(t, repo, "b.txt", "b\n")
	gitIn(t, repo, "add", "b.txt")
	gitIn(t, repo, "commit", "-q", "-m", "b")
	gitIn(t, repo, "switch", "-q", "main")
	writeIn(t, repo, "main.txt", "main\n")
	gitIn(t, repo, "add", "main.txt")
	gitIn(t, repo, "commit", "-q", "-m", "main moves on")
	return repo
}

func writeIn(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func openReadThrough(t *testing.T, repo string, request ReadThroughRequest) ReadThroughInfo {
	t.Helper()
	info, err := new(ReadThroughService).Open(repo, request)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func pathsOf(info ReadThroughInfo) []string {
	return mapSlice(info.Files, func(f ReadThroughFile) string { return f.Path })
}

func TestReadThroughABranchAgainstItsMergeBase(t *testing.T) {
	repo := readThroughRepo(t)
	info := openReadThrough(t, repo, ReadThroughRequest{Ref: "refs/heads/side", Uncommitted: true})
	if !slices.Equal(pathsOf(info), []string{"b.txt"}) || info.Target != "main" || info.Name != "side" || info.CheckedOut || info.Uncommitted {
		t.Errorf("info = %+v", info)
	}
	if info := openReadThrough(t, repo, ReadThroughRequest{Ref: "refs/heads/main"}); len(info.Files) != 0 || !info.CheckedOut {
		t.Errorf("the default branch against itself = %+v", info)
	}
	if info := openReadThrough(t, repo, ReadThroughRequest{Ref: "refs/heads/main", Target: "side"}); !slices.Equal(pathsOf(info), []string{"main.txt"}) {
		t.Errorf("against a chosen target = %+v", info)
	}
	if _, err := new(ReadThroughService).Open(repo, ReadThroughRequest{Ref: "refs/heads/side", Target: "nowhere"}); err == nil {
		t.Error("a target that doesn't exist should be refused")
	}
}

func TestReadThroughAWorktreesUncommittedChanges(t *testing.T) {
	repo := readThroughRepo(t)
	worktree := filepath.Join(t.TempDir(), "linked")
	gitIn(t, repo, "worktree", "add", "-q", worktree, "side")
	writeIn(t, worktree, "new.txt", "new\n")

	for _, request := range []ReadThroughRequest{{Path: worktree}, {Ref: "refs/heads/side"}} {
		request.Uncommitted = true
		info := openReadThrough(t, repo, request)
		if !slices.Equal(pathsOf(info), []string{"b.txt", "new.txt"}) || info.Key != "refs/heads/side" {
			t.Errorf("%+v = %+v", request, info)
		}
		diff, err := new(ReadThroughService).FileDiff(info, "", "new.txt", false, false)
		if err != nil || len(diff.Hunks) != 1 {
			t.Errorf("diff = %+v, %v", diff, err)
		}
	}
	if info := openReadThrough(t, repo, ReadThroughRequest{Path: worktree}); !slices.Equal(pathsOf(info), []string{"b.txt"}) {
		t.Errorf("committed only = %v", pathsOf(info))
	}

	gitIn(t, worktree, "switch", "-q", "--detach")
	info := openReadThrough(t, repo, ReadThroughRequest{Path: worktree, Uncommitted: true})
	if !strings.HasPrefix(info.Key, worktreeKey) {
		t.Errorf("a detached worktree's key = %q", info.Key)
	}
	s := new(ReadThroughService)
	if err := s.SetRead(info, info.Files, true); err != nil {
		t.Fatal(err)
	}
	if again := openReadThrough(t, repo, ReadThroughRequest{Path: worktree, Uncommitted: true}); !again.Files[0].Read || !again.Files[1].Read {
		t.Errorf("a detached worktree's marks = %+v", again.Files)
	}
	if err := new(WorktreeService).Remove(repo, worktree, true); err != nil {
		t.Fatal(err)
	}
	if refs := gitOut(t, repo, "for-each-ref", "refs/current-client/read"); refs != "" {
		t.Errorf("a removed worktree's marks are still there: %s", refs)
	}
}

func TestReadThroughAStackedBranchAgainstItsParent(t *testing.T) {
	repo := readThroughRepo(t)
	gitIn(t, repo, "switch", "-q", "-c", "layer", "side")
	writeIn(t, repo, "c.txt", "c\n")
	gitIn(t, repo, "add", "c.txt")
	gitIn(t, repo, "commit", "-q", "-m", "c")
	if err := new(StackService).AddLayer(repo, "layer", "side"); err != nil {
		t.Fatal(err)
	}
	if info := openReadThrough(t, repo, ReadThroughRequest{Ref: "refs/heads/layer"}); info.Target != "side" || !slices.Equal(pathsOf(info), []string{"c.txt"}) {
		t.Errorf("info = %+v", info)
	}
	gitIn(t, repo, "branch", "-q", "-D", "side")
	if info := openReadThrough(t, repo, ReadThroughRequest{Ref: "refs/heads/layer"}); info.Target != "main" || len(info.Files) != 2 {
		t.Errorf("with its parent deleted, info = %+v", info)
	}
}

func TestReadThroughMarksFollowTheirBranch(t *testing.T) {
	repo := readThroughRepo(t)
	s := new(ReadThroughService)
	info := openReadThrough(t, repo, ReadThroughRequest{Ref: "refs/heads/side"})
	if err := s.SetRead(info, info.Files, true); err != nil {
		t.Fatal(err)
	}
	if err := new(BranchService).RenameBranch(repo, "side", "renamed"); err != nil {
		t.Fatal(err)
	}
	if info := openReadThrough(t, repo, ReadThroughRequest{Ref: "refs/heads/renamed"}); !info.Files[0].Read {
		t.Error("a renamed branch lost its marks")
	}
	if _, err := new(BranchService).DeleteBranch(repo, "renamed", true); err != nil {
		t.Fatal(err)
	}
	if refs := gitOut(t, repo, "for-each-ref", "refs/current-client/read"); refs != "" {
		t.Errorf("a deleted branch's marks are still there: %s", refs)
	}
}

func TestRemovingARepositoryRemovesItsMarks(t *testing.T) {
	repo := readThroughRepo(t)
	info := openReadThrough(t, repo, ReadThroughRequest{Ref: "refs/heads/side"})
	if err := new(ReadThroughService).SetRead(info, info.Files, true); err != nil {
		t.Fatal(err)
	}
	if err := new(RepositoryService).RemoveRecentRepository(repo); err != nil {
		t.Fatal(err)
	}
	if refs := gitOut(t, repo, "for-each-ref", "refs/current-client"); refs != "" {
		t.Errorf("refs = %s", refs)
	}
}

func TestRefOfAChangeOnThisComputer(t *testing.T) {
	repo := readThroughRepo(t)
	s := new(ReadThroughService)
	side := gitOut(t, repo, "rev-parse", "side")
	gitIn(t, repo, "update-ref", "refs/remotes/origin/feature", side)
	if request, _ := s.OfChange(repo, "feature", "main", side); request.Target != "main" {
		t.Errorf("target = %q", request.Target)
	}
	gitIn(t, repo, "update-ref", "refs/remotes/origin/main", gitOut(t, repo, "rev-parse", "main"))
	if request, _ := s.OfChange(repo, "feature", "main", side); request.Target != "origin/main" {
		t.Errorf("target with a remote = %q", request.Target)
	}
	gitIn(t, repo, "update-ref", "refs/current-client/hosts/origin/mr/7", side)
	for _, c := range []struct{ branch, sha, want string }{
		{"feature", side, "refs/remotes/origin/feature"},
		{"unknown", side, "refs/heads/side"},
		{"feature", "0123456789012345678901234567890123456789", ""},
		{"feature", "", ""},
	} {
		if ref, err := s.OfChange(repo, c.branch, "main", c.sha); err != nil || ref.Ref != c.want {
			t.Errorf("OfChange(%q, %q) = %q, %v", c.branch, c.sha, ref.Ref, err)
		}
	}
	gitIn(t, repo, "switch", "-q", "side")
	gitIn(t, repo, "commit", "-q", "--allow-empty", "-m", "ahead")
	if ref, _ := s.OfChange(repo, "feature", "main", side); ref.Ref != "refs/remotes/origin/feature" {
		t.Errorf("with the branch ahead, ref = %q", ref.Ref)
	}
	gitIn(t, repo, "update-ref", "-d", "refs/remotes/origin/feature")
	if ref, _ := s.OfChange(repo, "feature", "main", side); ref.Ref != "refs/current-client/hosts/origin/mr/7" {
		t.Errorf("with only a fetched head, ref = %q", ref.Ref)
	}
	gitIn(t, repo, "update-ref", "-d", "refs/current-client/hosts/origin/mr/7")
	if ref, _ := s.OfChange(repo, "side", "main", side); ref.Ref != "refs/heads/side" {
		t.Errorf("with only a branch ahead of it, ref = %q", ref.Ref)
	}
}

func reviewNotes(t *testing.T, info ReadThroughInfo) []ReadThroughNote {
	t.Helper()
	list, err := new(ReadThroughService).Notes(info)
	if err != nil {
		t.Fatal(err)
	}
	return list
}

func TestReadThroughNotes(t *testing.T) {
	repo := readThroughRepo(t)
	gitIn(t, repo, "switch", "-q", "side")
	writeIn(t, repo, "b.txt", "first\nsecond\nthird\n")
	writeIn(t, repo, "a.txt", "replaced\n")
	gitIn(t, repo, "commit", "-q", "-a", "-m", "more")
	s := new(ReadThroughService)
	request := ReadThroughRequest{Ref: "refs/heads/side", Uncommitted: true}
	info := openReadThrough(t, repo, request)
	for _, note := range []ReadThroughNote{
		{Path: "b.txt", Line: 2, Text: " why?\n"},
		{Path: "b.txt", Line: 3, Text: "fine"},
		{Path: "b.txt", Text: "the whole file"},
		{Path: "a.txt", OldLine: 1, Text: "was this needed"},
	} {
		if err := s.AddNote(info, note); err != nil {
			t.Fatal(err)
		}
	}
	list := reviewNotes(t, info)
	if len(list) != 4 || list[0].Path != "a.txt" || list[0].LineText != "a" || list[1].Text != "the whole file" ||
		list[2].Text != "why?" || list[2].LineText != "second" {
		t.Fatalf("notes = %+v", list)
	}

	writeIn(t, repo, "b.txt", "new above\nfirst\nchanged\nthird\n")
	list = reviewNotes(t, info)
	if !list[2].Outdated || list[2].Line != 2 || list[3].Outdated || list[3].Line != 4 || list[0].Outdated || list[1].Outdated {
		t.Errorf("after an edit, notes = %+v", list)
	}
	committed := openReadThrough(t, repo, ReadThroughRequest{Ref: "refs/heads/side"})
	if list := reviewNotes(t, committed); list[2].Outdated {
		t.Errorf("without the edit, notes = %+v", list)
	}
	gitIn(t, repo, "checkout", "-q", ".")

	notesService := new(NotesService)
	if err := notesService.EditReview(repo, list[2].ID, "because"); err != nil {
		t.Fatal(err)
	}
	if err := notesService.EditReview(repo, list[3].ID, " "); err != nil {
		t.Fatal(err)
	}
	if list := reviewNotes(t, info); len(list) != 3 || list[2].Text != "because" || list[2].LineText != "second" {
		t.Errorf("after editing, notes = %+v", list)
	}

	gitIn(t, repo, "switch", "-q", "main")
	if err := new(BranchService).RenameBranch(repo, "side", "renamed"); err != nil {
		t.Fatal(err)
	}
	if list := reviewNotes(t, openReadThrough(t, repo, ReadThroughRequest{Ref: "refs/heads/renamed"})); len(list) != 3 {
		t.Errorf("after a rename, notes = %+v", list)
	}
	if note := noteOn(t, repo, "review", "refs/heads/renamed"); note == nil || note.Stale || note.Path == "" {
		t.Errorf("listed as %+v", note)
	}
	if _, err := new(BranchService).DeleteBranch(repo, "renamed", true); err != nil {
		t.Fatal(err)
	}
	if note := noteOn(t, repo, "review", "refs/heads/renamed"); note == nil || !note.Stale {
		t.Errorf("a deleted branch's note is %+v", note)
	}
}

func TestReadThroughFindings(t *testing.T) {
	repo := readThroughRepo(t)
	worktree := filepath.Join(t.TempDir(), "linked")
	gitIn(t, repo, "worktree", "add", "-q", worktree, "side")
	writeIn(t, worktree, "AGENTS.md", "x\n")
	s := new(ReadThroughService)
	rulesOf := func(info ReadThroughInfo) []string {
		t.Helper()
		report, err := s.Findings(info)
		if err != nil {
			t.Fatal(err)
		}
		return mapSlice(report.Findings, func(finding RiskFindingInfo) string { return finding.Rule })
	}
	uncommitted := openReadThrough(t, repo, ReadThroughRequest{Path: worktree, Uncommitted: true})
	objects := gitOut(t, repo, "count-objects")
	if rules := rulesOf(uncommitted); !slices.Equal(rules, []string{"agent-instructions"}) {
		t.Errorf("uncommitted = %v", rules)
	}
	if after := gitOut(t, repo, "count-objects"); after != objects {
		t.Errorf("objects went from %s to %s", objects, after)
	}
	if rules := rulesOf(openReadThrough(t, repo, ReadThroughRequest{Path: worktree})); len(rules) != 0 {
		t.Errorf("committed = %v", rules)
	}
	if err := new(SettingsService).SetRiskRulesOff([]string{"agent-instructions"}); err != nil {
		t.Fatal(err)
	}
	if rules := rulesOf(uncommitted); len(rules) != 0 {
		t.Errorf("turned off = %v", rules)
	}
}

func TestReadThroughSinceLastRead(t *testing.T) {
	repo := readThroughRepo(t)
	s := new(ReadThroughService)
	request := ReadThroughRequest{Ref: "refs/heads/side"}
	info := openReadThrough(t, repo, request)
	if info.Marked != 0 {
		t.Errorf("marked = %d", info.Marked)
	}
	if err := s.SetRead(info, info.Files, true); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "switch", "-q", "side")
	writeIn(t, repo, "b.txt", "b\nmore\n")
	gitIn(t, repo, "commit", "-q", "-a", "-m", "more")
	info = openReadThrough(t, repo, request)
	if f := info.Files[0]; info.Marked != 1 || f.Read || !f.Again || f.SinceAdded != 1 || f.SinceRemoved != 0 {
		t.Errorf("info = %+v", info)
	}
	diff, err := s.SinceDiff(info, "b.txt", false, false)
	if err != nil || len(diff.Hunks) != 1 || len(diff.Hunks[0].Lines) != 2 {
		t.Errorf("diff = %+v, %v", diff, err)
	}

	gitIn(t, repo, "rm", "-q", "b.txt")
	gitIn(t, repo, "commit", "-q", "-m", "gone")
	info = openReadThrough(t, repo, request)
	if len(info.Files) != 1 || !info.Files[0].Gone || info.Files[0].Status != "D" {
		t.Fatalf("info = %+v", info)
	}
	if err := s.SetRead(info, info.Files, true); err != nil {
		t.Fatal(err)
	}
	if info := openReadThrough(t, repo, request); len(info.Files) != 0 || info.Marked != 0 {
		t.Errorf("read again, info = %+v", info)
	}
}
