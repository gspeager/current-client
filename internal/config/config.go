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

// HostConfig is the user's choice for one Git host, keyed by hostname (with
// its port when the remote has one). Empty Provider means the type detected
// from the remote URL; empty APIURL means the address derived from the type.
// Enabled is the only thing that lets the app contact the host's API.
// HasToken records that an API token was saved with Git's credential
// helper; the token itself is never written here.
type HostConfig struct {
	Provider string `json:"provider,omitempty"`
	APIURL   string `json:"apiUrl,omitempty"`
	Enabled  bool   `json:"enabled,omitempty"`
	HasToken bool   `json:"hasToken,omitempty"`
	// Dismissed remembers "Not now" on the first-use prompt.
	Dismissed bool `json:"dismissed,omitempty"`
}

type CustomTask struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// StackLink is one layer of a stack. Git stores no branch's base, and amending the
// layer below breaks the ancestry, so ParentTip remembers what it was built on.
type StackLink struct {
	Branch    string `json:"branch"`
	Parent    string `json:"parent"`
	ParentTip string `json:"parentTip"`
}

type StackMove struct {
	Branch string `json:"branch"`
	Before string `json:"before"`
	After  string `json:"after"`
}

// StackUndo takes back the last restack, which moves several branches.
type StackUndo struct {
	Links   []StackLink `json:"links"`
	Moves   []StackMove `json:"moves"`
	Stopped bool        `json:"stopped,omitempty"`
}

// TaskChecklist runs alongside a commit or push. Failures warn and never block.
type TaskChecklist struct {
	BeforeCommit []string `json:"beforeCommit,omitempty"`
	BeforePush   []string `json:"beforePush,omitempty"`
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

	// Sidebar sections the user opened or closed, by repository path then
	// section id; a section not listed falls back to its default.
	SidebarSections map[string]map[string]bool `json:"sidebarSections,omitempty"`

	WindowWidth  int `json:"windowWidth,omitempty"`
	WindowHeight int `json:"windowHeight,omitempty"`

	OpenTabs   []string `json:"openTabs,omitempty"`
	FocusedTab string   `json:"focusedTab,omitempty"`

	NavCollapsed bool `json:"navCollapsed,omitempty"`

	// Commands, by repository path. TaskApprovals holds what each approval saw.
	CustomTasks   map[string][]CustomTask      `json:"customTasks,omitempty"`
	TaskApprovals map[string]map[string]string `json:"taskApprovals,omitempty"`
	TaskChecklist map[string]TaskChecklist     `json:"taskChecklist,omitempty"`
	Stacks        map[string][]StackLink       `json:"stacks,omitempty"`
	StackUndo     map[string]StackUndo         `json:"stackUndo,omitempty"`

	// The terminal panel's shell on Windows: "", "git-bash" or "cmd". "" is PowerShell.
	TerminalShell string `json:"terminalShell,omitempty"`

	// Snapshots kept per repository; zero means the default.
	SnapshotLimit int `json:"snapshotLimit,omitempty"`

	// Rules of the staged changes' risk report that are turned off.
	RiskRulesOff []string `json:"riskRulesOff,omitempty"`

	// Working Copy's diff takes the file list's space too.
	DiffExpanded bool `json:"diffExpanded,omitempty"`
	// Zero keeps long diff lines wrapping, as they always have.
	DiffNoWrap bool `json:"diffNoWrap,omitempty"`
	// "compact" tightens History, the file list and the sidebar; empty is comfortable.
	Density string `json:"density,omitempty"`

	// History's branch filter: "" is all branches, "HEAD" the current branch,
	// anything else a branch name.
	HistoryBranchFilter string `json:"historyBranchFilter,omitempty"`
	// Whether History follows only each merge's first parent.
	HistoryFirstParent bool `json:"historyFirstParent,omitempty"`

	// Whether the header's fetch also deletes remote-tracking branches gone from the remote.
	PruneOnFetch bool `json:"pruneOnFetch,omitempty"`

	// Zero keeps the Changelog tab and commit type picker on.
	DisableConventionalCommits bool `json:"disableConventionalCommits,omitempty"`

	Changelog *ChangelogPrefs `json:"changelog,omitempty"`

	// The files the release readiness check looks for the version in, by
	// repository path, each relative to its root.
	ReleaseVersionFiles map[string][]string `json:"releaseVersionFiles,omitempty"`

	Hosts map[string]HostConfig `json:"hosts,omitempty"`

	// Issue trackers beside the hosts, such as a team's Jira site. A
	// repository path is in at most one tracker's Repos.
	Trackers []TrackerConfig `json:"trackers,omitempty"`
	// Each tracker's integration, by site, including the organization's
	// default: whether it may be contacted and whether a token was saved.
	TrackerAccess map[string]TrackerAccess `json:"trackerAccess,omitempty"`
}

type TrackerAccess struct {
	Enabled  bool `json:"enabled,omitempty"`
	HasToken bool `json:"hasToken,omitempty"`
}

type TrackerConfig struct {
	Site  string   `json:"site"`
	Kind  string   `json:"kind"`
	Keys  []string `json:"keys,omitempty"`
	Repos []string `json:"repos,omitempty"`
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

// Save writes a temporary file beside path and renames it over path, so a
// crash mid-save leaves the previous settings rather than a truncated file.
func Save(path string, cfg Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "config-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
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
