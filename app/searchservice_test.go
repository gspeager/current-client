package app

import (
	"context"
	"slices"
	"testing"

	"os"
	"path/filepath"

	"github.com/gspeager/current-client/core/git"
)

func TestFindingInEachOpenRepository(t *testing.T) {
	first, second := notesRepo(t), notesRepo(t)
	writeIn(t, first, ".gitignore", "build/\n")
	if err := os.MkdirAll(filepath.Join(first, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeAndCommit(t, first, "src/login.go", "package src", "Fix the login redirect")
	if err := os.MkdirAll(filepath.Join(first, "build"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeIn(t, first, "build/login.out", "ignored")
	writeIn(t, first, "notes-login.txt", "untracked")
	gitIn(t, first, "branch", "login-form")
	service := &SearchService{}
	found, err := service.InRepository(context.Background(), first, "Login")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(found.Files, []string{"src/login.go"}) {
		t.Fatalf("files, with ignored and untracked ones left out: %q", found.Files)
	}
	if len(found.Branches) != 1 || found.Branches[0].Name != "login-form" || found.Branches[0].CheckoutName != "login-form" {
		t.Fatalf("branches: %+v", found.Branches)
	}
	if len(found.Commits) != 1 || found.Commits[0].Subject != "Fix the login redirect" {
		t.Fatalf("commits: %+v", found.Commits)
	}

	if err := (&NotesService{}).Set(second, "branch", "side", "", "Ask about the login flow"); err != nil {
		t.Fatal(err)
	}
	elsewhere, err := service.InRepository(context.Background(), second, "login")
	if err != nil {
		t.Fatal(err)
	}
	if len(elsewhere.Files)+len(elsewhere.Branches)+len(elsewhere.Commits) != 0 || len(elsewhere.Notes) != 1 {
		t.Fatalf("the other repository: %+v", elsewhere)
	}

	cancelled, stop := context.WithCancel(context.Background())
	stop()
	if _, err := service.InRepository(cancelled, first, "login"); err == nil {
		t.Fatal("a cancelled search ran on")
	}
}

func TestTheNameARemoteBranchIsCheckedOutAs(t *testing.T) {
	remotes := []git.Remote{{Name: "origin"}, {Name: "origin/mirror"}}
	for name, want := range map[string]string{"origin/feat": "feat", "origin/mirror/feat": "feat", "upstream/x/y": "x/y"} {
		if got := localNameOf(name, remotes); got != want {
			t.Errorf("%s: %q, want %q", name, got, want)
		}
	}
}
