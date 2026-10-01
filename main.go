package main

import (
	"embed"
	"log"

	"github.com/gspeager/current-client/app"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

const (
	defaultWindowWidth  = 1000
	defaultWindowHeight = 618
)

func main() {
	windowWidth, windowHeight := app.Startup()
	if windowWidth <= 0 || windowHeight <= 0 {
		windowWidth, windowHeight = defaultWindowWidth, defaultWindowHeight
	}

	currentClient := application.New(application.Options{
		Name:        "current-client",
		Description: "A privacy-first, cross-platform desktop Git client.",
		Services:    app.Services(),
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	win := currentClient.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Current Client",
		Width:  windowWidth,
		Height: windowHeight,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(10, 14, 22), // design-system.md --surface-lowest (#0a0e16)
		Windows:          application.WindowsWindow{Theme: app.TitleBarTheme()},
		URL:              "/",
	})

	win.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) {
		width, height := win.Size()
		app.SaveWindowSize(width, height)
	})

	if err := currentClient.Run(); err != nil {
		log.Fatal(err)
	}
}
