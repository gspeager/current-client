package app

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gspeager/current-client/internal/config"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestChangelogPrefsDefaultTypesOnlyWhenUnset(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
		want []string
	}{
		{"no changelog prefs", config.Config{}, defaultChangelogTypes},
		{"types never changed", config.Config{Changelog: &config.ChangelogPrefs{Dates: true}}, defaultChangelogTypes},
		{"every type turned off", config.Config{Changelog: &config.ChangelogPrefs{Types: []string{}}}, []string{}},
		{"chosen types", config.Config{Changelog: &config.ChangelogPrefs{Types: []string{"docs"}}}, []string{"docs"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := changelogPrefs(tt.cfg).Types; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Types = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestServicesSaveIntoChosenSettingsFolder(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	base, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir: %v", err)
	}
	SetSettingsFolder("current")
	t.Cleanup(func() { SetSettingsFolder("current-client") })

	if err := (&SettingsService{}).SetTheme("light"); err != nil {
		t.Fatalf("SetTheme: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, "current", "config.json")); err != nil {
		t.Errorf("settings not saved in the chosen folder: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, "current-client")); !os.IsNotExist(err) {
		t.Errorf("Current Client's settings folder was created: %v", err)
	}
}

func TestTitleBarThemeFollowsSavedTheme(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	settings := &SettingsService{}

	if got := TitleBarTheme(); got != application.Dark {
		t.Errorf("no saved theme: %v, want Dark (Current Client's default)", got)
	}
	for theme, want := range map[string]application.Theme{"light": application.Light, "dark": application.Dark} {
		if err := settings.SetTheme(theme); err != nil {
			t.Fatalf("SetTheme(%q): %v", theme, err)
		}
		if got := TitleBarTheme(); got != want {
			t.Errorf("%s: %v, want %v", theme, got, want)
		}
	}
	if err := settings.SetTheme("system"); err != nil {
		t.Fatal(err)
	}
	if got := TitleBarTheme(); got == application.SystemDefault {
		t.Error("system: SystemDefault, want an explicit Dark or Light so Wails doesn't override SetDarkTitleBar")
	}
}

func TestSidebarSectionsAreKeptPerRepository(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("APPDATA", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	svc := &SettingsService{}

	if err := svc.SetSidebarSection("/repos/a", "stash", false); err != nil {
		t.Fatalf("SetSidebarSection: %v", err)
	}
	if err := svc.SetSidebarSection("/repos/a", "tags", true); err != nil {
		t.Fatalf("SetSidebarSection: %v", err)
	}
	got, err := svc.GetSidebarSections("/repos/a")
	if err != nil {
		t.Fatalf("GetSidebarSections: %v", err)
	}
	if want := map[string]bool{"stash": false, "tags": true}; !reflect.DeepEqual(got, want) {
		t.Errorf("sections = %#v, want %#v", got, want)
	}
	other, err := svc.GetSidebarSections("/repos/b")
	if err != nil {
		t.Fatalf("GetSidebarSections: %v", err)
	}
	if len(other) != 0 {
		t.Errorf("another repository has sections %#v", other)
	}
}
