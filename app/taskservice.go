package app

import (
	"encoding/base64"
	"errors"
	"os"
	"slices"

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
	Custom   bool   `json:"custom"`
}

type TaskListInfo struct {
	Tasks        []TaskInfo `json:"tasks"`
	BeforeCommit []string   `json:"beforeCommit"`
	BeforePush   []string   `json:"beforePush"`
	Dir          string     `json:"dir"`
}

var taskRunner = tasks.NewRunner()

var errTaskNotApproved = errors.New("this command hasn't been approved")

type TaskService struct{}

func customTasks(cfg config.Config, repoPath string) []tasks.Custom {
	return mapSlice(cfg.CustomTasks[repoPath], func(item config.CustomTask) tasks.Custom {
		return tasks.Custom{Name: item.Name, Command: item.Command}
	})
}

func (s *TaskService) List(repoPath string) (TaskListInfo, error) {
	cfg := configOrDefault()
	checklist := cfg.TaskChecklist[repoPath]
	list := TaskListInfo{
		Tasks:        []TaskInfo{},
		BeforeCommit: append([]string{}, checklist.BeforeCommit...),
		BeforePush:   append([]string{}, checklist.BeforePush...),
		Dir:          repoPath,
	}
	for _, task := range tasks.Discover(repoPath, customTasks(cfg, repoPath)) {
		list.Tasks = append(list.Tasks, TaskInfo{
			ID:       task.ID,
			Source:   task.Source,
			Name:     task.Name,
			Command:  task.Command,
			File:     task.File,
			Note:     task.Note,
			Approved: cfg.TaskApprovals[repoPath][task.ID] == tasks.Fingerprint(repoPath, task),
			Custom:   task.Source == tasks.SourceCustom,
		})
	}
	return list, nil
}

// Approve records what the user saw, so it asks again if the command changes.
func (s *TaskService) Approve(repoPath, id string) error {
	cfg := configOrDefault()
	task, ok := tasks.Find(repoPath, customTasks(cfg, repoPath), id)
	if !ok {
		return errors.New("this command is no longer here")
	}
	fingerprint := tasks.Fingerprint(repoPath, task)
	return updateConfig(func(cfg *config.Config) {
		if cfg.TaskApprovals == nil {
			cfg.TaskApprovals = map[string]map[string]string{}
		}
		if cfg.TaskApprovals[repoPath] == nil {
			cfg.TaskApprovals[repoPath] = map[string]string{}
		}
		cfg.TaskApprovals[repoPath][id] = fingerprint
	})
}

// Run starts an approved command. It checks again here, so nothing that changed
// since the list was read runs without being seen.
func (s *TaskService) Run(repoPath, id string) (string, error) {
	if policy.LocalCommandsOff() {
		return "", errLocalCommandsOff
	}
	cfg := configOrDefault()
	task, ok := tasks.Find(repoPath, customTasks(cfg, repoPath), id)
	if !ok {
		return "", errors.New("this command is no longer here")
	}
	if cfg.TaskApprovals[repoPath][id] != tasks.Fingerprint(repoPath, task) {
		return "", errTaskNotApproved
	}
	var runID string
	started := make(chan struct{})
	runID, err := taskRunner.Start(repoPath, task.Command, tasks.Options{
		Env: os.Environ(),
		OnOutput: func(data []byte) {
			<-started
			emit(taskOutputEvent, TaskOutputEvent{runID, base64.StdEncoding.EncodeToString(data)})
		},
		OnDone: func(result tasks.Result) {
			<-started
			emit(taskDoneEvent, TaskDoneEvent{runID, result.ExitCode, result.Duration.Milliseconds(), result.Stopped})
		},
	})
	close(started)
	return runID, err
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

func (s *TaskService) AddCustom(repoPath, name, command string) error {
	return updateConfig(func(cfg *config.Config) {
		if cfg.CustomTasks == nil {
			cfg.CustomTasks = map[string][]config.CustomTask{}
		}
		kept := slices.DeleteFunc(slices.Clone(cfg.CustomTasks[repoPath]), func(item config.CustomTask) bool { return item.Name == name })
		cfg.CustomTasks[repoPath] = append(kept, config.CustomTask{Name: name, Command: command})
	})
}

func (s *TaskService) RemoveCustom(repoPath, name string) error {
	return updateConfig(func(cfg *config.Config) {
		cfg.CustomTasks[repoPath] = slices.DeleteFunc(slices.Clone(cfg.CustomTasks[repoPath]), func(item config.CustomTask) bool { return item.Name == name })
	})
}
