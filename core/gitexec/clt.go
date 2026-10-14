package gitexec

import (
	"os/exec"
	"runtime"
	"sync/atomic"
)

// ErrCommandLineToolsMissing is returned instead of running macOS's stand-in
// /usr/bin/git, which opens Apple's install window and fails until Apple's
// Command Line Tools are installed.
var ErrCommandLineToolsMissing = &AppError{Message: "Apple's Command Line Tools aren't installed. Git comes with them."}

var (
	lookPath    = exec.LookPath
	xcodeSelect = "/usr/bin/xcode-select"
	toolsFound  atomic.Bool
)

// CommandLineToolsMissing says whether "git" on PATH is macOS's stand-in,
// without running it: xcode-select -p answers without Apple's window. Once
// they're found, or another Git is first on PATH, it stops checking.
func CommandLineToolsMissing() bool {
	if runtime.GOOS != "darwin" || toolsFound.Load() {
		return false
	}
	path, err := lookPath("git")
	if err != nil {
		return false
	}
	if path != "/usr/bin/git" || exec.Command(xcodeSelect, "-p").Run() == nil {
		toolsFound.Store(true)
		return false
	}
	return true
}

// InstallCommandLineTools opens Apple's installer for them, which downloads
// them and takes a few minutes.
func InstallCommandLineTools() error {
	return exec.Command(xcodeSelect, "--install").Run()
}
