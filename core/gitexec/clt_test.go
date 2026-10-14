package gitexec

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
)

// fakeMac makes "git" resolve to path and xcode-select -p exit with code.
func fakeMac(t *testing.T, path string, code int) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		t.Skip("macOS only")
	}
	stub := filepath.Join(t.TempDir(), "xcode-select")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nexit "+strconv.Itoa(code)+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	oldLookPath, oldXcodeSelect := lookPath, xcodeSelect
	lookPath = func(string) (string, error) { return path, nil }
	xcodeSelect = stub
	toolsFound.Store(false)
	t.Cleanup(func() {
		lookPath, xcodeSelect = oldLookPath, oldXcodeSelect
		toolsFound.Store(false)
	})
}

func TestStandInGitIsNotRunWithoutCommandLineTools(t *testing.T) {
	fakeMac(t, "/usr/bin/git", 2)

	if !CommandLineToolsMissing() {
		t.Fatal("CommandLineToolsMissing = false, want true")
	}
	if _, err := NewExecutor("git").Run(context.Background(), Command{Args: []string{"--version"}}); !errors.Is(err, ErrCommandLineToolsMissing) {
		t.Errorf("Run err = %v, want ErrCommandLineToolsMissing", err)
	}
	if _, err := DetectSupported(context.Background(), ""); !errors.Is(err, ErrCommandLineToolsMissing) {
		t.Errorf("DetectSupported err = %v, want ErrCommandLineToolsMissing", err)
	}
	if err := WrapRunError(ErrCommandLineToolsMissing); err != ErrCommandLineToolsMissing {
		t.Errorf("WrapRunError = %v, want the error itself", err)
	}
}

func TestCommandLineToolsFound(t *testing.T) {
	fakeMac(t, "/usr/bin/git", 0)
	if CommandLineToolsMissing() {
		t.Error("xcode-select -p succeeded, want the tools found")
	}
}

func TestAnotherGitNeedsNoCommandLineTools(t *testing.T) {
	fakeMac(t, "/opt/homebrew/bin/git", 2)
	if CommandLineToolsMissing() {
		t.Error("a Homebrew git first on PATH, want no check")
	}
}
