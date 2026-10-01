//go:build windows

package platform

import "os/exec"

func RevealInFileManager(dir string) error {
	return exec.Command("explorer.exe", dir).Start()
}
