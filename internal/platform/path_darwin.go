//go:build darwin

package platform

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

// UseLoginShellPath gives the app the PATH the user's login shell sets up,
// ahead of its own. Opened from Finder or the Dock, a Mac app only gets
// launchd's /usr/bin:/bin:/usr/sbin:/sbin, which misses Homebrew, VS Code's
// code command and credential helpers installed with them.
func UseLoginShellPath() {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, shell, "-l", "-c", `printf '%s%s' "`+pathMarker+`" "$PATH"`).Output()
	if err != nil {
		return
	}
	_, shellPath, found := strings.Cut(string(out), pathMarker)
	if !found || shellPath == "" {
		return
	}
	_ = os.Setenv("PATH", mergePath(strings.TrimSpace(shellPath), os.Getenv("PATH")))
}
