package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func structureRepo(t *testing.T) string {
	t.Helper()
	repo := notesRepo(t)
	for name, content := range map[string]string{
		"go.mod":        "module a\n",
		"app/run.go":    "package app\n\nimport \"a/core\"\n\nfunc Run() { core.Open() }\n",
		"core/open.go":  "package core\n\nfunc Open() {}\n",
		"core/other.go": "package core\n",
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(repo, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		writeIn(t, repo, name, content)
	}
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-q", "-m", "code")
	gitIn(t, repo, "tag", "v9")
	return repo
}

func structureOf(t *testing.T, repo, path string, comparison *ExploreComparison) (StructureInfo, []string) {
	t.Helper()
	info, err := new(ExploreService).Structure(repo, path, comparison)
	if err != nil {
		t.Fatal(err)
	}
	var all []string
	for _, n := range info.Nodes {
		all = append(all, strings.TrimSpace(fmt.Sprintf("%s %s %d files %d lines %d changed %s", n.Side, n.Path, n.Count.Files, n.Count.Lines, n.Count.Changed, n.State)))
	}
	for _, l := range info.Links {
		all = append(all, strings.TrimSpace(fmt.Sprintf("%s > %s %s", info.Nodes[l.From].Path, info.Nodes[l.To].Path, l.State)))
	}
	slices.Sort(all)
	return info, all
}

func expectStructure(t *testing.T, got []string, want ...string) {
	t.Helper()
	slices.Sort(want)
	got = slices.DeleteFunc(slices.Clone(got), func(line string) bool { return strings.Contains(line, "card go.mod") || strings.Contains(line, ".txt") })
	if !slices.Equal(got, want) {
		t.Errorf("got %s\nwant %s", strings.Join(got, "\n    "), strings.Join(want, "\n     "))
	}
}

func TestStructure(t *testing.T) {
	repo := structureRepo(t)
	info, all := structureOf(t, repo, "", nil)
	expectStructure(t, all, "card app 1 files 5 lines 0 changed", "card core 2 files 4 lines 0 changed", "app > core")
	if info.Compared || !slices.Equal(info.Languages, []string{"Go"}) || info.Count.Files < 4 {
		t.Errorf("info = %+v", info)
	}
	_, all = structureOf(t, repo, "core", nil)
	expectStructure(t, all, "card core/open.go 1 files 3 lines 0 changed", "card core/other.go 1 files 1 lines 0 changed",
		"used-by app/run.go 1 files 5 lines 0 changed", "app/run.go > core/open.go")
}

func TestStructureComparedWithAnotherPoint(t *testing.T) {
	repo := structureRepo(t)
	before := gitOut(t, repo, "count-objects", "-v")
	// Uncommitted, and untracked.
	writeIn(t, repo, "core/other.go", "package core\n\nimport \"a/text\"\n\nfunc o() { text.Cut() }\n")
	if err := os.Mkdir(filepath.Join(repo, "text"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeIn(t, repo, "text/t.go", "package text\n\nfunc Cut() {}\n")
	info, all := structureOf(t, repo, "", &ExploreComparison{Kind: "working"})
	expectStructure(t, all, "card app 1 files 5 lines 0 changed", "card core 2 files 4 lines 1 changed",
		"card text 1 files 0 lines 1 changed added", "app > core", "core > text added")
	if !info.Compared || info.Count.Changed != 2 {
		t.Errorf("info = %+v", info)
	}
	if after := gitOut(t, repo, "count-objects", "-v"); after != before {
		t.Errorf("objects after = %s, before = %s", after, before)
	}

	gitIn(t, repo, "switch", "-q", "-c", "feat")
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-q", "-m", "feat")
	gitIn(t, repo, "switch", "-q", "main")
	if info, all = structureOf(t, repo, "", &ExploreComparison{Kind: "branch", Ref: "feat"}); info.Target != "main" || !slices.Contains(all, "core > text added") {
		t.Errorf("a branch, against %q: %q", info.Target, all)
	}
	gitIn(t, repo, "merge", "-q", "feat")
	for _, since := range []ExploreComparison{{Kind: "release", Ref: "v9"}, {Kind: "commit", Ref: "HEAD~1"}} {
		if _, all = structureOf(t, repo, "", &since); !slices.Contains(all, "core > text added") {
			t.Errorf("since %v: %q", since, all)
		}
	}
	if _, err := new(ExploreService).Structure(repo, "", &ExploreComparison{Kind: "commit", Ref: "nowhere"}); err == nil {
		t.Error("a point that isn't there has a structure")
	}
	// A moved HEAD is another structure.
	gitIn(t, repo, "reset", "-q", "--hard", "v9")
	if _, all = structureOf(t, repo, "", nil); slices.Contains(all, "core > text") {
		t.Errorf("after moving back: %q", all)
	}
}

func TestStructureInAPartialCloneFetchesNothing(t *testing.T) {
	origin := structureRepo(t)
	writeIn(t, origin, "core/other.go", "package core\n\nvar later = 1\n")
	gitIn(t, origin, "commit", "-qam", "later")
	gitIn(t, origin, "config", "uploadpack.allowFilter", "true")
	clone := filepath.Join(t.TempDir(), "clone")
	gitIn(t, origin, "clone", "-q", "--filter=blob:none", "file://"+filepath.ToSlash(origin), clone)
	missing := func() string { return gitOut(t, clone, "rev-list", "--objects", "--all", "--missing=print") }
	before := missing()
	for _, comparison := range []*ExploreComparison{nil, {Kind: "working"}, {Kind: "release", Ref: "v9"}} {
		info, _ := structureOf(t, clone, "", comparison)
		if len(info.Links) != 1 || info.Partial != (comparison != nil && comparison.Kind == "release") {
			t.Errorf("%v: %+v", comparison, info)
		}
	}
	if after := missing(); after != before {
		t.Errorf("fetched: missing after = %s, before = %s", after, before)
	}
}

func TestStructureBeforeTheFirstCommit(t *testing.T) {
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q")
	info, err := new(ExploreService).Structure(repo, "", nil)
	if err != nil || info.Nodes == nil || len(info.Nodes) != 0 {
		t.Error(info, errors.Join(err))
	}
}
