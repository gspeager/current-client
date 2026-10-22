package app

import (
	"context"
	"errors"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
	"github.com/gspeager/current-client/internal/config"
	"github.com/gspeager/current-client/internal/platform"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type PlatformService struct{}

const (
	gitDownloadURL         = "https://git-scm.com/downloads"
	conventionalCommitsURL = "https://www.conventionalcommits.org"
)

func (s *PlatformService) OpenGitDownloadPage() error {
	return application.Get().Browser.OpenURL(gitDownloadURL)
}

func (s *PlatformService) OpenConventionalCommitsPage() error {
	return application.Get().Browser.OpenURL(conventionalCommitsURL)
}

// OpenWebPage opens a page on the repository's own host in the user's
// browser. The frontend only passes URLs built from the repository's remotes.
func (s *PlatformService) OpenWebPage(pageURL string) error {
	if err := checkWebPageURL(pageURL); err != nil {
		return err
	}
	return application.Get().Browser.OpenURL(pageURL)
}

func checkWebPageURL(pageURL string) error {
	u, err := url.Parse(pageURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return &gitexec.AppError{Message: "That isn't a web address.", Detail: pageURL}
	}
	return nil
}

func (s *PlatformService) OpenTerminal(repoPath string) error {
	gitPath := ""
	if info, err := gitexec.Detect(context.Background(), configOrDefault().GitExecutablePath); err == nil {
		gitPath = info.Path
	}
	return platform.OpenTerminal(repoPath, gitPath)
}

func (s *PlatformService) RevealInFileManager(repoPath string) error {
	return platform.RevealInFileManager(repoPath)
}

func (s *PlatformService) OpenInEditor(repoPath string) error {
	editorPath := configOrDefault().EditorPath
	return editorError(platform.OpenInEditor(repoPath, editorPath), editorPath)
}

func (s *PlatformService) OpenFileInEditor(repoPath, path string, line int) error {
	editorPath := configOrDefault().EditorPath
	return editorError(platform.OpenFileInEditor(repoPath, path, line, editorPath), editorPath)
}

// editorError explains a missing editor in place of Go's exec error.
func editorError(err error, editorPath string) error {
	if !errors.Is(err, exec.ErrNotFound) && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if editorPath == "" {
		return &gitexec.AppError{Message: "VS Code's code command isn't installed. Choose an editor in Settings.", Detail: err.Error()}
	}
	return &gitexec.AppError{Message: "The editor " + editorPath + " wasn't found. Choose another in Settings.", Detail: err.Error()}
}

// appVersion is set by the Taskfiles with -ldflags "-X github.com/gspeager/current-client/app.appVersion=<version>";
// a plain go build leaves it as "dev".
var appVersion = "dev"

func (s *PlatformService) AppVersion() string {
	return appVersion
}

func (s *PlatformService) HomeDir() string {
	home, _ := os.UserHomeDir()
	return home
}

type RecentErrorInfo struct {
	At      string `json:"at"`
	Message string `json:"message"`
}

// GetDiagnostics takes the errors the UI has shown, newest first, since only
// the frontend knows which failures reached the user.
func (s *PlatformService) GetDiagnostics(recentErrors []RecentErrorInfo, omitHostnames bool) string {
	cfg := configOrDefault()
	in := platform.DiagnosticsInput{
		AppVersion:      appVersion,
		SettingsVersion: cfg.Version,
		RecentErrors:    mapSlice(recentErrors, func(e RecentErrorInfo) platform.RecentError { return platform.RecentError(e) }),
		RepoPaths:       append(append([]string(nil), cfg.RecentRepositories...), cfg.OpenTabs...),
		HostIntegration: hostIntegrationSummary(cfg),
		Hostnames:       knownHostnames(cfg),
		OmitHostnames:   omitHostnames,
	}
	in.HomeDir, _ = os.UserHomeDir()
	info, err := gitexec.DetectSupported(context.Background(), cfg.GitExecutablePath)
	if err != nil {
		in.GitProblem = err.Error()
	}
	in.GitVersion, in.GitPath = info.Version, info.Path
	return platform.Diagnostics(in)
}

// HostIntegrationAvailable is false in a build made with the nohost tag,
// which has no host integration code at all; the frontend then shows none
// of its UI.
func (s *PlatformService) HostIntegrationAvailable() bool {
	return hostIntegrationBuilt
}

func hostIntegrationSummary(cfg config.Config) string {
	if !hostIntegrationBuilt {
		return "not built in"
	}
	var on []string
	for name, host := range cfg.Hosts {
		if host.Enabled {
			on = append(on, name)
		}
	}
	if len(on) == 0 {
		return "off"
	}
	sort.Strings(on)
	return "on for " + strings.Join(on, ", ")
}

// knownHostnames are the hosts in settings and their API hosts, which is
// where an error message would name one.
func knownHostnames(cfg config.Config) []string {
	var names []string
	for name := range cfg.Hosts {
		names = append(names, name, "api."+name)
	}
	return names
}

// SetDarkTitleBar matches the calling window's native title bar to the app's
// theme. It only has an effect on Windows.
func (s *PlatformService) SetDarkTitleBar(ctx context.Context, dark bool) {
	if window, ok := ctx.Value(application.WindowKey).(application.Window); ok {
		application.InvokeAsync(func() { setDarkTitleBar(window, dark) })
	}
}
