package app

import (
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// The dialog API reports a cancelled prompt as an error, so these return ""
// instead of failing.

func pickFolder(title string) (string, error) {
	dialog := application.Get().Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		SetTitle(title)
	if dir := configOrDefault().DefaultRepoLocation; dir != "" {
		dialog = dialog.SetDirectory(dir)
	}
	path, err := dialog.PromptForSingleSelection()
	if err != nil {
		return "", nil
	}
	return path, nil
}

func pickFile(title string) (string, error) {
	path, err := application.Get().Dialog.OpenFile().
		CanChooseDirectories(false).
		CanChooseFiles(true).
		SetTitle(title).
		PromptForSingleSelection()
	if err != nil {
		return "", nil
	}
	return path, nil
}

// saveFile reports false, not an error, when the dialog is cancelled.
func saveFile(message, filename, filterName, pattern string, data []byte) (bool, error) {
	path, err := application.Get().Dialog.SaveFile().
		SetMessage(message).
		SetFilename(filename).
		AddFilter(filterName, pattern).
		PromptForSingleSelection()
	if err != nil || path == "" {
		return false, nil
	}
	return true, os.WriteFile(path, data, 0o644)
}
