package app

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/gspeager/current-client/internal/config"
)

func useTempSettings(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	path, err := config.DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	return path
}

func TestConcurrentSettingsUpdatesAreAllKept(t *testing.T) {
	useTempSettings(t)
	var wg sync.WaitGroup
	for i := range 40 {
		wg.Go(func() {
			if err := (&SettingsService{}).SetPaneWidth(fmt.Sprintf("pane-%d", i), i+100); err != nil {
				t.Errorf("SetPaneWidth: %v", err)
			}
		})
	}
	wg.Wait()

	widths, err := (&SettingsService{}).GetPaneWidths()
	if err != nil {
		t.Fatalf("GetPaneWidths: %v", err)
	}
	if len(widths) != 40 {
		t.Fatalf("kept %d of 40 pane widths", len(widths))
	}
}

func TestUnreadableSettingsAreMovedAsideAndReportedOnce(t *testing.T) {
	path := useTempSettings(t)
	if err := os.MkdirAll(strings.TrimSuffix(path, "config.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"version": 1, "theme": "li`), 0o600); err != nil {
		t.Fatal(err)
	}
	settings := &SettingsService{}

	if err := settings.SetTheme("light"); err != nil {
		t.Fatalf("SetTheme after a truncated file: %v", err)
	}
	if got, _ := settings.GetSettings(); got.Theme != "light" {
		t.Errorf("Theme = %q, want the new setting saved over defaults", got.Theme)
	}
	if broken, err := os.ReadFile(path + ".broken"); err != nil || !strings.Contains(string(broken), `"li`) {
		t.Errorf("old file not kept at %s.broken: %v", path, err)
	}
	if notice := settings.TakeSettingsResetNotice(); !strings.Contains(notice, path+".broken") {
		t.Errorf("notice = %q, want it to name the kept file", notice)
	}
	if notice := settings.TakeSettingsResetNotice(); notice != "" {
		t.Errorf("second notice = %q, want it reported once", notice)
	}
}
