package git

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

// tidyRepo has main with base, and topic with three commits on separate files.
func tidyRepo(t *testing.T) string {
	t.Helper()
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "branch", "-M", "main")
	gittest.CommitFile(t, dir, "base.txt", "base\n", "base")
	gittest.Run(t, dir, "switch", "-c", "topic")
	gittest.CommitFile(t, dir, "a.txt", "a\n", "add a\n\nWhy a.")
	gittest.CommitFile(t, dir, "b.txt", "b\n", "add b")
	gittest.CommitFile(t, dir, "c.txt", "c\n", "add c")
	return dir
}

func keepAll(plan TidyPlan) []TidyStep {
	steps := make([]TidyStep, len(plan.Commits))
	for i, commit := range plan.Commits {
		steps[i] = TidyStep{SHA: commit.SHA, Action: TidyKeep}
	}
	return steps
}

func plannedSteps(t *testing.T, dir string) (TidyPlan, []TidyStep) {
	t.Helper()
	plan, err := PlanTidy(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	return plan, keepAll(plan)
}

func runTidy(t *testing.T, dir string, plan TidyPlan, steps []TidyStep) {
	t.Helper()
	if err := RunTidy(context.Background(), dir, plan.Head, steps); err != nil {
		t.Fatal(err)
	}
}

func TestPlanTidyOffersTheBranchsOwnCommits(t *testing.T) {
	dir := tidyRepo(t)
	plan, _ := plannedSteps(t, dir)
	if plan.BaseName != "main" || len(plan.Commits) != 3 || plan.Commits[0].Subject != "add a" {
		t.Fatalf("plan = %+v", plan)
	}
	if !strings.Contains(plan.Commits[0].Message, "Why a.") {
		t.Fatalf("message = %q", plan.Commits[0].Message)
	}
}

func TestTidyReorders(t *testing.T) {
	dir := tidyRepo(t)
	plan, steps := plannedSteps(t, dir)
	steps[0], steps[2] = steps[2], steps[0]
	runTidy(t, dir, plan, steps)
	if got := logSubjects(t, dir, "main..HEAD"); !equalStrings(got, []string{"add c", "add b", "add a"}) {
		t.Fatalf("log = %v", got)
	}
}

func TestTidyDrops(t *testing.T) {
	dir := tidyRepo(t)
	plan, steps := plannedSteps(t, dir)
	steps[1].Action = TidyDrop
	runTidy(t, dir, plan, steps)
	if got := logSubjects(t, dir, "main..HEAD"); !equalStrings(got, []string{"add a", "add c"}) {
		t.Fatalf("log = %v", got)
	}
	if _, err := os.Stat(dir + "/b.txt"); !os.IsNotExist(err) {
		t.Fatalf("b.txt should be gone: %v", err)
	}
}

func TestTidyRewords(t *testing.T) {
	dir := tidyRepo(t)
	plan, steps := plannedSteps(t, dir)
	steps[1].Action, steps[1].Message = TidyReword, "better subject\n\nWith a body."
	runTidy(t, dir, plan, steps)
	if got := logSubjects(t, dir, "main..HEAD"); !equalStrings(got, []string{"add a", "better subject", "add c"}) {
		t.Fatalf("log = %v", got)
	}
	if body := gittest.Run(t, dir, "log", "-1", "--format=%b", "HEAD~1"); body != "With a body." {
		t.Fatalf("body = %q", body)
	}
}

func TestTidyFixupKeepsTheFirstMessageAndSquashJoinsThem(t *testing.T) {
	dir := tidyRepo(t)
	plan, steps := plannedSteps(t, dir)
	steps[1].Action = TidyFixup
	runTidy(t, dir, plan, steps)
	if got := logSubjects(t, dir, "main..HEAD"); !equalStrings(got, []string{"add a", "add c"}) {
		t.Fatalf("after fixup = %v", got)
	}
	if body := gittest.Run(t, dir, "log", "-1", "--format=%B", "HEAD~1"); strings.Contains(body, "add b") {
		t.Fatalf("a fixup kept its message: %q", body)
	}
	if files := gittest.Run(t, dir, "show", "--name-only", "--format=", "HEAD~1"); !strings.Contains(files, "b.txt") {
		t.Fatalf("files = %q", files)
	}

	plan, steps = plannedSteps(t, dir)
	steps[1].Action = TidySquash
	runTidy(t, dir, plan, steps)
	message := gittest.Run(t, dir, "log", "-1", "--format=%B", "HEAD")
	if !strings.Contains(message, "add a") || !strings.Contains(message, "Why a.") || !strings.Contains(message, "add c") {
		t.Fatalf("squashed message = %q", message)
	}
}

func TestTidyKeepsAuthorship(t *testing.T) {
	dir := tidyRepo(t)
	gittest.Run(t, dir, "-c", "user.name=Ada", "-c", "user.email=ada@example.com", "commit", "--allow-empty", "--amend", "--reset-author", "--no-edit")
	plan, steps := plannedSteps(t, dir)
	steps[2].Action, steps[2].Message = TidyReword, "reworded"
	runTidy(t, dir, plan, steps)
	if author := gittest.Run(t, dir, "log", "-1", "--format=%an"); author != "Ada" {
		t.Fatalf("author = %q", author)
	}
}

// The review's finding: Continue answers the editor with GIT_EDITOR=true, so a
// reword waiting behind a conflict must not depend on an editor.
func TestARewordAfterAConflictSurvivesContinue(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "branch", "-M", "main")
	gittest.CommitFile(t, dir, "f.txt", "1\n", "base")
	gittest.Run(t, dir, "switch", "-c", "topic")
	gittest.CommitFile(t, dir, "f.txt", "2\n", "first edit")
	gittest.CommitFile(t, dir, "f.txt", "3\n", "second edit")
	gittest.CommitFile(t, dir, "g.txt", "g\n", "old message")
	gittest.Run(t, dir, "commit", "--amend", "-m", "old message")
	plan, steps := plannedSteps(t, dir)
	ctx := context.Background()

	steps[0], steps[1] = steps[1], steps[0]
	steps[2].Action, steps[2].Message = TidyReword, "new message"
	runTidy(t, dir, plan, steps)
	state, err := DetectConflictState(ctx, dir)
	if err != nil || state.Operation != ConflictRebase {
		t.Fatalf("expected a stop: %+v, %v", state, err)
	}

	// Resolve each stop as the banner would, however many there are.
	for range 5 {
		state, err = DetectConflictState(ctx, dir)
		if err != nil || state.Operation == "" {
			break
		}
		gittest.WriteFile(t, dir, "f.txt", "resolved\n")
		gittest.Run(t, dir, "add", "f.txt")
		if err := ContinueConflictOperation(ctx, dir, ConflictRebase); err != nil {
			t.Fatal(err)
		}
	}
	if got := logSubjects(t, dir, "main..HEAD"); got[len(got)-1] != "new message" {
		t.Fatalf("log = %v", got)
	}
}

