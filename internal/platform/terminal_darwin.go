//go:build darwin

package platform

import "os/exec"

func OpenTerminal(dir, gitBinaryPath string) error {
	return exec.Command("open", "-a", "Terminal", dir).Start()
}
