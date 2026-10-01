//go:build windows

package app

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/w32"
)

func setDarkTitleBar(window application.Window, dark bool) {
	if hwnd := window.NativeWindow(); hwnd != nil {
		w32.SetTheme(uintptr(hwnd), dark)
	}
}

func systemPrefersDark() bool {
	return w32.IsCurrentlyDarkMode()
}
