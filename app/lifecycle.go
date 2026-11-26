package app

import (
	"github.com/gspeager/current-client/internal/config"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Shutdown leaves nothing running when the app quits.
func Shutdown() {
	terminals.CloseAll()
	taskRunner.StopAll()
}

// GuardWindowClose asks before quitting with something running in a terminal.
func GuardWindowClose(current *application.App, window *application.WebviewWindow) {
	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if !terminals.AnyBusy("") && !taskRunner.Running() {
			return
		}
		event.Cancel()
		askBeforeQuitting(current)
	})
}

// ShouldQuit covers quitting from the menu, which doesn't close a window first.
func ShouldQuit() bool {
	if !terminals.AnyBusy("") && !taskRunner.Running() {
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
	dialog.SetMessage("A process is still running in a terminal or task. Quitting ends it.")
	quit := dialog.AddButton("Quit")
	quit.OnClick(func() {
		Shutdown()
		current.Quit()
	})
	cancel := dialog.AddButton("Cancel")
	dialog.SetCancelButton(cancel)
	dialog.Show()
}

func SaveWindowSize(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	_ = updateConfig(func(cfg *config.Config) {
		cfg.WindowWidth = width
		cfg.WindowHeight = height
	})
}
