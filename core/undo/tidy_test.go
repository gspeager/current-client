package undo

import (
	"context"
	"testing"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestUndoTakesBackAWholeTidyUp(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "branch", "-M", "main")
	gittest.CommitFile(t, dir, "base.txt", "base\n", "base")
	gittest.Run(t, dir, "switch", "-c", "topic")
	gittest.CommitFile(t, dir, "a.txt", "a\n", "add a")
	gittest.CommitFile(t, dir, "b.txt", "b\n", "add b")
	before := gittest.Run(t, dir, "rev-parse", "HEAD")
	ctx := context.Background()
	plan, err := git.PlanTidy(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	steps := []git.TidyStep{
		{SHA: plan.Commits[1].SHA, Action: git.TidyReword, Message: "reworded"},
		{SHA: plan.Commits[0].SHA, Action: git.TidyKeep},
	}
	if err := git.RunTidy(ctx, dir, plan.Head, steps); err != nil {
		t.Fatal(err)
	}
	if gittest.Run(t, dir, "rev-parse", "HEAD") == before {
		t.Fatal("nothing moved")
	}
	undoPlan, err := Preview(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if undoPlan.Operation != Rebase {
		t.Fatalf("plan = %+v", undoPlan)
	}
	if err := Apply(ctx, dir, undoPlan); err != nil {
		t.Fatal(err)
	}
	if got := gittest.Run(t, dir, "rev-parse", "HEAD"); got != before {
		t.Fatalf("HEAD = %s, want %s", got, before)
	}
}
