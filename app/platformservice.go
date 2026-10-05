package app

import (
	"context"
	"os"

	"github.com/gspeager/current-client/core/gitexec"
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
	return platform.OpenInEditor(repoPath, configOrDefault().EditorPath)
}

func (s *PlatformService) OpenFileInEditor(repoPath, path string, line int) error {
	return platform.OpenFileInEditor(repoPath, path, line, configOrDefault().EditorPath)
}

// appVersion is set by the Taskfiles with -ldflags "-X github.com/gspeager/current-client/app.appVersion=<version>";
// a plain go build leaves it as "dev".
var appVersion = "dev"

func (s *PlatformService) AppVersion() string {
	return appVersion
}

type RecentErrorInfo struct {
	At      string `json:"at"`
	Message string `json:"message"`
}

// GetDiagnostics takes the errors the UI has shown, newest first, since only
// the frontend knows which failures reached the user.
func (s *PlatformService) GetDiagnostics(recentErrors []RecentErrorInfo) string {
	cfg := configOrDefault()
	in := platform.DiagnosticsInput{
		AppVersion:      appVersion,
		SettingsVersion: cfg.Version,
		RecentErrors:    mapSlice(recentErrors, func(e RecentErrorInfo) platform.RecentError { return platform.RecentError(e) }),
		RepoPaths:       append(append([]string(nil), cfg.RecentRepositories...), cfg.OpenTabs...),
	}
	in.HomeDir, _ = os.UserHomeDir()
	info, err := gitexec.DetectSupported(context.Background(), cfg.GitExecutablePath)
	if err != nil {
		in.GitProblem = err.Error()
	}
	in.GitVersion, in.GitPath = info.Version, info.Path
	return platform.Diagnostics(in)
}

// SetDarkTitleBar matches the calling window's native title bar to the app's
// theme. It only has an effect on Windows.
func (s *PlatformService) SetDarkTitleBar(ctx context.Context, dark bool) {
	if window, ok := ctx.Value(application.WindowKey).(application.Window); ok {
		application.InvokeAsync(func() { setDarkTitleBar(window, dark) })
	}
}
