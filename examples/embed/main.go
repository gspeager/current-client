// Command embed shows another app hosting Current Client: Current Client's services and settings
// alongside the host's own view, switched from one window.
package main

import (
	"embed"
	"log"

	"github.com/gspeager/current-client/app"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app.SetSettingsFolder("current-client-embed-example")
	width, height := app.Startup()
	if width <= 0 || height <= 0 {
		width, height = 1000, 618
	}

	host := application.New(application.Options{
		Name:     "current-client-embed-example",
		Services: app.Services(),
		Assets:   application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
	})
	host.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:   "Current Client embed example",
		Width:   width,
		Height:  height,
		Windows: application.WindowsWindow{Theme: app.TitleBarTheme()},
		URL:     "/",
	})
	if err := host.Run(); err != nil {
		log.Fatal(err)
	}
}
