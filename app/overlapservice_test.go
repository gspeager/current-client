package app

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gspeager/current-client/internal/config"
)

func TestAPartialCloneHidesTheForecastAndTheGrid(t *testing.T) {
	origin := notesRepo(t)
	gitIn(t, origin, "config", "uploadpack.allowFilter", "true")
	clone := filepath.Join(t.TempDir(), "clone")
	gitIn(t, origin, "clone", "-q", "--filter=blob:none", "file://"+filepath.ToSlash(origin), clone)
	var s OverlapService
	if report, err := s.Predict(clone); err != nil || report.Supported {
		t.Errorf("Predict = %+v, %v", report, err)
	}
	if grid, err := s.Grid(clone); err != nil || !grid.Partial {
		t.Errorf("Grid = %+v, %v", grid, err)
	}
}

// What the app reads by itself, with nobody asking.
func TestAPartialCloneFetchesNothingUnasked(t *testing.T) {
	useTempSettings(t)
	origin := t.TempDir()
	gitIn(t, origin, "init", "-q", "-b", "main")
	gitIn(t, origin, "config", "user.email", "test@example.com")
	gitIn(t, origin, "config", "user.name", "Test")
	gitIn(t, origin, "config", "uploadpack.allowFilter", "true")
	commit := func(content string) {
		t.Helper()
		writeIn(t, origin, "a.txt", content)
		gitIn(t, origin, "add", ".")
		gitIn(t, origin, "commit", "-qm", "edit")
	}
	commit("a\n")
	gitIn(t, origin, "switch", "-qc", "side")
	commit("a\nside\n")
	gitIn(t, origin, "switch", "-qc", "layer")
	commit("a\nlayer\n")
	gitIn(t, origin, "switch", "-q", "main")
	commit("main\na\n")
	clone := filepath.Join(t.TempDir(), "clone")
	gitIn(t, origin, "clone", "-q", "--filter=blob:none", "file://"+filepath.ToSlash(origin), clone)
	gitIn(t, clone, "branch", "-q", "side", "origin/side")
	gitIn(t, clone, "branch", "-q", "layer", "origin/layer")
	if err := updateConfig(func(cfg *config.Config) {
		cfg.Stacks = map[string][]config.StackLink{clone: {{Branch: "side", Parent: "main"}, {Branch: "layer", Parent: "side"}}}
	}); err != nil {
		t.Fatal(err)
	}
	missing := func() string { return gitOut(t, clone, "rev-list", "--objects", "--all", "--missing=print") }
	before := missing()
	if !strings.Contains(before, "?") {
		t.Fatal("the clone has every file")
	}
	for name, read := range map[string]func(){
		"the Inbox":    func() { (&InboxService{}).Unseen([]string{clone}) },
		"the stacks":   func() { _, _ = (&StackService{}).List(clone) },
		"the forecast": func() { _, _ = (&OverlapService{}).Predict(clone) },
		"the branches": func() { _, _ = (&BranchService{}).ListLocal(clone) },
		"the status":   func() { _, _ = (&StatusService{}).GetStatus(clone) },
	} {
		if read(); missing() != before {
			t.Fatalf("%s fetched", name)
		}
	}
}
