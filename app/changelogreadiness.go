package app

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/gspeager/current-client/core/changelog"
	"github.com/gspeager/current-client/internal/config"
)

// ReleaseCheck and ReleaseReadiness are changelog.Check and changelog.Readiness
// for the bindings.
type ReleaseCheck struct {
	Label    string   `json:"label"`
	Level    string   `json:"level"`
	Value    string   `json:"value"`
	Evidence []string `json:"evidence"`
	// AsOf is when a host fact was read, for a check from the host cache.
	AsOf *time.Time `json:"asOf"`
}

type ReleaseReadiness struct {
	Ref      string `json:"ref"`
	SHA      string `json:"sha"`
	Version  string `json:"version"`
	Previous string `json:"previous"`
	Target   string `json:"target"`
	// Tag is the tag the version takes, with a "v" when the last one had it.
	Tag string `json:"tag"`
	// TaggedAs is the tag already on the candidate, if any.
	TaggedAs string         `json:"taggedAs"`
	Checks   []ReleaseCheck `json:"checks"`
}

// ReleaseReadiness is changelog.CheckRelease with CI and protection from the
// host cache when the host is on. It publishes nothing.
func (s *ChangelogService) ReleaseReadiness(repoPath, ref, version, target string) (ReleaseReadiness, error) {
	checked, err := changelog.CheckRelease(context.Background(), repoPath, ref, version, target, configOrDefault().ReleaseVersionFiles[repoPath])
	if err != nil {
		return ReleaseReadiness{}, err
	}
	r := ReleaseReadiness{Ref: checked.Ref, SHA: checked.SHA, Version: checked.Version, Previous: checked.Previous, Target: checked.Target, Tag: checked.Tag, TaggedAs: checked.TaggedAs}
	r.Checks = mapSlice(checked.Checks, func(c changelog.Check) ReleaseCheck { return ReleaseCheck(c) })
	r.Checks = append(r.Checks, releaseHostChecks(repoPath, r.SHA, r.Target)...)
	return r, nil
}

func (s *ChangelogService) VersionFiles(repoPath string) []string {
	return configOrDefault().ReleaseVersionFiles[repoPath]
}

func (s *ChangelogService) SetVersionFiles(repoPath string, files []string) error {
	var kept []string
	for _, f := range files {
		if f = strings.TrimSpace(f); f != "" {
			kept = append(kept, filepath.ToSlash(f))
		}
	}
	return updateConfig(func(cfg *config.Config) {
		if cfg.ReleaseVersionFiles == nil {
			cfg.ReleaseVersionFiles = map[string][]string{}
		}
		if len(kept) == 0 {
			delete(cfg.ReleaseVersionFiles, repoPath)
			return
		}
		cfg.ReleaseVersionFiles[repoPath] = kept
	})
}
