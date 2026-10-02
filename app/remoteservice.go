package app

import (
	"context"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
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

// Network operations take Wails' per-call ctx so the frontend can cancel them,
// and auth: nil, or what the user typed into the sign-in form after Git found
// no credentials.

// withAuth offers auth to Git for this one call; Git's own credential helpers
// decide whether it's remembered.
func withAuth(ctx context.Context, auth *gitexec.Credential) (context.Context, error) {
	if auth == nil {
		return ctx, nil
	}
	if err := gitexec.ValidateCredential(*auth); err != nil {
		return nil, err
	}
	return gitexec.WithCredential(ctx, *auth), nil
}

// CredentialHelper names the helper Git stores sign-ins with, or "" when
// none is set up and a sign-in is used once and forgotten.
func (s *RemoteService) CredentialHelper() string {
	return git.CredentialHelper(context.Background(), "")
}

func (s *RemoteService) Fetch(ctx context.Context, repoPath, remote string, auth *gitexec.Credential) error {
	ctx, err := withAuth(ctx, auth)
	if err != nil {
		return err
	}
	return git.Fetch(ctx, repoPath, remote)
}

func (s *RemoteService) FetchAll(ctx context.Context, repoPath string, auth *gitexec.Credential) error {
	ctx, err := withAuth(ctx, auth)
	if err != nil {
		return err
	}
	return git.FetchAll(ctx, repoPath)
}

func (s *RemoteService) LastFetchTime(repoPath string) (*time.Time, error) {
	return git.LastFetchTime(context.Background(), repoPath)
}

func (s *RemoteService) Pull(ctx context.Context, repoPath string, auth *gitexec.Credential) error {
	ctx, err := withAuth(ctx, auth)
	if err != nil {
		return err
	}
	return git.Pull(ctx, repoPath)
}

func (s *RemoteService) Push(ctx context.Context, repoPath string, auth *gitexec.Credential) error {
	ctx, err := withAuth(ctx, auth)
	if err != nil {
		return err
	}
	return git.Push(ctx, repoPath)
}

func (s *RemoteService) PushSetUpstream(ctx context.Context, repoPath, remote, branch string, auth *gitexec.Credential) error {
	ctx, err := withAuth(ctx, auth)
	if err != nil {
		return err
	}
	return git.PushSetUpstream(ctx, repoPath, remote, branch)
}

func (s *RemoteService) ForcePush(ctx context.Context, repoPath, remote, branch string, auth *gitexec.Credential) error {
	ctx, err := withAuth(ctx, auth)
	if err != nil {
		return err
	}
	return git.ForcePush(ctx, repoPath, remote, branch)
}
