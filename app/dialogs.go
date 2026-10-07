package app

import (
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Windows reports a cancelled dialog as this error; macOS and Linux return no
// path. Either way the helpers below return "" for a cancel, and pass on any
// other failure.
const dialogCancelled = "cancelled by user"

func dialogResult(path string, err error) (string, error) {
	switch {
	case err == nil:
		return path, nil
	case err.Error() == dialogCancelled:
		return "", nil
	default:
		return "", err
	}
}

func pickFolder(title string) (string, error) {
	dialog := application.Get().Dialog.OpenFile().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		SetTitle(title)
	if dir := configOrDefault().DefaultRepoLocation; dir != "" {
		dialog = dialog.SetDirectory(dir)
	}
	return dialogResult(dialog.PromptForSingleSelection())
}

func pickFile(title string) (string, error) {
	return dialogResult(application.Get().Dialog.OpenFile().
		CanChooseDirectories(false).
		CanChooseFiles(true).
		SetTitle(title).
		PromptForSingleSelection())
}

// saveFile reports false, not an error, when the dialog is cancelled.
func saveFile(message, filename, filterName, pattern string, data []byte) (bool, error) {
	path, err := dialogResult(application.Get().Dialog.SaveFile().
		SetMessage(message).
		SetFilename(filename).
		AddFilter(filterName, pattern).
		PromptForSingleSelection())
	if err != nil || path == "" {
		return false, err
	}
	return true, os.WriteFile(path, data, 0o644)
}
