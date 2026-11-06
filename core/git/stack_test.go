package git

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

// threeLayers makes main <- bottom <- middle <- top, with one commit each.
func threeLayers(t *testing.T) (string, []StackLink) {
	t.Helper()
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "branch", "-M", "main")
	gittest.CommitFile(t, dir, "base.txt", "base\n", "base")
	var links []StackLink
	parent := "main"
	for _, name := range []string{"bottom", "middle", "top"} {
		tip := gittest.Run(t, dir, "rev-parse", "HEAD")
		gittest.Run(t, dir, "switch", "-c", name)
		gittest.CommitFile(t, dir, name+".txt", name+"\n", name)
		links = append(links, StackLink{Branch: name, Parent: parent, ParentTip: tip})
		parent = name
	}
	return dir, links
}

func assertLine(t *testing.T, dir, branch string, want ...string) {
	t.Helper()
	got := logSubjects(t, dir, branch)
	if !equalStrings(got, want) {
		t.Fatalf("%s: got %v, want %v", branch, got, want)
	}
}

func TestRestackAfterTheBottomIsAmended(t *testing.T) {
	ctx := context.Background()
	dir, links := threeLayers(t)
	gittest.Run(t, dir, "switch", "bottom")
	gittest.Run(t, dir, "commit", "--amend", "-m", "bottom amended")
	gittest.Run(t, dir, "switch", "top")

	result, err := Restack(ctx, dir, links)
	if err != nil {
		t.Fatal(err)
	}
	if result.Stopped != "" || len(result.Moves) != 2 {
		t.Fatalf("got %+v", result)
	}
	assertLine(t, dir, "top", "base", "bottom amended", "middle", "top")
	assertLine(t, dir, "middle", "base", "bottom amended", "middle")
	if head := gittest.Run(t, dir, "symbolic-ref", "--short", "HEAD"); head != "top" {
		t.Fatalf("left on %s", head)
	}

	again, err := Restack(ctx, dir, result.Links)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Moves) != 0 {
		t.Fatalf("a second restack moved %+v", again.Moves)
	}
}

func TestRestackAfterTheBottomIsMerged(t *testing.T) {
	merges := map[string][]string{
		"merge commit": {"merge", "--no-ff", "-m", "merged", "bottom"},
		"squash":       {"merge", "--squash", "bottom"},
	}
	for name, merge := range merges {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			dir, links := threeLayers(t)
			gittest.Run(t, dir, "switch", "main")
			gittest.CommitFile(t, dir, "main.txt", "main\n", "main moved")
			gittest.Run(t, dir, merge...)
			if name == "squash" {
				gittest.Run(t, dir, "commit", "-m", "squashed bottom")
			}
			gittest.Run(t, dir, "switch", "top")

			layers, err := StackLayers(ctx, dir, links)
			if err != nil {
				t.Fatal(err)
			}
			if !layers[0].Merged || !layers[1].Behind || !layers[2].Behind {
				t.Fatalf("got %+v", layers)
			}
			result, err := Restack(ctx, dir, links)
			if err != nil {
				t.Fatal(err)
			}
			if !equalStrings(result.Left, []string{"bottom"}) {
				t.Fatalf("left: %v", result.Left)
			}
			if result.Links[0].Branch != "middle" || result.Links[0].Parent != "main" {
				t.Fatalf("links: %+v", result.Links)
			}
			own := logSubjects(t, dir, "main..top")
			if !equalStrings(own, []string{"middle", "top"}) {
				t.Fatalf("own commits: %v", own)
			}
		})
	}
}

func TestANewEmptyLayerIsntTakenAsMerged(t *testing.T) {
	dir, links := threeLayers(t)
	tip := gittest.Run(t, dir, "rev-parse", "top")
	gittest.Run(t, dir, "branch", "empty", "top")
	layers, err := StackLayers(context.Background(), dir, append(links, StackLink{Branch: "empty", Parent: "top", ParentTip: tip}))
	if err != nil {
		t.Fatal(err)
	}
	if last := layers[3]; last.Merged || last.Behind {
		t.Fatalf("got %+v", last)
	}
}

