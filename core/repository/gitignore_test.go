package repository

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddToGitignoreCreatesFile(t *testing.T) {
	dir := t.TempDir()

	if err := AddToGitignore(dir, "*.log"); err != nil {
		t.Fatalf("AddToGitignore: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "*.log\n" {
		t.Fatalf("got %q, want %q", content, "*.log\n")
	}
}

func TestAddToGitignoreAppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := AddToGitignore(dir, "*.log"); err != nil {
		t.Fatalf("AddToGitignore: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "node_modules/\n*.log\n" {
		t.Fatalf("got %q, want %q", content, "node_modules/\n*.log\n")
	}
}

func TestAddToGitignoreSkipsDuplicate(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := AddToGitignore(dir, "*.log"); err != nil {
		t.Fatalf("AddToGitignore: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "*.log\n" {
		t.Fatalf("got %q, want no duplicate: %q", content, "*.log\n")
	}
}

func TestAddToGitignoreHandlesMissingTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := AddToGitignore(dir, "*.log"); err != nil {
		t.Fatalf("AddToGitignore: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "node_modules/\n*.log\n" {
		t.Fatalf("got %q, want %q", content, "node_modules/\n*.log\n")
	}
}
