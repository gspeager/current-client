package git

import (
	"context"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func logSubjects(t *testing.T, dir, revs string) []string {
	t.Helper()
	out := gittest.Run(t, dir, "log", "--reverse", "--format=%s", revs)
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}

func previewSubjects(preview RebasePreview, skip ...ReplayOutcome) []string {
	var subjects []string
	for _, commit := range preview.Commits {
		skipped := false
		for _, outcome := range skip {
			skipped = skipped || commit.Outcome == outcome
		}
		if !skipped {
			subjects = append(subjects, commit.Subject)
		}
	}
	return subjects
}

func equalStrings(a, b []string) bool {
	return strings.Join(a, "\n") == strings.Join(b, "\n")
}

func TestPreviewRebaseMatchesTheRebase(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "branch", "-M", "main")
	gittest.CommitFile(t, dir, "base.txt", "base\n", "base")
	gittest.Run(t, dir, "switch", "-c", "topic")
	gittest.CommitFile(t, dir, "one.txt", "one\n", "one")
	gittest.CommitFile(t, dir, "two.txt", "two\n", "two")
	gittest.CommitFile(t, dir, "picked.txt", "picked\n", "already on main")
	picked := gittest.Run(t, dir, "rev-parse", "HEAD")
	gittest.Run(t, dir, "switch", "main")
	gittest.CommitFile(t, dir, "main.txt", "main\n", "main moved")
	gittest.Run(t, dir, "cherry-pick", picked)
	gittest.Run(t, dir, "switch", "topic")
	ctx := context.Background()

	preview, err := PreviewRebase(ctx, dir, "main")
	if err != nil {
		t.Fatal(err)
	}
	if preview.Current != "topic" || preview.Command != "git rebase main" || !preview.Predicted || preview.Dirty {
		t.Fatalf("preview = %+v", preview)
	}
	predicted := previewSubjects(preview, ReplayEmpty)
	if !equalStrings(predicted, []string{"one", "two"}) {
		t.Fatalf("predicted %v", predicted)
	}

	if err := RebaseOnto(ctx, dir, "main"); err != nil {
		t.Fatal(err)
	}
	if got := logSubjects(t, dir, "main..HEAD"); !equalStrings(got, predicted) {
		t.Fatalf("rebase made %v, predicted %v", got, predicted)
	}
}

func TestPreviewRebasePredictsAConflictAMergeWouldnt(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "branch", "-M", "main")
	gittest.CommitFile(t, dir, "f.txt", "X\n", "base")
	gittest.Run(t, dir, "switch", "-c", "topic")
	gittest.CommitFile(t, dir, "f.txt", "Y\n", "X to Y")
	gittest.CommitFile(t, dir, "f.txt", "X\n", "Y back to X")
	gittest.CommitFile(t, dir, "g.txt", "g\n", "add g")
	gittest.Run(t, dir, "switch", "main")
	gittest.CommitFile(t, dir, "f.txt", "Z\n", "X to Z")
	gittest.Run(t, dir, "switch", "topic")
	ctx := context.Background()

	if files, err := MergeConflicts(ctx, dir, "main", "topic"); err != nil || len(files) != 0 {
		t.Fatalf("a merge of the tips should be clean: %v, %v", files, err)
	}
	preview, err := PreviewRebase(ctx, dir, "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Commits) != 3 {
		t.Fatalf("commits = %+v", preview.Commits)
	}
	first, rest := preview.Commits[0], preview.Commits[1:]
	if first.Subject != "X to Y" || first.Outcome != ReplayConflict || len(first.Files) != 1 || first.Files[0] != "f.txt" {
		t.Fatalf("first = %+v", first)
	}
	for _, commit := range rest {
		if commit.Outcome != ReplayUnknown {
			t.Fatalf("after a conflict = %+v", commit)
		}
	}

	if err := RebaseOnto(ctx, dir, "main"); err != nil {
		t.Fatal(err)
	}
	if status := gittest.Run(t, dir, "status", "--porcelain"); !strings.Contains(status, "UU f.txt") {
		t.Fatalf("the real rebase should stop on the conflict: %q", status)
	}
	gittest.Run(t, dir, "rebase", "--abort")
}

func TestReplayCommitsFollowsTheGivenOrder(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "branch", "-M", "main")
	gittest.CommitFile(t, dir, "f.txt", "1\n", "base")
	base := gittest.Run(t, dir, "rev-parse", "HEAD")
	gittest.CommitFile(t, dir, "f.txt", "2\n", "first edit")
	first := gittest.Run(t, dir, "rev-parse", "HEAD")
	gittest.CommitFile(t, dir, "f.txt", "3\n", "second edit")
	second := gittest.Run(t, dir, "rev-parse", "HEAD")
	ctx := context.Background()

	inOrder, err := ReplayCommits(ctx, dir, base, []string{first, second})
	if err != nil || inOrder[0].Outcome != ReplayClean || inOrder[1].Outcome != ReplayClean {
		t.Fatalf("in order = %+v, %v", inOrder, err)
	}
	swapped, err := ReplayCommits(ctx, dir, base, []string{second, first})
	if err != nil || swapped[0].Outcome != ReplayConflict || swapped[1].Outcome != ReplayUnknown {
		t.Fatalf("swapped = %+v, %v", swapped, err)
	}
}

