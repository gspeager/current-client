package platform

import "os/exec"

// OpenInEditor defaults to VS Code's "code" command. Editors all treat a
// directory argument as "open this folder", so no per-OS handling is needed.
func OpenInEditor(dir, editorPath string) error {
	if editorPath == "" {
		editorPath = "code"
	}
	return exec.Command(editorPath, dir).Start()
}
