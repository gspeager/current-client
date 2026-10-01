package repository

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenRepositoryWithSpaceInPath(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my repo")
	if _, err := Init(context.Background(), dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	repo, err := Open(context.Background(), dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if repo.Path != dir {
		t.Fatalf("Path = %q, want %q", repo.Path, dir)
	}
}

func TestOpenRepositoryWithUnicodePath(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "café_日本語_répo")
	if _, err := Init(context.Background(), dir); err != nil {
		t.Fatalf("Init: %v", err)
	}

	repo, err := Open(context.Background(), dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if repo.Path != dir {
		t.Fatalf("Path = %q, want %q", repo.Path, dir)
	}
}
