//go:build !windows

package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// A folder in HOME, so installing needs no administrator. It goes on the PATH in
// the shell's startup file, which the app doesn't edit.
func launcherPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "bin", launcherName), nil
}

// The app runs apart from the terminal, so the command returns at once, on the
// first launch too.
func launcherScript(target string) string {
	return "#!/bin/sh\n# " + launcherMarker + ". Settings, Tools and folders, removes it.\n" +
		"nohup " + shellQuote(target) + " \"$@\" >/dev/null 2>&1 &\n"
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// The app's PATH is the login shell's (UseLoginShellPath) or the desktop
// session's, so the line goes in a file those read, not ~/.zshrc or ~/.bashrc.
func onPath(dir string) (bool, string, string) {
	if hasDir(os.Getenv("PATH"), dir) {
		return true, "", ""
	}
	line := `export PATH="$HOME/.local/bin:$PATH"`
	switch filepath.Base(os.Getenv("SHELL")) {
	case "fish":
		return false, "fish_add_path ~/.local/bin", "~/.config/fish/config.fish"
	case "zsh":
		return false, line, "~/.zprofile"
	case "bash":
		if runtime.GOOS == "darwin" {
			return false, line, "~/.bash_profile"
		}
	}
	return false, line, "~/.profile"
}

func addToPath(string) error      { return nil }
func removeFromPath(string) error { return nil }

func sameDir(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}
