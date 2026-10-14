package app

import (
	"context"
	"sync"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
)

type GitService struct{}

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
