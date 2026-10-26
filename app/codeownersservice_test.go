package app

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCodeownersOwnersAndSuggestedReviewers(t *testing.T) {
	useTempSettings(t)
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	gitIn(t, repo, "config", "user.email", "t@example.com")
	gitIn(t, repo, "config", "user.name", "T")
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
	gitIn(t, repo, "add", ".")
	gitIn(t, repo, "commit", "-q", "-m", "initial")
	gitIn(t, repo, "checkout", "-q", "-b", "feature")
	write("docs/guide.md", "guide")
	write("src/a.go", "package a")
	write("other.txt", "x")
	gitIn(t, repo, "add", ".")
	gitIn(t, repo, "commit", "-q", "-m", "feature")

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
	gitIn(t, repo, "init", "-q", "-b", "main")
	svc := &CodeownersService{}
	owners, err := svc.Owners(repo, []string{"a"})
	if err != nil || len(owners) != 0 {
		t.Fatalf("got %v, %v", owners, err)
	}
}
