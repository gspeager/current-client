package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	want := Config{
		Version:                     CurrentVersion,
		RecentRepositories:          []string{"/a", "/b"},
		IdentityColors:              map[string]string{"a@example.com": "#ffcc00"},
		Theme:                       "light",
		GitExecutablePath:           "/usr/local/bin/git",
		DefaultRepoLocation:         "/home/dev/projects",
		DiffIgnoreWhitespaceDefault: true,
		EditorPath:                  "/usr/local/bin/code",
		AutoFetchIntervalMinutes:    15,
		LaneColorTheme:              "vivid",
		PaneWidths:                  map[string]int{"pane-width-nav": 240},
		WindowWidth:                 1280,
		WindowHeight:                800,
		OpenTabs:                    []string{"/a", "/b"},
		FocusedTab:                  "/b",
		NavCollapsed:                true,
		DiffExpanded:                true,
		DiffNoWrap:                  true,
		PruneOnFetch:                true,
		DisableConventionalCommits:  true,
		Changelog:                   &ChangelogPrefs{Types: []string{"feat"}, Dates: true, SplitByRelease: true},
	}
	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestLoadMissingFileReturnsDefault(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, Default()) {
		t.Fatalf("got %+v, want default %+v", got, Default())
	}
}

func TestDefaultPathEndsInConfigFile(t *testing.T) {
	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if filepath.Base(path) != "config.json" {
		t.Fatalf("DefaultPath = %q, want it to end in config.json", path)
	}
}

// useConfigDir points os.UserConfigDir at a temp folder on every platform.
func useConfigDir(t *testing.T) string {
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	base, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir: %v", err)
	}
	return base
}

// The pattern for future format changes: each step upgrades one version, and
// they run in order, so step 2 sees step 1's result.
func TestMigrateRunsEachStepInOrder(t *testing.T) {
	raw := map[string]any{"version": float64(1), "theme": "light"}
	steps := map[int]func(map[string]any){
		1: func(r map[string]any) {
			r["appearance"] = r["theme"]
			delete(r, "theme")
		},
		2: func(r map[string]any) {
			r["appearance"] = r["appearance"].(string) + "-v3"
		},
	}

	migrate(raw, 1, 3, steps)

	want := map[string]any{"version": 3, "appearance": "light-v3"}
	if !reflect.DeepEqual(raw, want) {
		t.Fatalf("migrated = %v, want %v", raw, want)
	}
}

func TestLoadCurrentFormatUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"theme":"light","recentRepositories":["/a"]}`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := Config{Version: 1, Theme: "light", RecentRepositories: []string{"/a"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load = %+v, want %+v", got, want)
	}
}

func TestLoadFileWithoutVersionIsStampedCurrent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"theme":"dark"}`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := Load(path)
	if err != nil || got.Version != CurrentVersion || got.Theme != "dark" {
		t.Fatalf("Load = %+v, %v; want version %d with theme kept", got, err, CurrentVersion)
	}
}

func TestLoadNewerFormatKeepsWhatItUnderstands(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"version":99,"theme":"light","somethingNew":{"a":1}}`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := Load(path)
	if err != nil || got.Theme != "light" {
		t.Fatalf("Load = %+v, %v; want the known fields of a newer file", got, err)
	}
}

func TestChangelogTypesKeepAnEmptyListDistinctFromUnset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := Save(path, Config{Version: CurrentVersion, Changelog: &ChangelogPrefs{Types: []string{}}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Changelog == nil || got.Changelog.Types == nil || len(got.Changelog.Types) != 0 {
		t.Errorf("Types = %#v, want an empty, non-nil list", got.Changelog)
	}
}

func TestSetDirNameMovesSettings(t *testing.T) {
	base := useConfigDir(t)
	SetDirName("current")
	t.Cleanup(func() { SetDirName(defaultDirName) })

	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if want := filepath.Join(base, "current", "config.json"); path != want {
		t.Errorf("DefaultPath = %q, want %q", path, want)
	}
}