func TestRestackStopsOnAConflictAndCarriesOn(t *testing.T) {
	ctx := context.Background()
	dir, links := threeLayers(t)
	gittest.Run(t, dir, "switch", "bottom")
	gittest.CommitFile(t, dir, "middle.txt", "bottom's version\n", "bottom touches middle")
	gittest.Run(t, dir, "switch", "top")

	result, err := Restack(ctx, dir, links)
	if err != nil {
		t.Fatal(err)
	}
	if result.Stopped != "middle" {
		t.Fatalf("stopped on %q", result.Stopped)
	}
	if _, err := Restack(ctx, dir, result.Links); err == nil || !strings.Contains(err.Error(), "Finish or abort") {
		t.Fatalf("during a conflict: %v", err)
	}

	gittest.WriteFile(t, dir, "middle.txt", "resolved\n")
	gittest.Run(t, dir, "add", "middle.txt")
	gittest.Run(t, dir, "-c", "core.editor=true", "rebase", "--continue")

	links, _ = RefreshStackLinks(ctx, dir, result.Links)
	next, err := Restack(ctx, dir, links)
	if err != nil {
		t.Fatal(err)
	}
	if next.Stopped != "" || len(next.Moves) != 1 || next.Moves[0].Branch != "top" {
		t.Fatalf("got %+v", next)
	}
	assertLine(t, dir, "top", "base", "bottom", "bottom touches middle", "middle", "top")
}

func TestRestackAfterAConflictIsAborted(t *testing.T) {
	ctx := context.Background()
	dir, links := threeLayers(t)
	gittest.Run(t, dir, "switch", "bottom")
	gittest.CommitFile(t, dir, "middle.txt", "bottom's version\n", "bottom touches middle")
	gittest.Run(t, dir, "switch", "top")
	result, err := Restack(ctx, dir, links)
	if err != nil {
		t.Fatal(err)
	}
	gittest.Run(t, dir, "rebase", "--abort")
	gittest.Run(t, dir, "switch", "bottom")
	gittest.Run(t, dir, "reset", "--hard", "HEAD~1")
	gittest.CommitFile(t, dir, "other.txt", "other\n", "bottom again")
	gittest.Run(t, dir, "switch", "top")

	if _, err := Restack(ctx, dir, result.Links); err != nil {
		t.Fatal(err)
	}
	assertLine(t, dir, "top", "base", "bottom", "bottom again", "middle", "top")
}

func TestRestackRefusesALayerInAnotherWorktree(t *testing.T) {
	dir, links := threeLayers(t)
	gittest.Run(t, dir, "switch", "bottom")
	gittest.Run(t, dir, "commit", "--amend", "-m", "bottom amended")
	before := gittest.Run(t, dir, "rev-parse", "top")
	gittest.Run(t, dir, "worktree", "add", filepath.Join(t.TempDir(), "other"), "top")

	_, err := Restack(context.Background(), dir, links)
	if err == nil || !strings.Contains(err.Error(), "top is checked out in the worktree") {
		t.Fatalf("got %v", err)
	}
	if after := gittest.Run(t, dir, "rev-parse", "top"); after != before {
		t.Fatal("top moved")
	}
	preview, err := PreviewRestack(context.Background(), dir, links)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(preview.Blocked, "top is checked out") {
		t.Fatalf("preview: %+v", preview)
	}
}

func TestRestackRefusesAChangedWorkingTree(t *testing.T) {
	dir, links := threeLayers(t)
	gittest.Run(t, dir, "switch", "bottom")
	gittest.Run(t, dir, "commit", "--amend", "-m", "bottom amended")
	gittest.WriteFile(t, dir, "bottom.txt", "changed\n")
	_, err := Restack(context.Background(), dir, links)
	if err == nil || !strings.Contains(err.Error(), "Commit or stash") {
		t.Fatalf("got %v", err)
	}
}

func TestPreviewRestackBuildsOnThePredictedLayerBelow(t *testing.T) {
	ctx := context.Background()
	dir, links := threeLayers(t)
	gittest.Run(t, dir, "switch", "top")
	gittest.CommitFile(t, dir, "shared.txt", "top's\n", "top touches shared")
	gittest.Run(t, dir, "switch", "main")
	gittest.CommitFile(t, dir, "main.txt", "main\n", "main moved")
	gittest.Run(t, dir, "switch", "top")

	preview, err := PreviewRestack(ctx, dir, links)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Predicted || len(preview.Layers) != 3 || !preview.Layers[2].Moves {
		t.Fatalf("got %+v", preview)
	}
	if got := previewSubjects(RebasePreview{Commits: preview.Layers[2].Commits}); !equalStrings(got, []string{"top", "top touches shared"}) {
		t.Fatalf("top's commits: %v", got)
	}
	for _, layer := range preview.Layers {
		for _, commit := range layer.Commits {
			if commit.Outcome != ReplayClean {
				t.Fatalf("%s: %+v", layer.Branch, commit)
			}
		}
	}

	gittest.Run(t, dir, "switch", "bottom")
	gittest.CommitFile(t, dir, "middle.txt", "bottom's version\n", "bottom touches middle")
	gittest.Run(t, dir, "switch", "top")
	preview, err = PreviewRestack(ctx, dir, links)
	if err != nil {
		t.Fatal(err)
	}
	if got := preview.Layers[1].Commits[0].Outcome; got != ReplayConflict {
		t.Fatalf("middle: %s", got)
	}
	if got := preview.Layers[2].Commits[0].Outcome; got != ReplayUnknown {
		t.Fatalf("above the conflict: %s", got)
	}
	if head := gittest.Run(t, dir, "rev-parse", "top"); !strings.HasPrefix(head, preview.Layers[2].Commits[1].SHA[:7]) {
		t.Fatal("the preview moved top")
	}
}

