// Package gittest runs git in temporary repositories for tests. It's the one
// internal package core's tests may import: tests aren't built into core.
package gittest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

// Output is Run's stdout alone, for a command whose warnings go to stderr.
func Output(t testing.TB, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

// RunFails fails the test if git succeeds, for a step such as a merge that
// must stop on a conflict.
func RunFails(t testing.TB, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err == nil {
		t.Fatalf("git %v: expected failure, succeeded:\n%s", args, out)
	}
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

// CommitAt commits name with content, or nothing new when name is empty, with
// the author and committer dated when, in any form Git reads.
func CommitAt(t testing.TB, dir, name, content, message, when string) {
	t.Helper()
	args := []string{"commit", "-q", "--allow-empty", "-m", message}
	if name != "" {
		WriteFile(t, dir, name, content)
		Run(t, dir, "add", name)
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE="+when, "GIT_COMMITTER_DATE="+when)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
}

// DaysAgo is a date days before now, as CommitAt takes it.
func DaysAgo(days int) string {
	return fmt.Sprintf("%d +0000", time.Now().AddDate(0, 0, -days).Unix())
}