func TestReplayChangesNoRefOrFile(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "branch", "-M", "main")
	gittest.CommitFile(t, dir, "f.txt", "1\n", "base")
	gittest.Run(t, dir, "switch", "-c", "topic")
	gittest.CommitFile(t, dir, "g.txt", "g\n", "topic")
	before := gittest.Run(t, dir, "for-each-ref")
	head := gittest.Run(t, dir, "rev-parse", "HEAD")
	if _, err := PreviewRebase(context.Background(), dir, "main"); err != nil {
		t.Fatal(err)
	}
	if gittest.Run(t, dir, "for-each-ref") != before || gittest.Run(t, dir, "rev-parse", "HEAD") != head {
		t.Fatal("a preview moved a ref")
	}
	if status := gittest.Run(t, dir, "status", "--porcelain"); status != "" {
		t.Fatalf("a preview changed files: %q", status)
	}
}

func TestPreviewRebaseForcePushAndDirty(t *testing.T) {
	remote := t.TempDir()
	gittest.Run(t, remote, "init", "--bare", "-b", "main")
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "branch", "-M", "main")
	gittest.CommitFile(t, dir, "f.txt", "1\n", "base")
	gittest.Run(t, dir, "remote", "add", "origin", remote)
	gittest.Run(t, dir, "push", "-u", "origin", "main")
	gittest.Run(t, dir, "switch", "-c", "topic")
	gittest.CommitFile(t, dir, "g.txt", "g\n", "pushed")
	gittest.Run(t, dir, "push", "-u", "origin", "topic")
	gittest.Run(t, dir, "switch", "main")
	gittest.CommitFile(t, dir, "h.txt", "h\n", "main moved")
	gittest.Run(t, dir, "switch", "topic")
	ctx := context.Background()

	preview, err := PreviewRebase(ctx, dir, "main")
	if err != nil || !preview.ForcePush || preview.Upstream != "origin/topic" {
		t.Fatalf("pushed commits = %+v, %v", preview, err)
	}

	gittest.Run(t, dir, "switch", "-c", "unpushed", "main")
	gittest.CommitFile(t, dir, "i.txt", "i\n", "not pushed")
	preview, err = PreviewRebase(ctx, dir, "main~1")
	if err != nil || preview.ForcePush {
		t.Fatalf("unpushed = %+v, %v", preview, err)
	}

	gittest.Run(t, dir, "switch", "topic")
	gittest.WriteFile(t, dir, "f.txt", "changed\n")
	if preview, err = PreviewRebase(ctx, dir, "main"); err != nil || !preview.Dirty {
		t.Fatalf("tracked change = %+v, %v", preview, err)
	}
	gittest.Run(t, dir, "config", "rebase.autostash", "true")
	if preview, err = PreviewRebase(ctx, dir, "main"); err != nil || preview.Dirty {
		t.Fatalf("with autostash = %+v, %v", preview, err)
	}
	gittest.Run(t, dir, "config", "rebase.autostash", "false")
	gittest.Run(t, dir, "checkout", "--", "f.txt")
	gittest.WriteFile(t, dir, "untracked.txt", "new\n")
	if preview, err = PreviewRebase(ctx, dir, "main"); err != nil || preview.Dirty {
		t.Fatalf("untracked only = %+v, %v", preview, err)
	}
}

func TestPreviewRebaseWithNothingToDo(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "branch", "-M", "main")
	gittest.CommitFile(t, dir, "f.txt", "1\n", "base")
	gittest.Run(t, dir, "branch", "same")
	preview, err := PreviewRebase(context.Background(), dir, "same")
	if err != nil || len(preview.Commits) != 0 {
		t.Fatalf("preview = %+v, %v", preview, err)
	}
}

func TestPreviewRebaseWithABranchAlreadyOnTheTarget(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "branch", "-M", "main")
	gittest.CommitFile(t, dir, "f.txt", "1\n", "base")
	gittest.Run(t, dir, "switch", "-c", "topic")
	gittest.CommitFile(t, dir, "g.txt", "g\n", "ahead")
	preview, err := PreviewRebase(context.Background(), dir, "main")
	if err != nil || len(preview.Commits) != 0 {
		t.Fatalf("preview = %+v, %v", preview, err)
	}
}

func TestPreviewRebaseWithUnrelatedHistoryStillOffersIt(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "branch", "-M", "main")
	gittest.CommitFile(t, dir, "f.txt", "1\n", "base")
	gittest.Run(t, dir, "switch", "--orphan", "other")
	gittest.CommitFile(t, dir, "g.txt", "g\n", "unrelated root")
	preview, err := PreviewRebase(context.Background(), dir, "main")
	if err != nil || preview.Predicted || len(preview.Commits) != 1 {
		t.Fatalf("preview = %+v, %v", preview, err)
	}
}

func TestPreviewRebaseCountsMergesItWouldFlatten(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "branch", "-M", "main")
	gittest.CommitFile(t, dir, "f.txt", "1\n", "base")
	gittest.Run(t, dir, "switch", "-c", "side")
	gittest.CommitFile(t, dir, "s.txt", "s\n", "side")
	gittest.Run(t, dir, "switch", "main")
	gittest.Run(t, dir, "switch", "-c", "topic")
	gittest.CommitFile(t, dir, "t.txt", "t\n", "topic")
	gittest.Run(t, dir, "merge", "--no-ff", "-m", "merge side", "side")
	preview, err := PreviewRebase(context.Background(), dir, "main")
	if err != nil || preview.DroppedMerges != 1 || len(preview.Commits) != 2 {
		t.Fatalf("preview = %+v, %v", preview, err)
	}
}
