//go:build !windows

package gitexec

import (
	"os/exec"
	"syscall"
)

// A new session has no controlling terminal, so ssh can't prompt for a
// passphrase or host key on the terminal the app was launched from.
func detachFromTerminal(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
