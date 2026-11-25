package app

import (
	"path/filepath"
	"testing"
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
