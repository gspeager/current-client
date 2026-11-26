package app

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gspeager/current-client/internal/gittest"
)

func TestCodeownersOwnersAndSuggestedReviewers(t *testing.T) {
	useTempSettings(t)
	repo := t.TempDir()
	gittest.Run(t, repo, "init", "-q", "-b", "main")
	gittest.Run(t, repo, "config", "user.email", "t@example.com")
	gittest.Run(t, repo, "config", "user.name", "T")
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".github/CODEOWNERS", "src/ @alice\ndocs/ @bob @alice\n")
	write("README.md", "hi")
	gittest.Run(t, repo, "add", ".")
	gittest.Run(t, repo, "commit", "-q", "-m", "initial")
	gittest.Run(t, repo, "checkout", "-q", "-b", "feature")
	write("docs/guide.md", "guide")
	write("src/a.go", "package a")
	write("other.txt", "x")
	gittest.Run(t, repo, "add", ".")
	gittest.Run(t, repo, "commit", "-q", "-m", "feature")

	svc := &CodeownersService{}
	owners, err := svc.Owners(repo, []string{"src/a.go", "other.txt", "docs/guide.md"})
	if err != nil {
		t.Fatalf("Owners: %v", err)
	}
	want := map[string][]string{"src/a.go": {"@alice"}, "docs/guide.md": {"@bob", "@alice"}}
	if !reflect.DeepEqual(owners, want) {
		t.Fatalf("Owners: got %v, want %v", owners, want)
	}

	reviewers, err := svc.SuggestedReviewers(repo, "feature", "main")
	if err != nil {
		t.Fatalf("SuggestedReviewers: %v", err)
	}
	if got := reviewers; !reflect.DeepEqual(got, []string{"@bob", "@alice"}) {
		t.Fatalf("SuggestedReviewers: got %v", got)
	}

	if _, err := svc.SuggestedReviewers(repo, "feature", "missing"); err == nil {
		t.Fatal("an unknown target should fail, not suggest nobody")
	}
}

func TestCodeownersWithoutAFile(t *testing.T) {
	useTempSettings(t)
	repo := t.TempDir()
	gittest.Run(t, repo, "init", "-q", "-b", "main")
	svc := &CodeownersService{}
	owners, err := svc.Owners(repo, []string{"a"})
	if err != nil || len(owners) != 0 {
		t.Fatalf("got %v, %v", owners, err)
	}
}
