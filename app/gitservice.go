package app

import (
	"context"
	"sync"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
)

type GitService struct{}

type GitSettingInfo struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Scope  string `json:"scope"`
	Origin string `json:"origin"`
}

type GitSettingFlagInfo struct {
	Key     string `json:"key"`
	Message string `json:"message"`
}

type GitSettingsInfo struct {
	Entries []GitSettingInfo     `json:"entries"`
	Flags   []GitSettingFlagInfo `json:"flags"`
}

func (s *GitService) GetGitSettings(repoPath string) (GitSettingsInfo, error) {
	report, err := git.InspectConfig(context.Background(), repoPath)
	if err != nil {
		return GitSettingsInfo{}, err
	}
	info := GitSettingsInfo{Entries: []GitSettingInfo{}, Flags: []GitSettingFlagInfo{}}
	for _, entry := range report.Entries {
		info.Entries = append(info.Entries, GitSettingInfo(entry))
	}
	for _, flag := range report.Flags {
		info.Flags = append(info.Flags, GitSettingFlagInfo(flag))
	}
	return info, nil
}

func (s *GitService) LFSInstalled() bool {
	return git.LFSInstalled(context.Background())
}

// gitVersions holds each git path's version once it's found to be supported,
// so a failure is checked again, after installing Git or changing the path.
var gitVersions sync.Map

func (s *GitService) GetGitVersion() (string, error) {
	path := configOrDefault().GitExecutablePath
	if version, ok := gitVersions.Load(path); ok {
		return version.(string), nil
	}
	info, err := gitexec.DetectSupported(context.Background(), path)
	if err != nil {
		return "", err
	}
	gitVersions.Store(path, info.Version)
	return info.Version, nil
}

// CommandLineToolsMissing is true on a Mac whose only Git is the stand-in
// that Apple's Command Line Tools replace.
func (s *GitService) CommandLineToolsMissing() bool {
	return configOrDefault().GitExecutablePath == "" && gitexec.CommandLineToolsMissing()
}

func (s *GitService) InstallCommandLineTools() error {
	return gitexec.InstallCommandLineTools()
}
