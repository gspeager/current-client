package app

import (
	"context"
	"encoding/base64"
	"errors"
	"os"

	"github.com/gspeager/current-client/core/gitexec"
	"github.com/gspeager/current-client/internal/policy"
	"github.com/gspeager/current-client/internal/terminal"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
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

var errLocalCommandsOff = errors.New("your organization has turned this off")

func emit(name string, data any) {
	if running := application.Get(); running != nil {
		running.Event.Emit(name, data)
	}
}

// Start runs the user's shell in the repository and returns the session's ID.
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

// RepoBusy says whether anything is running in a repository's shells.
func (s *TerminalService) RepoBusy(repoPath string) bool {
	return terminals.AnyBusy(repoPath)
}

func (s *TerminalService) Close(id string) error {
	return terminals.Close(id)
}

func (s *TerminalService) CloseRepo(repoPath string) {
	terminals.CloseRepo(repoPath)
}

// Shutdown leaves nothing running when the app quits.
func Shutdown() {
	terminals.CloseAll()
}

// GuardWindowClose asks before quitting with something running in a terminal.
func GuardWindowClose(current *application.App, window *application.WebviewWindow) {
	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if !terminals.AnyBusy("") {
			return
		}
		event.Cancel()
		askBeforeQuitting(current)
	})
}

// ShouldQuit covers quitting from the menu, which doesn't close a window first.
func ShouldQuit() bool {
	if !terminals.AnyBusy("") {
		return true
	}
	if running := application.Get(); running != nil {
		askBeforeQuitting(running)
	}
	return false
}

func askBeforeQuitting(current *application.App) {
	dialog := current.Dialog.Question()
	dialog.SetTitle("Quit Current Client")
	dialog.SetMessage("A process is still running in a terminal. Quitting ends it.")
	quit := dialog.AddButton("Quit")
	quit.OnClick(func() {
		terminals.CloseAll()
		current.Quit()
	})
	cancel := dialog.AddButton("Cancel")
	dialog.SetCancelButton(cancel)
	dialog.Show()
}
