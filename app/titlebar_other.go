//go:build !windows

package app

import "github.com/wailsapp/wails/v3/pkg/application"

// Only Windows draws a native title bar here: macOS hides it behind Current Client's
// header, and on Linux the window manager themes it.
func setDarkTitleBar(application.Window, bool) {}

func systemPrefersDark() bool {
	return true
}