func TestPreviewTidyMatchesTheResult(t *testing.T) {
	dir := tidyRepo(t)
	plan, steps := plannedSteps(t, dir)
	steps[0].Action = TidyDrop
	steps[2].Action = TidyFixup
	preview, err := PreviewTidy(context.Background(), dir, steps)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Result) != 1 || preview.Result[0].Subject != "add b" || preview.Result[0].Folded != 1 || !preview.Predicted {
		t.Fatalf("preview = %+v", preview)
	}
	runTidy(t, dir, plan, steps)
	if got := logSubjects(t, dir, "main..HEAD"); !equalStrings(got, []string{"add b"}) {
		t.Fatalf("log = %v", got)
	}
}

func TestPreviewTidyPredictsAConflictFromAReorder(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "branch", "-M", "main")
	gittest.CommitFile(t, dir, "f.txt", "1\n", "base")
	gittest.Run(t, dir, "switch", "-c", "topic")
	gittest.CommitFile(t, dir, "f.txt", "2\n", "first")
	gittest.CommitFile(t, dir, "f.txt", "3\n", "second")
	_, steps := plannedSteps(t, dir)
	steps[0], steps[1] = steps[1], steps[0]
	preview, err := PreviewTidy(context.Background(), dir, steps)
	if err != nil || preview.Replay[0].Outcome != ReplayConflict {
		t.Fatalf("preview = %+v, %v", preview, err)
	}
}

func TestPreviewTidyForcePush(t *testing.T) {
	remote := t.TempDir()
	gittest.Run(t, remote, "init", "--bare", "-b", "main")
	dir := tidyRepo(t)
	gittest.Run(t, dir, "remote", "add", "origin", remote)
	gittest.Run(t, dir, "push", "-u", "origin", "main")
	gittest.Run(t, dir, "push", "-u", "origin", "topic~1:refs/heads/topic")
	gittest.Run(t, dir, "fetch", "origin")
	gittest.Run(t, dir, "branch", "-u", "origin/topic")
	ctx := context.Background()

	plan, err := PlanTidy(ctx, dir)
	if err != nil || len(plan.Commits) != 1 {
		// Ahead of its upstream by one: only the unpushed commit is offered.
		t.Fatalf("plan = %+v, %v", plan, err)
	}
	steps := keepAll(plan)
	steps[0].Action, steps[0].Message = TidyReword, "only local"
	preview, err := PreviewTidy(ctx, dir, steps)
	if err != nil || preview.ForcePush {
		t.Fatalf("rewriting an unpushed commit = %+v, %v", preview, err)
	}

	gittest.Run(t, dir, "push", "origin", "topic")
	gittest.Run(t, dir, "fetch", "origin")
	plan, err = PlanTidy(ctx, dir)
	if err != nil || plan.BaseName != "main" {
		t.Fatalf("plan = %+v, %v", plan, err)
	}
	steps = keepAll(plan)
	steps[0].Action, steps[0].Message = TidyReword, "pushed"
	preview, err = PreviewTidy(ctx, dir, steps)
	if err != nil || !preview.ForcePush {
		t.Fatalf("rewriting a pushed commit = %+v, %v", preview, err)
	}
}

