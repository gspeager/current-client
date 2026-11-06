package app

import (
	"os/exec"
	"strings"
	"testing"
)

func TestStackServiceFollowsAStackThroughARestack(t *testing.T) {
	s := &StackService{}
	repo := recoverRepo(t)
	commit := func(name string) {
		write(t, repo, name+".txt", name+"\n")
		gitIn(t, repo, "add", ".")
		gitIn(t, repo, "commit", "-q", "-m", name)
	}
	rev := func(name string) string {
		out, err := exec.Command("git", "-C", repo, "rev-parse", name).Output()
		if err != nil {
			t.Fatal(err)
		}
		return string(out)
	}

	if err := s.NewLayer(repo, "bottom", "main"); err != nil {
		t.Fatal(err)
	}
	commit("bottom")
	if err := s.NewLayer(repo, "top", "bottom"); err != nil {
		t.Fatal(err)
	}
	commit("top")

	if err := s.AddLayer(repo, "bottom", "top"); err == nil || !strings.Contains(err.Error(), "above it") {
		t.Fatalf("a cycle: %v", err)
	}

	list, err := s.List(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Stacks) != 1 || list.Stacks[0].Base != "main" || list.Stacks[0].NeedsRestack {
		t.Fatalf("got %+v", list)
	}
	if layers := list.Stacks[0].Layers; layers[0].Branch != "bottom" || layers[1].Branch != "top" || layers[1].Depth != 1 {
		t.Fatalf("got %+v", layers)
	}

	gitIn(t, repo, "switch", "bottom")
	gitIn(t, repo, "commit", "--amend", "-m", "bottom amended")
	gitIn(t, repo, "switch", "top")
	before := rev("top")
	if list, _ = s.List(repo); !list.Stacks[0].NeedsRestack || !list.Stacks[0].Layers[1].Behind {
		t.Fatalf("after an amend: %+v", list)
	}

	preview, err := s.Preview(repo, "bottom")
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Layers) != 2 || !preview.Layers[1].Moves || preview.Layers[1].Commits[0].Subject != "top" {
		t.Fatalf("preview: %+v", preview)
	}

	result, err := s.Restack(repo, "bottom")
	if err != nil {
		t.Fatal(err)
	}
	if result.Moved != 1 {
		t.Fatalf("got %+v", result)
	}
	if list, _ = s.List(repo); list.Stacks[0].NeedsRestack || !list.CanUndo {
		t.Fatalf("after restacking: %+v", list)
	}

	if err := s.UndoRestack(repo); err != nil {
		t.Fatal(err)
	}
	if got := rev("top"); got != before {
		t.Fatal("undo didn't put top back")
	}
	if list, _ = s.List(repo); list.CanUndo || !list.Stacks[0].NeedsRestack {
		t.Fatalf("after undo: %+v", list)
	}

	if err := s.RemoveLayer(repo, "bottom"); err != nil {
		t.Fatal(err)
	}
	if err := (&BranchService{}).RenameBranch(repo, "top", "renamed"); err != nil {
		t.Fatal(err)
	}
	list, _ = s.List(repo)
	if layer := list.Stacks[0].Layers[0]; layer.Branch != "renamed" || layer.Parent != "main" {
		t.Fatalf("after remove and rename: %+v", layer)
	}
}

func TestUndoRestackKeepsItsOfferWhenItCanOnlyWaitButNotWhenABranchMoved(t *testing.T) {
	s := &StackService{}
	repo := recoverRepo(t)
	if err := s.NewLayer(repo, "bottom", "main"); err != nil {
		t.Fatal(err)
	}
	write(t, repo, "b.txt", "b\n")
	gitIn(t, repo, "add", ".")
	gitIn(t, repo, "commit", "-q", "-m", "bottom")
	gitIn(t, repo, "switch", "-q", "main")
	write(t, repo, "m.txt", "m\n")
	gitIn(t, repo, "add", ".")
	gitIn(t, repo, "commit", "-q", "-m", "main moved")
	gitIn(t, repo, "switch", "-q", "bottom")
	if _, err := s.Restack(repo, "bottom"); err != nil {
		t.Fatal(err)
	}

	// A conflict in progress only has to finish.
	gitIn(t, repo, "switch", "-q", "main")
	write(t, repo, "m.txt", "main's\n")
	gitIn(t, repo, "commit", "-qam", "ours")
	gitIn(t, repo, "switch", "-q", "-c", "side", "HEAD~1")
	write(t, repo, "m.txt", "side's\n")
	gitIn(t, repo, "commit", "-qam", "theirs")
	if err := exec.Command("git", "-C", repo, "merge", "main").Run(); err == nil {
		t.Fatal("wanted a conflict")
	}
	if err := s.UndoRestack(repo); err == nil {
		t.Fatal("undid during a conflict")
	}
	if list, _ := s.List(repo); !list.CanUndo {
		t.Fatal("waiting for a conflict ended the offer")
	}
	gitIn(t, repo, "merge", "--abort")

	gitIn(t, repo, "switch", "-q", "bottom")
	write(t, repo, "later.txt", "later\n")
	gitIn(t, repo, "add", ".")
	gitIn(t, repo, "commit", "-q", "-m", "later")
	if err := s.UndoRestack(repo); err == nil {
		t.Fatal("undid over later work")
	}
	if list, _ := s.List(repo); list.CanUndo {
		t.Fatal("a branch that moved on should end the offer")
	}
}
