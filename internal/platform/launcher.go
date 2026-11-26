package platform

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Launcher is the terminal command that opens a path in the app. Path is where it
// is, or where installing writes it. ByPackage means a package put another
// command at Path, so there's nothing to install. PathLine is the line to add to
// the shell's startup file when Path's folder isn't on the PATH.
type Launcher struct {
	Path      string `json:"path"`
	Installed bool   `json:"installed"`
	ByPackage bool   `json:"byPackage"`
	OnPath    bool   `json:"onPath"`
	PathLine  string `json:"pathLine"`
}

const (
	launcherName   = "current-client"
	launcherMarker = "Current Client's terminal command"
)

var ErrNotOurs = errors.New("not Current Client's terminal command")

// LauncherStatus reads what's installed for the app at target.
func LauncherStatus(target string) (Launcher, error) {
	path, err := launcherPath()
	if err != nil {
		return Launcher{}, err
	}
	content, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Launcher{}, err
	}
	launcher := Launcher{Path: path, Installed: string(content) == launcherScript(target)}
	if found, err := exec.LookPath(launcherName); err == nil && !sameFile(found, path) && !isLauncher(found) {
		return Launcher{Path: found, ByPackage: true, Installed: true, OnPath: true}, nil
	}
	launcher.OnPath, launcher.PathLine = onPath(filepath.Dir(path))
	return launcher, nil
}

// InstallLauncher writes the command for the app at target, replacing one written
// for another copy of the app but nothing else.
func InstallLauncher(target string) (Launcher, error) {
	path, err := launcherPath()
	if err != nil {
		return Launcher{}, err
	}
	if _, err := os.Stat(path); err == nil && !isLauncher(path) {
		return Launcher{}, &fs.PathError{Op: "install", Path: path, Err: ErrNotOurs}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Launcher{}, err
	}
	written := path + ".new"
	if err := os.WriteFile(written, []byte(launcherScript(target)), 0o755); err != nil {
		return Launcher{}, err
	}
	if err := os.Rename(written, path); err != nil {
		_ = os.Remove(written)
		return Launcher{}, err
	}
	if err := addToPath(filepath.Dir(path)); err != nil {
		return Launcher{}, err
	}
	return LauncherStatus(target)
}

// RemoveLauncher takes out the command, only if it's the app's own.
func RemoveLauncher(target string) (Launcher, error) {
	path, err := launcherPath()
	if err != nil {
		return Launcher{}, err
	}
	if !isLauncher(path) {
		return LauncherStatus(target)
	}
	if err := os.Remove(path); err != nil {
		return Launcher{}, err
	}
	if err := removeFromPath(filepath.Dir(path)); err != nil {
		return Launcher{}, err
	}
	return LauncherStatus(target)
}

func isLauncher(path string) bool {
	content, err := os.ReadFile(path)
	return err == nil && strings.Contains(string(content), launcherMarker)
}

func sameFile(a, b string) bool {
	first, err := os.Stat(a)
	if err != nil {
		return false
	}
	second, err := os.Stat(b)
	return err == nil && os.SameFile(first, second)
}

// withDir and withoutDir edit a PATH list, leaving the rest as it is.
func withDir(list, dir string) string {
	if hasDir(list, dir) {
		return list
	}
	if list == "" {
		return dir
	}
	return list + string(os.PathListSeparator) + dir
}

func withoutDir(list, dir string) string {
	var kept []string
	for _, entry := range filepath.SplitList(list) {
		if !sameDir(entry, dir) {
			kept = append(kept, entry)
		}
	}
	return strings.Join(kept, string(os.PathListSeparator))
}

func hasDir(list, dir string) bool {
	for _, entry := range filepath.SplitList(list) {
		if sameDir(entry, dir) {
			return true
		}
	}
	return false
}
