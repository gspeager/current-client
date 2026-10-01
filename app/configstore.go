package app

import "github.com/gspeager/current-client/internal/config"

func loadCurrentConfig() (config.Config, string, error) {
	path, err := config.DefaultPath()
	if err != nil {
		return config.Config{}, "", err
	}
	cfg, err := config.Load(path)
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

func updateConfig(update func(*config.Config)) error {
	cfg, path, err := loadCurrentConfig()
	if err != nil {
		return err
	}
	update(&cfg)
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