func TestUndoRestackPutsEveryBranchBack(t *testing.T) {
	ctx := context.Background()
	dir, links := threeLayers(t)
	gittest.Run(t, dir, "switch", "bottom")
	gittest.Run(t, dir, "commit", "--amend", "-m", "bottom amended")
	gittest.Run(t, dir, "switch", "top")
	middle, top := gittest.Run(t, dir, "rev-parse", "middle"), gittest.Run(t, dir, "rev-parse", "top")

	result, err := Restack(ctx, dir, links)
	if err != nil {
		t.Fatal(err)
	}
	if err := UndoRestack(ctx, dir, result.Moves); err != nil {
		t.Fatal(err)
	}
	if got := gittest.Run(t, dir, "rev-parse", "middle"); got != middle {
		t.Fatal("middle not back")
	}
	if got := gittest.Run(t, dir, "rev-parse", "HEAD"); got != top {
		t.Fatal("top not back")
	}
}

func TestUndoRestackRefusesABranchThatMovedOn(t *testing.T) {
	ctx := context.Background()
	dir, links := threeLayers(t)
	gittest.Run(t, dir, "switch", "bottom")
	gittest.Run(t, dir, "commit", "--amend", "-m", "bottom amended")
	gittest.Run(t, dir, "switch", "top")
	result, err := Restack(ctx, dir, links)
	if err != nil {
		t.Fatal(err)
	}
	gittest.CommitFile(t, dir, "later.txt", "later\n", "later work")
	middle := gittest.Run(t, dir, "rev-parse", "middle")

	err = UndoRestack(ctx, dir, result.Moves)
	if err == nil || !strings.Contains(err.Error(), "top has changed") {
		t.Fatalf("got %v", err)
	}
	if got := gittest.Run(t, dir, "rev-parse", "middle"); got != middle {
		t.Fatal("a refused undo moved a branch")
	}
}

func TestPushStackIsAllOrNoneAndUsesTheLease(t *testing.T) {
	ctx := context.Background()
	dir, links := threeLayers(t)
	remote := t.TempDir()
	gittest.Run(t, remote, "init", "--bare", "-q")
	gittest.Run(t, dir, "remote", "add", "origin", remote)
	branches := []string{"bottom", "middle", "top"}
	if err := PushStack(ctx, dir, "origin", branches); err != nil {
		t.Fatal(err)
	}
	for _, branch := range branches {
		if gittest.Run(t, remote, "rev-parse", branch) != gittest.Run(t, dir, "rev-parse", branch) {
			t.Fatalf("%s wasn't pushed", branch)
		}
	}
	if up := gittest.Run(t, dir, "rev-parse", "--abbrev-ref", "top@{upstream}"); up != "origin/top" {
		t.Fatalf("upstream: %s", up)
	}

	gittest.Run(t, dir, "switch", "bottom")
	gittest.Run(t, dir, "commit", "--amend", "-m", "bottom amended")
	gittest.Run(t, dir, "switch", "top")
	if _, err := Restack(ctx, dir, links); err != nil {
		t.Fatal(err)
	}

	// Someone else pushes to middle, unseen by this clone's last fetch.
	other := t.TempDir()
	gittest.Run(t, other, "clone", "-q", remote, ".")
	gittest.Run(t, other, "switch", "middle")
	gittest.Run(t, other, "config", "user.email", "o@example.com")
	gittest.Run(t, other, "config", "user.name", "o")
	gittest.CommitFile(t, other, "theirs.txt", "theirs\n", "their work")
	theirs := gittest.Run(t, other, "rev-parse", "HEAD")
	gittest.Run(t, other, "push", "-q", "origin", "middle")
	bottomOnRemote := gittest.Run(t, remote, "rev-parse", "bottom")

	if err := PushStack(ctx, dir, "origin", branches); err == nil {
		t.Fatal("overwrote work the last fetch didn't see")
	}
	if gittest.Run(t, remote, "rev-parse", "middle") != theirs {
		t.Fatal("their work was overwritten")
	}
	if gittest.Run(t, remote, "rev-parse", "bottom") != bottomOnRemote {
		t.Fatal("bottom was pushed though middle was rejected")
	}

	gittest.Run(t, dir, "fetch", "-q", "origin")
	if err := PushStack(ctx, dir, "origin", branches); err != nil {
		t.Fatal(err)
	}
	if gittest.Run(t, remote, "rev-parse", "top") != gittest.Run(t, dir, "rev-parse", "top") {
		t.Fatal("top wasn't pushed")
	}
}

