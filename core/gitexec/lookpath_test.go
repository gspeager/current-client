package gitexec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLookPathSearchesGitsPath(t *testing.T) {
	dir := t.TempDir()
	program := filepath.Join(dir, "only-beside-git")
	if err := os.WriteFile(program, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := LookPath("only-beside-git"); err == nil {
		t.Fatal("found a program outside PATH")
	}

	defaultBinary.mu.Lock()
	saved := defaultBinary.env
	defaultBinary.env = []string{"PATH=" + dir}
	defaultBinary.mu.Unlock()
	t.Cleanup(func() {
		defaultBinary.mu.Lock()
		defaultBinary.env = saved
		defaultBinary.mu.Unlock()
	})

	if found, err := LookPath("only-beside-git"); err != nil || found != program {
		t.Fatalf("LookPath = %q, %v, want %q", found, err, program)
	}
}
