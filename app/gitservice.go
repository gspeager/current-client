package app

import (
	"context"

	"github.com/gspeager/current-client/core/gitexec"
)

type GitService struct{}

func (s *GitService) GetGitVersion() (string, error) {
	info, err := gitexec.DetectSupported(context.Background(), configOrDefault().GitExecutablePath)
	if err != nil {
		return "", err
	}
	return info.Version, nil
}
