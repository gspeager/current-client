package app

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"

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
