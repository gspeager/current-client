//go:build linux

package platform

import "os/exec"

func RevealInFileManager(dir string) error {
	return exec.Command("xdg-open", dir).Start()
}
