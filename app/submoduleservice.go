package app

import (
	"context"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
)

type SubmoduleService struct{}

type SubmoduleInfo struct {
	Path          string `json:"path"`
	Commit        string `json:"commit"`
	Initialized   bool   `json:"initialized"`
	CommitChanged bool   `json:"commitChanged"`
	Conflicted    bool   `json:"conflicted"`
}

type SubmoduleCommit struct {
	SHA     string `json:"sha"`
	Subject string `json:"subject"`
	Added   bool   `json:"added"`
}

func (s *SubmoduleService) List(repoPath string) ([]SubmoduleInfo, error) {
	submodules, err := git.ListSubmodules(context.Background(), repoPath)
	if err != nil {
		return nil, err
	}
	return mapSlice(submodules, func(sm git.Submodule) SubmoduleInfo { return SubmoduleInfo(sm) }), nil
}

// Update fetches over the network like any other remote operation, so it takes
// Wails' per-call ctx for cancelling and auth for the sign-in retry. No paths
// means every submodule.
func (s *SubmoduleService) Update(ctx context.Context, repoPath string, paths []string, auth *gitexec.Credential) error {
	ctx, err := withAuth(ctx, auth)
	if err != nil {
		return err
	}
	return git.UpdateSubmodules(ctx, repoPath, paths)
}

func (s *SubmoduleService) Commits(repoPath, path, from, to string) ([]SubmoduleCommit, error) {
	commits, err := git.SubmoduleCommits(context.Background(), repoPath, path, from, to)
	if err != nil {
		return nil, err
	}
	return mapSlice(commits, func(c git.SubmoduleCommit) SubmoduleCommit { return SubmoduleCommit(c) }), nil
}
