package gitexec

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// LookPath finds a program as Git would, including the directories the executor
// adds to Git's PATH: Git for Windows keeps gpg and ssh-keygen in one of those.
func LookPath(name string) (string, error) {
	defaultBinary.mu.RLock()
	env := defaultBinary.env
	defaultBinary.mu.RUnlock()
	if !strings.ContainsAny(name, `/\`) {
		for _, entry := range env {
			path, ok := strings.CutPrefix(entry, "PATH=")
			if !ok {
				continue
			}
			for _, dir := range filepath.SplitList(path) {
				if found, err := exec.LookPath(filepath.Join(dir, name)); err == nil {
					return found, nil
				}
			}
		}
	}
	return exec.LookPath(name)
}
