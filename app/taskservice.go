package app

import (
	"context"
	"encoding/base64"
	"os"
	"slices"
	"time"

	"github.com/gspeager/current-client/core/board"
	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
	"github.com/gspeager/current-client/internal/config"
	"github.com/gspeager/current-client/internal/policy"
	"github.com/gspeager/current-client/internal/tasks"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	taskOutputEvent = "task:output"
	taskDoneEvent   = "task:done"
)

func init() {
	application.RegisterEvent[TaskOutputEvent](taskOutputEvent)
	application.RegisterEvent[TaskDoneEvent](taskDoneEvent)
}

type TaskOutputEvent struct {
	RunID string `json:"runId"`
	Data  string `json:"data"`
}

type TaskDoneEvent struct {
	RunID      string `json:"runId"`
	ExitCode   int    `json:"exitCode"`
	DurationMs int64  `json:"durationMs"`
	Stopped    bool   `json:"stopped"`
}

// TaskInfo.Approved is false until the user has seen this command, and again if
// the command or the file it comes from has changed since.
type TaskInfo struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Name     string `json:"name"`
	Command  string `json:"command"`
	File     string `json:"file"`
	Note     string `json:"note"`
	Approved bool   `json:"approved"`
	// Fingerprint goes back to Approve, so approval covers what was shown.
	Fingerprint string `json:"fingerprint"`
	Custom      bool   `json:"custom"`
}

type TaskListInfo struct {
	Tasks        []TaskInfo `json:"tasks"`
	BeforeCommit []string   `json:"beforeCommit"`
	BeforePush   []string   `json:"beforePush"`
	// Shown on the worktree board.
	OnBoard []string `json:"onBoard"`
	Dir     string   `json:"dir"`
}

// TaskResultInfo is a task's last result. State is "current" on the code as it is
// now, "outdated", "changed" when files changed while it ran, which are Changed, or
// "unknown".
type TaskResultInfo struct {
	Passed     bool      `json:"passed"`
	At         time.Time `json:"at"`
	DurationMs int64     `json:"durationMs"`
	State      string    `json:"state"`
	Changed    []string  `json:"changed"`
}

var (
	taskRunner  = tasks.NewRunner()
	taskResults = tasks.NewResults()
)

var errTaskNotApproved = &gitexec.AppError{Message: "This command hasn't been approved."}

type TaskService struct{}

// tasksKey is what a repository's own commands are kept under, the same for all its
// worktrees.
func tasksKey(repoPath string) string {
	commonDir, err := git.CommonDir(context.Background(), repoPath)
	if err != nil {
		return repoPath
	}
	return git.PathKey(commonDir)
}

func customTasks(cfg config.Config, repoPath string) []tasks.Custom {
	kept := func(key string) []tasks.Custom {
		return mapSlice(cfg.CustomTasks[key], func(item config.CustomTask) tasks.Custom {
			return tasks.Custom{Name: item.Name, Command: item.Command}
		})
	}
	return tasks.MergeCustom(kept(tasksKey(repoPath)), kept(repoPath))
}

func taskInfo(cfg config.Config, repoPath string, task tasks.Task) TaskInfo {
	fingerprint := tasks.Fingerprint(repoPath, task)
	return TaskInfo{
		ID:          task.ID,
		Source:      task.Source,
		Name:        task.Name,
		Command:     task.Command,
		File:        task.File,
		Note:        task.Note,
		Approved:    cfg.TaskApprovals[repoPath][task.ID] == fingerprint,
		Fingerprint: fingerprint,
		Custom:      task.Source == tasks.SourceCustom,
	}
}

func (s *TaskService) List(repoPath string) (TaskListInfo, error) {
	cfg := configOrDefault()
	checklist := cfg.TaskChecklist[repoPath]
	list := TaskListInfo{
		Tasks:        []TaskInfo{},
		BeforeCommit: append([]string{}, checklist.BeforeCommit...),
		BeforePush:   append([]string{}, checklist.BeforePush...),
		OnBoard:      append([]string{}, cfg.BoardTasks[tasksKey(repoPath)]...),
		Dir:          repoPath,
	}
	for _, task := range tasks.Discover(repoPath, customTasks(cfg, repoPath)) {
		list.Tasks = append(list.Tasks, taskInfo(cfg, repoPath, task))
	}
	return list, nil
}

// Results are the last result of each of the worktree's tasks that has one, by id.
func (s *TaskService) Results(repoPath string) (map[string]TaskResultInfo, error) {
	results := map[string]TaskResultInfo{}
	worktree := git.PathKey(repoPath)
	if !taskResults.Any(worktree) {
		return results, nil
	}
	ctx := context.Background()
	env, discard := git.ScratchWrites(ctx, repoPath)
	defer discard()
	// Without it the state is unknown.
	tree, _ := board.Tree(ctx, repoPath, env)
	for _, task := range tasks.Discover(repoPath, customTasks(configOrDefault(), repoPath)) {
		if outcome, has, _ := taskResults.Of(worktree, task.ID); has {
			results[task.ID] = resultInfo(outcome, tree, task.Command)
		}
	}
	return results, nil
}

func resultInfo(outcome tasks.Outcome, tree, command string) TaskResultInfo {
	return TaskResultInfo{
		Passed:     outcome.ExitCode == 0,
		At:         outcome.At,
		DurationMs: outcome.Duration.Milliseconds(),
		State:      outcome.State(tree, command),
		Changed:    outcome.Changed,
	}
}

