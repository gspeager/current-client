package app

import (
	"context"
	"time"

	"github.com/gspeager/current-client/core/git"
)

type RemoteService struct{}

type RemoteInfo struct {
	Name     string `json:"name"`
	FetchURL string `json:"fetchUrl"`
	PushURL  string `json:"pushUrl"`
}

func (s *RemoteService) List(repoPath string) ([]RemoteInfo, error) {
	remotes, err := git.ListRemotes(context.Background(), repoPath)
	if err != nil {
		return nil, err
	}
	return mapSlice(remotes, func(r git.Remote) RemoteInfo { return RemoteInfo(r) }), nil
}

func (s *RemoteService) AddRemote(repoPath, name, url string) error {
	return git.AddRemote(context.Background(), repoPath, name, url)
}

func (s *RemoteService) RemoveRemote(repoPath, name string) error {
	return git.RemoveRemote(context.Background(), repoPath, name)
}

func (s *RemoteService) EditRemote(repoPath, name, url string) error {
	return git.EditRemote(context.Background(), repoPath, name, url)
}

// Network operations take Wails' per-call ctx so the frontend can cancel them.

func (s *RemoteService) Fetch(ctx context.Context, repoPath, remote string) error {
	return git.Fetch(ctx, repoPath, remote)
}

func (s *RemoteService) FetchAll(ctx context.Context, repoPath string) error {
	return git.FetchAll(ctx, repoPath)
}

func (s *RemoteService) LastFetchTime(repoPath string) (*time.Time, error) {
	return git.LastFetchTime(context.Background(), repoPath)
}

func (s *RemoteService) Pull(ctx context.Context, repoPath string) error {
	return git.Pull(ctx, repoPath)
}

func (s *RemoteService) Push(ctx context.Context, repoPath string) error {
	return git.Push(ctx, repoPath)
}

func (s *RemoteService) PushSetUpstream(ctx context.Context, repoPath, remote, branch string) error {
	return git.PushSetUpstream(ctx, repoPath, remote, branch)
}

func (s *RemoteService) ForcePush(ctx context.Context, repoPath, remote, branch string) error {
	return git.ForcePush(ctx, repoPath, remote, branch)
}
