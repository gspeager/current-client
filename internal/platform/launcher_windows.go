package platform

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func launcherPath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "current-client", "bin", launcherName+".cmd"), nil
}

// start returns at once, so the terminal isn't held.
func launcherScript(target string) string {
	return "@echo off\r\nrem " + launcherMarker + ". Settings, Tools and folders, removes it.\r\n" +
		`start "" "` + strings.ReplaceAll(target, "%", "%%") + `" %*` + "\r\n"
}

// The folder is on the user's PATH in the registry, where installing put it. New
// terminals read it from there.
func onPath(dir string) (bool, string) {
	list, _, err := userPath()
	return err == nil && hasDir(list, dir), ""
}

func addToPath(dir string) error {
	list, kind, err := userPath()
	if err != nil {
		return err
	}
	return setUserPath(withDir(list, dir), kind)
}

func removeFromPath(dir string) error {
	list, kind, err := userPath()
	if err != nil {
		return err
	}
	if err := setUserPath(withoutDir(list, dir), kind); err != nil {
		return err
	}
	_ = os.Remove(dir)
	return nil
}

func userPath() (string, uint32, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, "Environment", registry.QUERY_VALUE)
	if err != nil {
		return "", registry.EXPAND_SZ, err
	}
	defer key.Close()
	list, kind, err := key.GetStringValue("Path")
	if errors.Is(err, registry.ErrNotExist) {
		return "", registry.EXPAND_SZ, nil
	}
	return list, kind, err
}

func setUserPath(list string, kind uint32) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, "Environment", registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	if kind == registry.SZ {
		err = key.SetStringValue("Path", list)
	} else {
		err = key.SetExpandStringValue("Path", list)
	}
	if err != nil {
		return err
	}
	// Explorer starts new terminals with the PATH it has, until told it changed.
	environment, _ := windows.UTF16PtrFromString("Environment")
	_, _, _ = windows.NewLazySystemDLL("user32.dll").NewProc("SendMessageTimeoutW").Call(
		uintptr(0xffff), 0x001A, 0, uintptr(unsafe.Pointer(environment)), 0x0002, 5000, 0)
	return nil
}

func sameDir(a, b string) bool {
	if expanded, err := registry.ExpandString(a); err == nil {
		a = expanded
	}
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}