func TestRunTidyRefuses(t *testing.T) {
	dir := tidyRepo(t)
	plan, steps := plannedSteps(t, dir)
	ctx := context.Background()
	if err := RunTidy(ctx, dir, "stale", steps); err == nil || !strings.Contains(err.Error(), "moved") {
		t.Fatalf("a moved branch: %v", err)
	}
	bad := keepAll(plan)
	bad[0].Action = TidyFixup
	if err := RunTidy(ctx, dir, plan.Head, bad); err == nil {
		t.Fatal("a first commit with nothing to fold into should be refused")
	}
	all := keepAll(plan)
	for i := range all {
		all[i].Action = TidyDrop
	}
	if err := RunTidy(ctx, dir, plan.Head, all); err == nil {
		t.Fatal("dropping everything should be refused")
	}
	if err := RunTidy(ctx, dir, plan.Head, steps[:2]); err == nil {
		t.Fatal("missing a commit should be refused")
	}
	blank := keepAll(plan)
	blank[0].Action = TidyReword
	if err := RunTidy(ctx, dir, plan.Head, blank); err == nil {
		t.Fatal("an empty message should be refused")
	}
	gittest.WriteFile(t, dir, "a.txt", "dirty\n")
	if err := RunTidy(ctx, dir, plan.Head, steps); err == nil || !strings.Contains(err.Error(), "stash") {
		t.Fatalf("changes in the way: %v", err)
	}
}

func TestPlanTidyFlagsAMerge(t *testing.T) {
	dir := tidyRepo(t)
	gittest.Run(t, dir, "switch", "-c", "side", "main")
	gittest.CommitFile(t, dir, "s.txt", "s\n", "side")
	gittest.Run(t, dir, "switch", "topic")
	gittest.Run(t, dir, "merge", "--no-ff", "-m", "merge side", "side")
	plan, _ := plannedSteps(t, dir)
	if !plan.HasMerges {
		t.Fatalf("plan = %+v", plan)
	}
}

func TestFixupIntoAndAutosquashArrangement(t *testing.T) {
	dir := tidyRepo(t)
	plan, _ := plannedSteps(t, dir)
	target := plan.Commits[0].SHA
	gittest.WriteFile(t, dir, "a.txt", "a fixed\n")
	gittest.Run(t, dir, "add", "a.txt")
	if err := FixupInto(context.Background(), dir, target); err != nil {
		t.Fatal(err)
	}

	plan, _ = plannedSteps(t, dir)
	subjects := make([]string, len(plan.Commits))
	for i, commit := range plan.Commits {
		subjects[i] = commit.Subject
	}
	if !equalStrings(subjects, []string{"add a", "fixup! add a", "add b", "add c"}) || plan.Commits[1].Suggested != TidyFixup {
		t.Fatalf("arranged = %v, %+v", subjects, plan.Commits[1])
	}
	steps := keepAll(plan)
	steps[1].Action = TidyFixup
	runTidy(t, dir, plan, steps)
	if got := logSubjects(t, dir, "main..HEAD"); !equalStrings(got, []string{"add a", "add b", "add c"}) {
		t.Fatalf("log = %v", got)
	}
	if got := gittest.Run(t, dir, "show", "HEAD~2:a.txt"); got != "a fixed" {
		t.Fatalf("a.txt = %q", got)
	}
}

func TestFixupIntoWithNothingStagedFails(t *testing.T) {
	dir := tidyRepo(t)
	if err := FixupInto(context.Background(), dir, "HEAD"); err == nil {
		t.Fatal("nothing staged should fail")
	}
}

func TestTidyReachesARootCommitOnAShortBranch(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "branch", "-M", "main")
	gittest.CommitFile(t, dir, "a.txt", "a\n", "first")
	gittest.CommitFile(t, dir, "b.txt", "b\n", "second")
	plan, steps := plannedSteps(t, dir)
	if plan.Base != "" || len(plan.Commits) != 2 {
		t.Fatalf("plan = %+v", plan)
	}
	steps[0].Action, steps[0].Message = TidyReword, "first, better"
	runTidy(t, dir, plan, steps)
	if got := logSubjects(t, dir, "HEAD"); !equalStrings(got, []string{"first, better", "second"}) {
		t.Fatalf("log = %v", got)
	}
}
