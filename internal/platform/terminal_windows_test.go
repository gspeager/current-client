//go:build windows

package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitBashPathFindsSiblingLauncher(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "cmd")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	gitExe := filepath.Join(binDir, "git.exe")
	if err := os.WriteFile(gitExe, nil, 0o644); err != nil {
		t.Fatalf("WriteFile git.exe: %v", err)
	}
	bashExe := filepath.Join(root, "git-bash.exe")
	if err := os.WriteFile(bashExe, nil, 0o644); err != nil {
		t.Fatalf("WriteFile git-bash.exe: %v", err)
	}

	if got := gitBashPath(gitExe); got != bashExe {
		t.Fatalf("gitBashPath(%q) = %q, want %q", gitExe, got, bashExe)
	}
}

func TestGitBashPathMissingLauncherReturnsEmpty(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "cmd")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	gitExe := filepath.Join(binDir, "git.exe")
	if err := os.WriteFile(gitExe, nil, 0o644); err != nil {
		t.Fatalf("WriteFile git.exe: %v", err)
	}
	// No git-bash.exe written — e.g. a portable/minimal git install.

	if got := gitBashPath(gitExe); got != "" {
		t.Fatalf("gitBashPath(%q) = %q, want empty", gitExe, got)
	}
}

func TestGitBashPathEmptyInputReturnsEmpty(t *testing.T) {
	if got := gitBashPath(""); got != "" {
		t.Fatalf("gitBashPath(\"\") = %q, want empty", got)
	}
}
