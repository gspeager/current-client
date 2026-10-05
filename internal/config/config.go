package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const CurrentVersion = 1

// ChangelogPrefs.Types is nil until the type filter is first changed, and the
// settings service then supplies defaults; an empty list means every type is
// filtered out.
type ChangelogPrefs struct {
	Types          []string `json:"types"`
	AuthorNames    bool     `json:"authorNames,omitempty"`
	Dates          bool     `json:"dates,omitempty"`
	Preview        bool     `json:"preview,omitempty"`
	SplitByRelease bool     `json:"splitByRelease,omitempty"`
}

type Config struct {
	Version            int               `json:"version"`
	RecentRepositories []string          `json:"recentRepositories,omitempty"`
	IdentityColors     map[string]string `json:"identityColors,omitempty"`

	Theme                       string `json:"theme,omitempty"`               // "system", "dark" (default) or "light"
	GitExecutablePath           string `json:"gitExecutablePath,omitempty"`   // empty = auto-detect via PATH
	DefaultRepoLocation         string `json:"defaultRepoLocation,omitempty"` // starting folder for Open/Clone dialogs
	DiffIgnoreWhitespaceDefault bool   `json:"diffIgnoreWhitespaceDefault,omitempty"`
	EditorPath                  string `json:"editorPath,omitempty"` // empty = "code" (VS Code) via PATH

	// Zero disables auto-fetch, keeping background network access opt-in.
	AutoFetchIntervalMinutes int `json:"autoFetchIntervalMinutes,omitempty"`

	LaneColorTheme string `json:"laneColorTheme,omitempty"`

	// Keyed by the storage key passed to the frontend's useResizableWidth.
	PaneWidths map[string]int `json:"paneWidths,omitempty"`

	WindowWidth  int `json:"windowWidth,omitempty"`
	WindowHeight int `json:"windowHeight,omitempty"`

	OpenTabs   []string `json:"openTabs,omitempty"`
	FocusedTab string   `json:"focusedTab,omitempty"`

	NavCollapsed bool `json:"navCollapsed,omitempty"`

	// Working Copy's diff takes the file list's space too.
	DiffExpanded bool `json:"diffExpanded,omitempty"`
	// Zero keeps long diff lines wrapping, as they always have.
	DiffNoWrap bool `json:"diffNoWrap,omitempty"`

	// Whether the header's fetch also deletes remote-tracking branches gone from the remote.
	PruneOnFetch bool `json:"pruneOnFetch,omitempty"`

	// Zero keeps the Changelog tab and commit type picker on.
	DisableConventionalCommits bool `json:"disableConventionalCommits,omitempty"`

	Changelog *ChangelogPrefs `json:"changelog,omitempty"`
}

func Default() Config {
	return Config{Version: CurrentVersion}
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Config{}, err
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return Config{}, err
	}
	// A file from a newer version is read as far as this version understands it.
	if version := fileVersion(raw); version <= CurrentVersion {
		migrate(raw, version, CurrentVersion, migrations)
	}
	upgraded, err := json.Marshal(raw)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(upgraded, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// migrations[v] upgrades a settings file from version v to v+1. They work on
// the raw JSON so a field can be renamed or reshaped before it's decoded. Add
// one here, and bump CurrentVersion, whenever Config's JSON shape changes.
var migrations = map[int]func(raw map[string]any){}

func migrate(raw map[string]any, from, to int, steps map[int]func(map[string]any)) {
	for v := from; v < to; v++ {
		if step, ok := steps[v]; ok {
			step(raw)
		}
	}
	raw["version"] = to
}

// Every file this app has written carries a version; one without is treated as 1.
func fileVersion(raw map[string]any) int {
	if v, ok := raw["version"].(float64); ok {
		return int(v)
	}
	return 1
}

func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

const defaultDirName = "current-client"

// dirName is only changed at startup, before anything reads settings.
var dirName = defaultDirName

// SetDirName lets an app that embeds Current Client keep its settings in its own folder
// under the user config directory. Call it before anything reads settings.
func SetDirName(name string) {
	dirName = name
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, dirName, "config.json"), nil
}