// Approve records what the user saw, so it asks again if the command changes. It
// refuses if the file changed while they were reading.
func (s *TaskService) Approve(repoPath, id, seen string) error {
	cfg := configOrDefault()
	task, ok := tasks.Find(repoPath, customTasks(cfg, repoPath), id)
	if !ok {
		return &gitexec.AppError{Message: "This command is no longer here."}
	}
	if tasks.Fingerprint(repoPath, task) != seen {
		return &gitexec.AppError{Message: "This command changed while it was shown. Run it again to see the new version."}
	}
	return updateConfig(func(cfg *config.Config) {
		if cfg.TaskApprovals == nil {
			cfg.TaskApprovals = map[string]map[string]string{}
		}
		if cfg.TaskApprovals[repoPath] == nil {
			cfg.TaskApprovals[repoPath] = map[string]string{}
		}
		cfg.TaskApprovals[repoPath][id] = seen
	})
}

// Run starts an approved command. It checks again here, so nothing that changed
// since the list was read runs without being seen.
func (s *TaskService) Run(repoPath, id, runID string) error {
	if policy.LocalCommandsOff() {
		return errLocalCommandsOff
	}
	cfg := configOrDefault()
	task, ok := tasks.Find(repoPath, customTasks(cfg, repoPath), id)
	if !ok {
		return &gitexec.AppError{Message: "This command is no longer here."}
	}
	if cfg.TaskApprovals[repoPath][id] != tasks.Fingerprint(repoPath, task) {
		return errTaskNotApproved
	}
	worktree := git.PathKey(repoPath)
	code := readCode(repoPath)
	taskResults.Started(worktree, id)
	err := taskRunner.Start(runID, repoPath, task.Command, tasks.Options{
		Env: os.Environ(),
		OnOutput: func(data []byte) {
			emit(taskOutputEvent, TaskOutputEvent{runID, base64.StdEncoding.EncodeToString(data)})
		},
		OnDone: func(result tasks.Result) {
			tree, changed := code.after()
			if result.Stopped {
				taskResults.Ended(worktree, id, nil)
			} else {
				taskResults.Ended(worktree, id, &tasks.Outcome{
					Command: task.Command, ExitCode: result.ExitCode, Duration: result.Duration, At: time.Now(), Tree: tree, Changed: changed,
				})
			}
			emit(taskDoneEvent, TaskDoneEvent{runID, result.ExitCode, result.Duration.Milliseconds(), result.Stopped})
		},
	})
	if err != nil {
		code.discard()
		taskResults.Ended(worktree, id, nil)
	}
	return err
}

// code is a worktree's as a task starts. Its objects are kept until the task ends, to
// say what changed meanwhile.
type code struct {
	repoPath string
	tree     string
	env      []string
	discard  func()
}

// A command isn't held up for long by a worktree that's slow to read. Its result's
// state is unknown then.
var mostReadingCode = 5 * time.Second

func readCode(repoPath string) code {
	ctx, cancel := context.WithTimeout(context.Background(), mostReadingCode)
	defer cancel()
	c := code{repoPath: repoPath}
	c.env, c.discard = git.ScratchWrites(ctx, repoPath)
	c.tree, _ = board.Tree(ctx, repoPath, c.env)
	return c
}

// after is the tree the task ran on, "" when that can't be worked out, and the files
// that changed meanwhile.
func (c code) after() (tree string, changed []string) {
	defer c.discard()
	ctx := context.Background()
	now, err := board.Tree(ctx, c.repoPath, c.env)
	if err != nil || c.tree == "" {
		return "", nil
	}
	if changed, err = board.Differing(ctx, c.repoPath, c.env, c.tree, now); err != nil {
		return "", nil
	}
	return c.tree, changed
}

func (s *TaskService) Stop(runID string) {
	taskRunner.Stop(runID)
}

// SetChecklist chooses what runs alongside a commit or push. kind is "commit" or
// "push". It warns and never blocks, and installs no Git hooks.
func (s *TaskService) SetChecklist(repoPath, kind string, ids []string) error {
	return updateConfig(func(cfg *config.Config) {
		if cfg.TaskChecklist == nil {
			cfg.TaskChecklist = map[string]config.TaskChecklist{}
		}
		checklist := cfg.TaskChecklist[repoPath]
		if kind == "push" {
			checklist.BeforePush = ids
		} else {
			checklist.BeforeCommit = ids
		}
		cfg.TaskChecklist[repoPath] = checklist
	})
}

// SetOnBoard chooses the tasks the worktree board shows, for every worktree.
func (s *TaskService) SetOnBoard(repoPath string, ids []string) error {
	return updateConfig(func(cfg *config.Config) {
		if cfg.BoardTasks == nil {
			cfg.BoardTasks = map[string][]string{}
		}
		cfg.BoardTasks[tasksKey(repoPath)] = ids
	})
}

func (s *TaskService) AddCustom(repoPath, name, command string) error {
	return updateConfig(func(cfg *config.Config) {
		dropCustom(cfg, repoPath, name)
		if cfg.CustomTasks == nil {
			cfg.CustomTasks = map[string][]config.CustomTask{}
		}
		key := tasksKey(repoPath)
		cfg.CustomTasks[key] = append(cfg.CustomTasks[key], config.CustomTask{Name: name, Command: command})
	})
}

func (s *TaskService) RemoveCustom(repoPath, name string) error {
	return updateConfig(func(cfg *config.Config) { dropCustom(cfg, repoPath, name) })
}

func dropCustom(cfg *config.Config, repoPath, name string) {
	for _, key := range []string{tasksKey(repoPath), repoPath} {
		if items, ok := cfg.CustomTasks[key]; ok {
			cfg.CustomTasks[key] = slices.DeleteFunc(slices.Clone(items), func(item config.CustomTask) bool { return item.Name == name })
		}
	}
}
