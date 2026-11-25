package app

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/internal/config"
	"github.com/gspeager/current-client/internal/policy"
	"github.com/gspeager/current-client/internal/tasks"
)

func TestARunNeedsApprovalAndAgainAfterAChange(t *testing.T) {
	useTempSettings(t)
	repo := t.TempDir()
	marker := filepath.Join(repo, "ran")
	makefile := filepath.Join(repo, "Makefile")
	if err := os.WriteFile(makefile, []byte("go:\n\ttouch "+marker+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var s TaskService

	list, err := s.List(repo)
	if err != nil || len(list.Tasks) != 1 || list.Tasks[0].Approved {
		t.Fatalf("first list = %+v, %v", list, err)
	}
	if err := s.Run(repo, "make:go", "r1"); !errors.Is(err, errTaskNotApproved) {
		t.Fatalf("an unapproved run: %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("an unapproved command ran")
	}

	listed, _ := s.List(repo)
	if err := s.Approve(repo, "make:go", "something else"); err == nil {
		t.Fatal("approved a version that wasn't shown")
	}
	if err := s.Approve(repo, "make:go", listed.Tasks[0].Fingerprint); err != nil {
		t.Fatal(err)
	}
	if list, _ = s.List(repo); !list.Tasks[0].Approved {
		t.Fatalf("after approving = %+v", list)
	}
	if err := s.Run(repo, "make:go", "r2"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the approved command didn't run")
		}
		time.Sleep(20 * time.Millisecond)
	}

	if err := os.WriteFile(makefile, []byte("go:\n\techo changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if list, _ = s.List(repo); list.Tasks[0].Approved {
		t.Fatal("a changed recipe should need approving again")
	}
	if err := s.Run(repo, "make:go", "r1"); !errors.Is(err, errTaskNotApproved) {
		t.Fatalf("after a change: %v", err)
	}
}

func TestChecklistAndCustomCommands(t *testing.T) {
	useTempSettings(t)
	repo := t.TempDir()
	var s TaskService
	if err := s.AddCustom(repo, "deploy", "echo deploy"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddCustom(repo, "deploy", "echo again"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetChecklist(repo, "commit", []string{"custom:deploy"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetChecklist(repo, "push", []string{"make:x"}); err != nil {
		t.Fatal(err)
	}
	list, err := s.List(repo)
	if err != nil || len(list.Tasks) != 1 || list.Tasks[0].Command != "echo again" || !list.Tasks[0].Custom {
		t.Fatalf("list = %+v, %v", list, err)
	}
	if len(list.BeforeCommit) != 1 || list.BeforeCommit[0] != "custom:deploy" || list.BeforePush[0] != "make:x" {
		t.Fatalf("checklist = %+v", list)
	}
	if err := s.RemoveCustom(repo, "deploy"); err != nil {
		t.Fatal(err)
	}
	if list, _ = s.List(repo); len(list.Tasks) != 0 {
		t.Fatalf("after removal = %+v", list)
	}
}

// runTask approves and runs a task, and waits for its result.
func runTask(t *testing.T, worktree, id string) {
	t.Helper()
	var s TaskService
	list, err := s.List(worktree)
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range list.Tasks {
		if task.ID == id {
			if err := s.Approve(worktree, id, task.Fingerprint); err != nil {
				t.Fatal(err)
			}
		}
	}
	key := git.PathKey(worktree)
	before, _, _ := taskResults.Of(key, id)
	if err := s.Run(worktree, id, t.Name()+id+time.Now().String()); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(20 * time.Second)
	for {
		after, _, running := taskResults.Of(key, id)
		if !running && !after.At.Equal(before.At) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s has no result", id)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func resultOf(t *testing.T, worktree, id string) TaskResultInfo {
	t.Helper()
	results, err := new(TaskService).Results(worktree)
	if err != nil {
		t.Fatal(err)
	}
	result, ok := results[id]
	if !ok {
		t.Fatalf("no result for %s in %+v", id, results)
	}
	return result
}

func taskRepo(t *testing.T) string {
	t.Helper()
	useTempSettings(t)
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	for _, setting := range [][2]string{{"user.email", "test@example.com"}, {"user.name", "Test"}, {"core.autocrlf", "false"}} {
		gitIn(t, repo, "config", setting[0], setting[1])
	}
	writeIn(t, repo, "a.txt", "a\n")
	writeIn(t, repo, ".gitignore", "ignored.txt\n")
	gitIn(t, repo, "add", ".")
	gitIn(t, repo, "commit", "-qm", "first")
	var s TaskService
	for name, command := range map[string]string{"pass": "exit 0", "fail": "exit 3", "writes": "echo x>written.txt", "ignored": "echo x>ignored.txt"} {
		if err := s.AddCustom(repo, name, command); err != nil {
			t.Fatal(err)
		}
	}
	return repo
}

func TestAResultIsCurrentOnlyOnTheCodeItRanOn(t *testing.T) {
	repo := taskRepo(t)
	expect := func(when, want string) {
		t.Helper()
		if got := resultOf(t, repo, "custom:pass"); got.State != want || !got.Passed {
			t.Errorf("%s: %+v, want %s", when, got, want)
		}
	}
	writeIn(t, repo, "new.txt", "new\n")
	before := gitOut(t, repo, "count-objects", "-v")
	runTask(t, repo, "custom:pass")
	expect("after it ran", tasks.StateCurrent)
	writeIn(t, repo, "a.txt", "edited\n")
	expect("after an edit", tasks.StateOutdated)
	writeIn(t, repo, "a.txt", "a\n")
	expect("after undoing it", tasks.StateCurrent)
	writeIn(t, repo, "ignored.txt", "x\n")
	expect("after an ignored file changed", tasks.StateCurrent)
	if after := gitOut(t, repo, "count-objects", "-v"); after != before {
		t.Errorf("objects after = %s, before = %s", after, before)
	}
	gitIn(t, repo, "add", "new.txt")
	gitIn(t, repo, "commit", "-qm", "new")
	expect("after committing it", tasks.StateCurrent)
	if err := new(TaskService).AddCustom(repo, "pass", "exit 0 "); err != nil {
		t.Fatal(err)
	}
	expect("after its command changed", tasks.StateOutdated)

	// Turned off by policy, results that exist are still shown.
	if err := os.WriteFile(policy.Path(), []byte(`{"localCommands": {"enabled": false}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := policy.Load(); err != nil {
		t.Fatal(err)
	}
	expect("with local commands off", tasks.StateOutdated)
	if err := new(TaskService).Run(repo, "custom:pass", "r"); !errors.Is(err, errLocalCommandsOff) {
		t.Errorf("running with local commands off: %v", err)
	}
}

func TestAFailureIsAResult(t *testing.T) {
	repo := taskRepo(t)
	runTask(t, repo, "custom:fail")
	if got := resultOf(t, repo, "custom:fail"); got.Passed || got.State != tasks.StateCurrent {
		t.Errorf("a failure = %+v", got)
	}
}

func TestAResultWithAChangeDuringTheRun(t *testing.T) {
	repo := taskRepo(t)
	runTask(t, repo, "custom:writes")
	got := resultOf(t, repo, "custom:writes")
	if got.State != tasks.StateChanged || !reflect.DeepEqual(got.Changed, []string{"written.txt"}) {
		t.Errorf("result = %+v, want it changed by written.txt", got)
	}
	// Run again it writes the same, so nothing changed.
	runTask(t, repo, "custom:writes")
	if got := resultOf(t, repo, "custom:writes"); got.State != tasks.StateCurrent {
		t.Errorf("a second run = %+v", got)
	}
	runTask(t, repo, "custom:ignored")
	if got := resultOf(t, repo, "custom:ignored"); got.State != tasks.StateCurrent {
		t.Errorf("a run that only wrote ignored files = %+v", got)
	}
}

func TestASlowWorktreeDoesntHoldUpARun(t *testing.T) {
	repo := taskRepo(t)
	most := mostReadingCode
	mostReadingCode = 0
	t.Cleanup(func() { mostReadingCode = most })
	runTask(t, repo, "custom:pass")
	if got := resultOf(t, repo, "custom:pass"); !got.Passed || got.State != tasks.StateUnknown {
		t.Errorf("result = %+v, want it passed and unknown", got)
	}
}

func TestAStoppedRunLeavesTheLastResult(t *testing.T) {
	repo := taskRepo(t)
	runTask(t, repo, "custom:pass")
	key := git.PathKey(repo)
	taskResults.Started(key, "custom:pass")
	if _, _, running := taskResults.Of(key, "custom:pass"); !running {
		t.Error("a started task isn't running")
	}
	taskResults.Ended(key, "custom:pass", nil)
	if _, has, running := taskResults.Of(key, "custom:pass"); !has || running {
		t.Errorf("has = %v, running = %v", has, running)
	}
}

func TestANewWorktreeHasTheRepositorysTasksAndNoApprovals(t *testing.T) {
	repo := taskRepo(t)
	runTask(t, repo, "custom:pass")
	var s TaskService
	if err := s.SetOnBoard(repo, []string{"custom:pass"}); err != nil {
		t.Fatal(err)
	}
	// As it was kept before, under one worktree's path.
	if err := updateConfig(func(cfg *config.Config) {
		cfg.CustomTasks[repo] = []config.CustomTask{{Name: "older", Command: "exit 0"}, {Name: "pass", Command: "exit 9"}}
	}); err != nil {
		t.Fatal(err)
	}
	worktree := filepath.Join(t.TempDir(), "other")
	gitIn(t, repo, "worktree", "add", "-q", "-b", "other", worktree)

	list, err := s.List(worktree)
	if err != nil || len(list.Tasks) != 4 || !reflect.DeepEqual(list.OnBoard, []string{"custom:pass"}) {
		t.Fatalf("list = %+v, %v", list, err)
	}
	for _, task := range list.Tasks {
		if task.Approved {
			t.Errorf("%s is approved in a new worktree", task.ID)
		}
	}
	if err := s.Run(worktree, "custom:pass", "r"); !errors.Is(err, errTaskNotApproved) {
		t.Errorf("an unapproved run: %v", err)
	}
	if results, _ := s.Results(worktree); len(results) != 0 {
		t.Errorf("results of a worktree nothing ran in = %+v", results)
	}

	// The repository's own command wins over an older one of its name.
	if list, _ = s.List(repo); len(list.Tasks) != 5 {
		t.Errorf("the first worktree lists %+v", list.Tasks)
	}
	if err := s.RemoveCustom(repo, "pass"); err != nil {
		t.Fatal(err)
	}
	if list, _ = s.List(repo); len(list.Tasks) != 4 {
		t.Errorf("after removing one, it lists %+v", list.Tasks)
	}
}

func TestTheBoardShowsTheChosenTasksResults(t *testing.T) {
	repo := taskRepo(t)
	worktree := filepath.Join(t.TempDir(), "other")
	gitIn(t, repo, "worktree", "add", "-q", "-b", "other", worktree)
	if err := new(TaskService).SetOnBoard(repo, []string{"custom:pass", "gone:task"}); err != nil {
		t.Fatal(err)
	}
	runTask(t, worktree, "custom:pass")
	var s WorktreeService
	states := func() []string {
		t.Helper()
		info, err := s.Board(repo)
		if err != nil {
			t.Fatal(err)
		}
		var all []string
		for _, column := range info.Columns {
			for _, task := range column.Tasks {
				state := "none"
				if task.Result != nil {
					state = task.Result.State
				}
				all = append(all, filepath.Base(column.Worktree.Path)+" "+task.Task.ID+" "+state)
			}
		}
		return all
	}
	main := filepath.Base(repo)
	for _, step := range []struct{ content, want string }{{"a\n", tasks.StateCurrent}, {"edited\n", tasks.StateOutdated}, {"a\n", tasks.StateCurrent}} {
		writeIn(t, worktree, "a.txt", step.content)
		if got, want := states(), []string{main + " custom:pass none", "other custom:pass " + step.want}; !reflect.DeepEqual(got, want) {
			t.Errorf("with %q: %q, want %q", step.content, got, want)
		}
	}
}
