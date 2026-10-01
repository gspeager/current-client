//go:build darwin

package platform

import "os/exec"

func RevealInFileManager(dir string) error {
	return exec.Command("open", dir).Start()
}
