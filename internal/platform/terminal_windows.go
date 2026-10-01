//go:build windows

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
)

// OpenTerminal prefers Git Bash, then Windows Terminal, then cmd.exe.
func OpenTerminal(dir, gitBinaryPath string) error {
	if bash := gitBashPath(gitBinaryPath); bash != "" {
		return start(bash, dir)
	}
	if wt, err := exec.LookPath("wt.exe"); err == nil {
		return start(wt, dir, "-d", dir)
	}
	return start("cmd.exe", dir)
}

// Git for Windows ships git-bash.exe two levels above bin/git.exe or cmd/git.exe.
func gitBashPath(gitBinaryPath string) string {
	if gitBinaryPath == "" {
		return ""
	}
	root := filepath.Dir(filepath.Dir(gitBinaryPath))
	candidate := filepath.Join(root, "git-bash.exe")
	if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
		return candidate
	}
	return ""
}

func start(name, dir string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.Start()
}
