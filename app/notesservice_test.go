package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gspeager/current-client/internal/config"
)

func notesRepo(t *testing.T) string {
	t.Helper()
	useTempSettings(t)
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	gitIn(t, repo, "config", "user.name", "T")
	gitIn(t, repo, "config", "user.email", "t@example.com")
	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "add", "a.txt")
	gitIn(t, repo, "commit", "-q", "-m", "first")
	gitIn(t, repo, "branch", "side")
	gitIn(t, repo, "tag", "v1")
	return repo
}

func noteOn(t *testing.T, repo, kind, target string) *NoteInfo {
	t.Helper()
	all, err := new(NotesService).List(repo)
	if err != nil {
		t.Fatal(err)
	}
	for i := range all {
		if all[i].Kind == kind && all[i].Target == target {
			return &all[i]
		}
	}
	return nil
}

func TestNotesAddEditAndRemoveOnEachKind(t *testing.T) {
	repo := notesRepo(t)
	head := strings.TrimSpace(string(mustOutput(t, repo, "rev-parse", "HEAD")))
	var s NotesService
	targets := map[string]string{"commit": head, "branch": "side", "tag": "v1", "file": "a.txt"}
	for kind, target := range targets {
		if err := s.Set(repo, kind, target, "first", "  why it was left\n"); err != nil {
			t.Fatal(err)
		}
		note := noteOn(t, repo, kind, target)
		if note == nil || note.Text != "why it was left" || note.Stale || note.Subject != "first" {
			t.Fatalf("%s note = %+v", kind, note)
		}
		if err := s.Set(repo, kind, target, "first", "edited"); err != nil {
			t.Fatal(err)
		}
		if note := noteOn(t, repo, kind, target); note == nil || note.Text != "edited" {
			t.Fatalf("%s edited = %+v", kind, note)
		}
		if err := s.Set(repo, kind, target, "first", " "); err != nil {
			t.Fatal(err)
		}
		if note := noteOn(t, repo, kind, target); note != nil {
			t.Fatalf("%s removed = %+v", kind, note)
		}
	}
	if err := s.Set(repo, "planet", "x", "", "y"); err == nil {
		t.Fatal("an unknown kind should be refused")
	}
}

func TestNotesGoStaleRatherThanBeingLost(t *testing.T) {
	repo := notesRepo(t)
	var s NotesService
	gitIn(t, repo, "switch", "-q", "-c", "work")
	if err := os.WriteFile(filepath.Join(repo, "b.txt"), []byte("b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "add", "b.txt")
	gitIn(t, repo, "commit", "-q", "-m", "to rewrite")
	rewritten := strings.TrimSpace(string(mustOutput(t, repo, "rev-parse", "HEAD")))
	for kind, target := range map[string]string{"commit": rewritten, "branch": "side", "tag": "v1", "file": "b.txt"} {
		if err := s.Set(repo, kind, target, "to rewrite", "keep me"); err != nil {
			t.Fatal(err)
		}
	}

	gitIn(t, repo, "commit", "-q", "--amend", "-m", "rewritten")
	gitIn(t, repo, "branch", "-q", "-D", "side")
	gitIn(t, repo, "tag", "-d", "v1")
	gitIn(t, repo, "rm", "-q", "b.txt")
	gitIn(t, repo, "commit", "-q", "-m", "remove")
	// The amended commit stays in the reflog, which doesn't count as reachable.

	for kind, target := range map[string]string{"commit": rewritten, "branch": "side", "tag": "v1", "file": "b.txt"} {
		note := noteOn(t, repo, kind, target)
		if note == nil || !note.Stale || note.Text != "keep me" {
			t.Fatalf("%s note = %+v", kind, note)
		}
	}
	if note := noteOn(t, repo, "commit", rewritten); note.Subject != "to rewrite" {
		t.Fatalf("subject = %q", note.Subject)
	}
}

func TestNotesNeverReachTheRepositoryOrItsRemote(t *testing.T) {
	repo := notesRepo(t)
	const secret = "private note text"
	var s NotesService
	if err := s.Set(repo, "branch", "main", "", secret); err != nil {
		t.Fatal(err)
	}
	bare := t.TempDir()
	gitIn(t, bare, "init", "-q", "--bare", "-b", "main")
	gitIn(t, repo, "remote", "add", "origin", bare)
	gitIn(t, repo, "push", "-q", "origin", "--mirror")
	clone := t.TempDir()
	gitIn(t, clone, "clone", "-q", bare, ".")
	gitIn(t, repo, "fetch", "-q", "origin")

	for _, dir := range []string{repo, bare, clone} {
		err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if strings.Contains(string(data), secret) {
				t.Errorf("the note reached %s", path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	settings, err := config.DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(filepath.Dir(settings), "notes.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("notes.json is %v", info.Mode().Perm())
	}
}

func TestWorktreesShareNotes(t *testing.T) {
	repo := notesRepo(t)
	linked := filepath.Join(t.TempDir(), "linked")
	gitIn(t, repo, "worktree", "add", "-q", linked, "side")
	var s NotesService
	if err := s.Set(repo, "branch", "main", "", "shared"); err != nil {
		t.Fatal(err)
	}
	if note := noteOn(t, linked, "branch", "main"); note == nil || note.Text != "shared" {
		t.Fatalf("from the worktree = %+v", note)
	}
}

func TestRemovingARepositoryKeepsItsNotes(t *testing.T) {
	repo := notesRepo(t)
	var s NotesService
	if err := s.Set(repo, "branch", "main", "", "keep"); err != nil {
		t.Fatal(err)
	}
	if err := new(RepositoryService).RemoveRecentRepository(repo); err != nil {
		t.Fatal(err)
	}
	if note := noteOn(t, repo, "branch", "main"); note == nil {
		t.Fatal("the note went with the repository's place on the recent list")
	}
}
