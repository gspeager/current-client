package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// OpenInEditor defaults to VS Code's "code" command. Editors all treat a
// directory argument as "open this folder", so no per-OS handling is needed.
func OpenInEditor(dir, editorPath string) error {
	return exec.Command(editorOrDefault(editorPath), dir).Start()
}

// OpenFileInEditor opens file (relative to dir) at line. Only VS Code and the
// editors built on it share a way to say the line, -g file:line; others just
// open the file.
func OpenFileInEditor(dir, file string, line int, editorPath string) error {
	editor := editorOrDefault(editorPath)
	full := filepath.Join(dir, filepath.FromSlash(file))
	return exec.Command(editor, editorFileArgs(editor, dir, full, line)...).Start()
}

func editorFileArgs(editor, dir, full string, line int) []string {
	base := strings.ToLower(editor[strings.LastIndexAny(editor, `/\`)+1:])
	name := strings.TrimSuffix(base, filepath.Ext(base))
	switch name {
	case "code", "code-insiders", "codium", "cursor", "windsurf":
		return []string{dir, "-g", full + ":" + strconv.Itoa(line)}
	default:
		return []string{full}
	}
}

// editorOrDefault falls back to the code command inside VS Code's app bundle
// on macOS, where it's often not on PATH.
func editorOrDefault(editorPath string) string {
	if editorPath != "" {
		return editorPath
	}
	if _, err := exec.LookPath("code"); err == nil || runtime.GOOS != "darwin" {
		return "code"
	}
	home, _ := os.UserHomeDir()
	for _, apps := range []string{"/Applications", filepath.Join(home, "Applications")} {
		bundled := filepath.Join(apps, "Visual Studio Code.app", "Contents", "Resources", "app", "bin", "code")
		if _, err := os.Stat(bundled); err == nil {
			return bundled
		}
	}
	return "code"
}
