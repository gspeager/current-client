package gitexec

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

// Without CREATE_NO_WINDOW each git call from the windowsgui build opens a
// console window, which is also somewhere ssh could sit waiting for input.
func detachFromTerminal(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}
