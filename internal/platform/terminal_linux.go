//go:build linux

package platform

import "os/exec"

// OpenTerminal tries common emulators, since no one terminal is standard.
func OpenTerminal(dir, gitBinaryPath string) error {
	candidates := [][]string{
		{"x-terminal-emulator", "--working-directory=" + dir},
		{"gnome-terminal", "--working-directory=" + dir},
		{"xterm", "-e", "bash"},
	}
	var lastErr error
	for _, args := range candidates {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		lastErr = cmd.Start()
		if lastErr == nil {
			return nil
		}
	}
	return lastErr
}
