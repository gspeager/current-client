package app

import (
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/gspeager/current-client/internal/config"
)

// configMu serialises every read and read-modify-write of the settings file:
// the frontend saves several settings at once, and an unlocked update could
// overwrite another's change with what it read before it.
var configMu sync.Mutex

// brokenConfigPath is where an unreadable settings file was moved, until
// SettingsService.TakeSettingsResetNotice reports it.
var brokenConfigPath string

func loadCurrentConfig() (config.Config, string, error) {
	configMu.Lock()
	defer configMu.Unlock()
	return loadConfigLocked()
}

// loadConfigLocked moves a settings file that isn't valid JSON aside and
// starts from defaults, rather than failing every later load and save.
func loadConfigLocked() (config.Config, string, error) {
	path, err := config.DefaultPath()
	if err != nil {
		return config.Config{}, "", err
	}
	cfg, err := config.Load(path)
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &syntaxErr) || errors.As(err, &typeErr) {
		broken := path + ".broken"
		if os.Rename(path, broken) == nil {
			brokenConfigPath = broken
			return config.Default(), path, nil
		}
	}
	if err != nil {
		return config.Config{}, "", err
	}
	return cfg, path, nil
}

// configOrDefault is for reads where a missing or unreadable config file
// shouldn't fail the caller.
func configOrDefault() config.Config {
	cfg, _, err := loadCurrentConfig()
	if err != nil {
		return config.Default()
	}
	return cfg
}

// updateConfig saves nothing when update returns an error.
func updateConfig(update func(*config.Config)) error {
	return updateConfigErr(func(cfg *config.Config) error {
		update(cfg)
		return nil
	})
}

func updateConfigErr(update func(*config.Config) error) error {
	configMu.Lock()
	defer configMu.Unlock()
	cfg, path, err := loadConfigLocked()
	if err != nil {
		return err
	}
	if err := update(&cfg); err != nil {
		return err
	}
	return config.Save(path, cfg)
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

func mapSlice[T, U any](in []T, convert func(T) U) []U {
	out := make([]U, len(in))
	for i, v := range in {
		out[i] = convert(v)
	}
	return out
}
