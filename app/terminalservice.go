package app

import (
	"context"
	"encoding/base64"
	"os"

	"github.com/gspeager/current-client/core/gitexec"
	"github.com/gspeager/current-client/internal/policy"
	"github.com/gspeager/current-client/internal/terminal"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	terminalOutputEvent = "terminal:output"
	terminalExitEvent   = "terminal:exit"
)

func init() {
	application.RegisterEvent[TerminalOutputEvent](terminalOutputEvent)
	application.RegisterEvent[TerminalExitEvent](terminalExitEvent)
}

// TerminalOutputEvent.Data is base64, since a chunk can end inside a character.
type TerminalOutputEvent struct {
	ID   string `json:"id"`
	Data string `json:"data"`
}

type TerminalExitEvent struct {
	ID   string `json:"id"`
	Code int    `json:"code"`
}

var terminals = terminal.NewManager()

type TerminalService struct{}

var errLocalCommandsOff = &gitexec.AppError{Message: "Turned off by organization policy."}

func (s *TerminalService) Start(repoPath string, cols, rows int) (string, error) {
	if policy.LocalCommandsOff() {
		return "", errLocalCommandsOff
	}
	cfg := configOrDefault()
	gitPath := ""
	if info, err := gitexec.Detect(context.Background(), cfg.GitExecutablePath); err == nil {
		gitPath = info.Path
	}
	name, args := terminal.Shell(cfg.TerminalShell, gitPath)
	var id string
	started := make(chan struct{})
	id, err := terminals.Start(repoPath, terminal.Options{
		Dir:  repoPath,
		Name: name,
		Args: args,
		Env:  terminal.Environment(os.Environ(), gitPath),
		Cols: cols,
		Rows: rows,
		Hold: true,
		OnOutput: func(data []byte) {
			<-started
			emit(terminalOutputEvent, TerminalOutputEvent{id, base64.StdEncoding.EncodeToString(data)})
		},
		OnExit: func(code int) {
			<-started
			emit(terminalExitEvent, TerminalExitEvent{id, code})
		},
	})
	close(started)
	return id, err
}

// Attach says the view is listening, so output can start.
func (s *TerminalService) Attach(id string) {
	terminals.Attach(id)
}

func (s *TerminalService) Write(id, data string) error {
	return terminals.Write(id, []byte(data))
}

func (s *TerminalService) Resize(id string, cols, rows int) error {
	return terminals.Resize(id, cols, rows)
}

func (s *TerminalService) Ack(id string, bytes int) {
	terminals.Ack(id, bytes)
}

func (s *TerminalService) Busy(id string) bool {
	return terminals.Busy(id)
}

func (s *TerminalService) RepoBusy(repoPath string) bool {
	return terminals.AnyBusy(repoPath)
}

func (s *TerminalService) Close(id string) error {
	return terminals.Close(id)
}

func (s *TerminalService) CloseRepo(repoPath string) {
	terminals.CloseRepo(repoPath)
}
