package app

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gspeager/current-client/internal/config"
	"github.com/gspeager/current-client/internal/gittest"
)

func TestAPartialCloneHidesTheForecastAndTheGrid(t *testing.T) {
	origin := notesRepo(t)
	gittest.Run(t, origin, "config", "uploadpack.allowFilter", "true")
	clone := filepath.Join(t.TempDir(), "clone")
	gittest.Run(t, origin, "clone", "-q", "--filter=blob:none", "file://"+filepath.ToSlash(origin), clone)
	var s OverlapService
	if report, err := s.Predict(clone); err != nil || report.Supported {
		t.Errorf("Predict = %+v, %v", report, err)
	}
	if grid, err := s.Grid(clone); err != nil || !grid.Partial {
		t.Errorf("Grid = %+v, %v", grid, err)
	}
}

// What the sidebar and the Inbox read, which is no Git operation anyone started.
func TestAPartialCloneFetchesNothingUnasked(t *testing.T) {
	useTempSettings(t)
	origin := t.TempDir()
	gittest.Run(t, origin, "init", "-q", "-b", "main")
	gittest.Run(t, origin, "config", "user.email", "test@example.com")
	gittest.Run(t, origin, "config", "user.name", "Test")
	gittest.Run(t, origin, "config", "uploadpack.allowFilter", "true")
	commit := func(content string) {
		t.Helper()
		writeIn(t, origin, "a.txt", content)
		gittest.Run(t, origin, "add", ".")
		gittest.Run(t, origin, "commit", "-qm", "edit")
	}
	commit("a\n")
	gittest.Run(t, origin, "switch", "-qc", "side")
	commit("a\nside\n")
	gittest.Run(t, origin, "switch", "-qc", "layer")
	commit("a\nlayer\n")
	gittest.Run(t, origin, "switch", "-q", "main")
	commit("main\na\n")
	for name, read := range map[string]func(clone string){
		"the Inbox":    func(clone string) { (&InboxService{}).Rows([]string{clone}, false) },
		"the stacks":   func(clone string) { _, _ = (&StackService{}).List(clone) },
		"the forecast": func(clone string) { _, _ = (&OverlapService{}).Predict(clone) },
		"the branches": func(clone string) { _, _ = (&BranchService{}).ListLocal(clone) },
		"the status":   func(clone string) { _, _ = (&StatusService{}).GetStatus(clone) },
	} {
		t.Run(name, func(t *testing.T) {
			clone := filepath.Join(t.TempDir(), "clone")
			gittest.Run(t, origin, "clone", "-q", "--filter=blob:none", "file://"+filepath.ToSlash(origin), clone)
			gittest.Run(t, clone, "branch", "-q", "side", "origin/side")
			gittest.Run(t, clone, "branch", "-q", "layer", "origin/layer")
			if err := updateConfig(func(cfg *config.Config) {
				cfg.Stacks = map[string][]config.StackLink{clone: {{Branch: "side", Parent: "main"}, {Branch: "layer", Parent: "side"}}}
			}); err != nil {
				t.Fatal(err)
			}
			missing := func() string { return gittest.Output(t, clone, "rev-list", "--objects", "--all", "--missing=print") }
			before := missing()
			if !strings.Contains(before, "?") {
				t.Fatal("the clone has every file")
			}
			if read(clone); missing() != before {
				t.Error("it fetched")
			}
		})
	}
}
