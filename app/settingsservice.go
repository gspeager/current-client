package app

import (
	"context"

	"github.com/gspeager/current-client/core/gitexec"
	"github.com/gspeager/current-client/core/identity"
	"github.com/gspeager/current-client/internal/config"
)

type SettingsService struct{}

type Settings struct {
	Theme                       string                `json:"theme"`
	GitExecutablePath           string                `json:"gitExecutablePath"`
	DefaultRepoLocation         string                `json:"defaultRepoLocation"`
	DiffIgnoreWhitespaceDefault bool                  `json:"diffIgnoreWhitespaceDefault"`
	EditorPath                  string                `json:"editorPath"`
	AutoFetchIntervalMinutes    int                   `json:"autoFetchIntervalMinutes"`
	LaneColorTheme              string                `json:"laneColorTheme"`
	NavCollapsed                bool                  `json:"navCollapsed"`
	DiffExpanded                bool                  `json:"diffExpanded"`
	DiffNoWrap                  bool                  `json:"diffNoWrap"`
	PruneOnFetch                bool                  `json:"pruneOnFetch"`
	DisableConventionalCommits  bool                  `json:"disableConventionalCommits"`
	Changelog                   config.ChangelogPrefs `json:"changelog"`
}

func normalizeTheme(theme string) string {
	switch theme {
	case "light", "system":
		return theme
	default:
		return "dark"
	}
}

func (s *SettingsService) GetSettings() (Settings, error) {
	cfg, _, err := loadCurrentConfig()
	if err != nil {
		return Settings{}, err
	}
	return Settings{
		Theme:                       normalizeTheme(cfg.Theme),
		GitExecutablePath:           cfg.GitExecutablePath,
		DefaultRepoLocation:         cfg.DefaultRepoLocation,
		DiffIgnoreWhitespaceDefault: cfg.DiffIgnoreWhitespaceDefault,
		EditorPath:                  cfg.EditorPath,
		AutoFetchIntervalMinutes:    cfg.AutoFetchIntervalMinutes,
		LaneColorTheme:              cfg.LaneColorTheme,
		NavCollapsed:                cfg.NavCollapsed,
		DiffExpanded:                cfg.DiffExpanded,
		DiffNoWrap:                  cfg.DiffNoWrap,
		PruneOnFetch:                cfg.PruneOnFetch,
		DisableConventionalCommits:  cfg.DisableConventionalCommits,
		Changelog:                   changelogPrefs(cfg),
	}, nil
}

func (s *SettingsService) SetTheme(theme string) error {
	return updateConfig(func(cfg *config.Config) { cfg.Theme = normalizeTheme(theme) })
}

// SetGitExecutablePath rejects a path that doesn't run git, so a bad choice
// can't break every later git call.
func (s *SettingsService) SetGitExecutablePath(gitPath string) error {
	if gitPath != "" {
		if _, err := gitexec.DetectSupported(context.Background(), gitPath); err != nil {
			return err
		}
	}
	if err := updateConfig(func(cfg *config.Config) { cfg.GitExecutablePath = gitPath }); err != nil {
		return err
	}
	gitexec.SetDefaultBinary(gitPath)
	return nil
}

func (s *SettingsService) PickGitExecutable() (string, error) {
	return pickFile("Choose Git Executable")
}

func (s *SettingsService) SetDefaultRepoLocation(dir string) error {
	return updateConfig(func(cfg *config.Config) { cfg.DefaultRepoLocation = dir })
}

func (s *SettingsService) PickDefaultRepoLocation() (string, error) {
	return pickFolder("Choose Default Repository Location")
}

func (s *SettingsService) SetDiffIgnoreWhitespaceDefault(value bool) error {
	return updateConfig(func(cfg *config.Config) { cfg.DiffIgnoreWhitespaceDefault = value })
}

func (s *SettingsService) SetEditorPath(editorPath string) error {
	return updateConfig(func(cfg *config.Config) { cfg.EditorPath = editorPath })
}

func (s *SettingsService) PickEditorExecutable() (string, error) {
	return pickFile("Choose Editor Executable")
}

func (s *SettingsService) SetAutoFetchIntervalMinutes(minutes int) error {
	return updateConfig(func(cfg *config.Config) { cfg.AutoFetchIntervalMinutes = minutes })
}

func (s *SettingsService) SetLaneColorTheme(theme string) error {
	return updateConfig(func(cfg *config.Config) { cfg.LaneColorTheme = theme })
}

func (s *SettingsService) SetNavCollapsed(collapsed bool) error {
	return updateConfig(func(cfg *config.Config) { cfg.NavCollapsed = collapsed })
}

func (s *SettingsService) SetPruneOnFetch(prune bool) error {
	return updateConfig(func(cfg *config.Config) { cfg.PruneOnFetch = prune })
}

func (s *SettingsService) SetDiffExpanded(expanded bool) error {
	return updateConfig(func(cfg *config.Config) { cfg.DiffExpanded = expanded })
}

func (s *SettingsService) SetDiffNoWrap(noWrap bool) error {
	return updateConfig(func(cfg *config.Config) { cfg.DiffNoWrap = noWrap })
}

func (s *SettingsService) SetDisableConventionalCommits(disable bool) error {
	return updateConfig(func(cfg *config.Config) { cfg.DisableConventionalCommits = disable })
}

func (s *SettingsService) SetChangelogPrefs(prefs config.ChangelogPrefs) error {
	return updateConfig(func(cfg *config.Config) { cfg.Changelog = &prefs })
}

// The kinds the Changelog's type filter shows until it is first changed:
// breaking changes and the types that belong in release notes.
var defaultChangelogTypes = []string{"breaking", "feat", "fix", "perf", "revert"}

func changelogPrefs(cfg config.Config) config.ChangelogPrefs {
	prefs := config.ChangelogPrefs{}
	if cfg.Changelog != nil {
		prefs = *cfg.Changelog
	}
	if prefs.Types == nil {
		prefs.Types = defaultChangelogTypes
	}
	return prefs
}

func (s *SettingsService) GetIdentityColorOverrides() (map[string]string, error) {
	cfg, _, err := loadCurrentConfig()
	if err != nil {
		return nil, err
	}
	return cfg.IdentityColors, nil
}

func (s *SettingsService) RemoveIdentityColorOverride(key string) error {
	return updateConfig(func(cfg *config.Config) { cfg.IdentityColors = identity.RemoveColorOverride(cfg.IdentityColors, key) })
}

func (s *SettingsService) GetPaneWidths() (map[string]int, error) {
	cfg, _, err := loadCurrentConfig()
	if err != nil {
		return nil, err
	}
	return cfg.PaneWidths, nil
}

func (s *SettingsService) SetPaneWidth(key string, width int) error {
	return updateConfig(func(cfg *config.Config) {
		if cfg.PaneWidths == nil {
			cfg.PaneWidths = map[string]int{}
		}
		cfg.PaneWidths[key] = width
	})
}
