package gitexec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitForWindowsFindsTheRealGitBehindTheLauncher(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"cmd", "mingw64/bin", "mingw64/libexec/git-core", "usr/bin"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []string{"cmd/git.exe", "mingw64/bin/git.exe"} {
		if err := os.WriteFile(filepath.Join(root, file), nil, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", "existing")
	t.Setenv("MSYSTEM", "")

	// git --exec-path prints forward slashes, even on Windows.
	binary, env, ok := gitForWindows(filepath.ToSlash(filepath.Join(root, "mingw64", "libexec", "git-core")) + "\n")
	if !ok {
		t.Fatal("ok = false, want the real git found")
	}
	if want := filepath.Join(root, "mingw64", "bin", "git.exe"); binary != want {
		t.Errorf("binary = %q, want %q", binary, want)
	}
	sep := string(os.PathListSeparator)
	wantPath := "PATH=" + filepath.Join(root, "mingw64", "bin") + sep + filepath.Join(root, "usr", "bin") + sep + "existing"
	if strings.Join(env, "\n") != wantPath+"\nMSYSTEM=MINGW64" {
		t.Errorf("env = %q, want %q and MSYSTEM=MINGW64", env, wantPath)
	}
}

func TestGitForWindowsKeepsAnExistingMSYSTEM(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "clangarm64", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "clangarm64", "bin", "git.exe"), nil, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MSYSTEM", "CLANGARM64")

	binary, env, ok := gitForWindows(filepath.Join(root, "clangarm64", "libexec", "git-core"))
	if !ok || binary != filepath.Join(root, "clangarm64", "bin", "git.exe") {
		t.Fatalf("binary, ok = %q, %v, want the Arm64 git", binary, ok)
	}
	if len(env) != 1 || !strings.HasPrefix(env[0], "PATH=") {
		t.Errorf("env = %q, want only PATH", env)
	}
}

func TestGitForWindowsRejectsAnotherLayout(t *testing.T) {
	// Cygwin's and Homebrew's git have no git.exe beside their exec path.
	if _, _, ok := gitForWindows("/usr/libexec/git-core"); ok {
		t.Error("ok = true for a git that isn't Git for Windows, want false")
	}
}
