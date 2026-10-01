package gitexec

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writeStubGit(t *testing.T, version string) string {
	t.Helper()
	dir := t.TempDir()

	var path, content string
	if runtime.GOOS == "windows" {
		path = filepath.Join(dir, "git.bat")
		content = "@echo git version " + version + "\r\n"
	} else {
		path = filepath.Join(dir, "git")
		content = "#!/bin/sh\necho git version " + version + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return dir
}

func TestDetectStubbedPath(t *testing.T) {
	t.Setenv("PATH", writeStubGit(t, "9.9.9.fake"))

	info, err := Detect(context.Background(), "")
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if info.Version != "9.9.9.fake" {
		t.Fatalf("got version %q, want %q", info.Version, "9.9.9.fake")
	}
	if info.Path == "" {
		t.Fatal("expected a resolved path")
	}
}

func TestDetectNoGitOnPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	if _, err := Detect(context.Background(), ""); err == nil {
		t.Fatal("expected an error when git is not on PATH")
	}
}

func TestDetectRealGit(t *testing.T) {
	info, err := Detect(context.Background(), "")
	if err != nil {
		t.Skipf("git not available in test environment: %v", err)
	}
	if info.Version == "" {
		t.Fatal("expected a non-empty version")
	}
}

func TestDetectWithOverridePath(t *testing.T) {
	dir := writeStubGit(t, "1.2.3.override")
	binName := "git"
	if runtime.GOOS == "windows" {
		binName = "git.bat"
	}
	overridePath := filepath.Join(dir, binName)

	// PATH has no git, so success proves the override was used.
	t.Setenv("PATH", t.TempDir())

	info, err := Detect(context.Background(), overridePath)
	if err != nil {
		t.Fatalf("Detect with override: %v", err)
	}
	if info.Version != "1.2.3.override" {
		t.Fatalf("got version %q, want %q", info.Version, "1.2.3.override")
	}
	if info.Path != overridePath {
		t.Fatalf("got path %q, want %q", info.Path, overridePath)
	}
}

func TestDetectWithInvalidOverridePathErrors(t *testing.T) {
	if _, err := Detect(context.Background(), filepath.Join(t.TempDir(), "not-a-real-binary")); err == nil {
		t.Fatal("expected an error for a nonexistent override path")
	}
}

func TestVersionAtLeast(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"2.23.0", true},
		{"2.23", true},
		{"2.45.1.windows.1", true},
		{"2.39.3 (Apple Git-146)", true},
		{"3.0.0", true},
		{"2.22.5", false},
		{"2.9.0", false},
		{"1.99.0", false},
		{"not a version", false},
	}
	for _, tt := range tests {
		if got := versionAtLeast(tt.version, "2.23"); got != tt.want {
			t.Errorf("versionAtLeast(%q, 2.23) = %v, want %v", tt.version, got, tt.want)
		}
	}
}

func TestDetectSupportedRejectsOldGit(t *testing.T) {
	t.Setenv("PATH", writeStubGit(t, "2.20.1"))

	_, err := DetectSupported(context.Background(), "")
	if err == nil || err.Error() != "Git 2.20.1 found; 2.23 or newer is required." {
		t.Fatalf("err = %v, want the too-old message", err)
	}
}

func TestDetectSupportedReportsMissingGit(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	_, err := DetectSupported(context.Background(), "")
	if err == nil || err.Error() != "Git not found." {
		t.Fatalf("err = %v, want \"Git not found.\"", err)
	}
}

func TestDetectSupportedAcceptsCurrentGit(t *testing.T) {
	t.Setenv("PATH", writeStubGit(t, "2.45.1.windows.1"))

	info, err := DetectSupported(context.Background(), "")
	if err != nil {
		t.Fatalf("DetectSupported: %v", err)
	}
	if info.Version != "2.45.1.windows.1" {
		t.Fatalf("Version = %q", info.Version)
	}
}
