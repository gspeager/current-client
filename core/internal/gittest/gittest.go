// Package gittest runs git in temporary repositories for core's tests.
package gittest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Run fails the test if git does, and returns its trimmed output.
func Run(t testing.TB, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// InitRepo creates a repository on main with a test identity and no line
// ending conversion, so tests see the bytes they wrote.
func InitRepo(t testing.TB) string {
	t.Helper()
	dir := t.TempDir()
	Run(t, dir, "init", "-b", "main")
	Run(t, dir, "config", "user.email", "test@example.com")
	Run(t, dir, "config", "user.name", "Test")
	Run(t, dir, "config", "core.autocrlf", "false")
	return dir
}

func WriteFile(t testing.TB, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func CommitFile(t testing.TB, dir, name, content, message string) {
	t.Helper()
	WriteFile(t, dir, name, content)
	Run(t, dir, "add", name)
	Run(t, dir, "commit", "-m", message)
}
