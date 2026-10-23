package app

import (
	"context"
	"strings"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
	"github.com/gspeager/current-client/core/hostlink"
	"github.com/gspeager/current-client/internal/config"
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

// syncWithRemote runs a fetch or pull with auth offered to Git, then refreshes
// what the host side knows, since a fetch is one of the refresh triggers.
func syncWithRemote(ctx context.Context, repoPath string, auth *gitexec.Credential, op func(context.Context) error) error {
	ctx, err := withAuth(ctx, auth)
	if err != nil {
		return err
	}
	if err := op(ctx); err != nil {
		return err
	}
	afterFetch(ctx, repoPath)
	return nil
}

// CredentialHelper names the helper Git stores sign-ins with, or "" when
// none is set up and a sign-in is used once and forgotten.
func (s *RemoteService) CredentialHelper() string {
	return git.CredentialHelper(context.Background(), "")
}

func (s *RemoteService) Fetch(ctx context.Context, repoPath, remote string, auth *gitexec.Credential) error {
	return syncWithRemote(ctx, repoPath, auth, func(ctx context.Context) error { return git.Fetch(ctx, repoPath, remote) })
}

func (s *RemoteService) FetchAll(ctx context.Context, repoPath string, auth *gitexec.Credential) error {
	return syncWithRemote(ctx, repoPath, auth, func(ctx context.Context) error { return git.FetchAll(ctx, repoPath) })
}

func (s *RemoteService) FetchAllPrune(ctx context.Context, repoPath string, auth *gitexec.Credential) error {
	return syncWithRemote(ctx, repoPath, auth, func(ctx context.Context) error { return git.FetchAllPrune(ctx, repoPath) })
}

func (s *RemoteService) LastFetchTime(repoPath string) (*time.Time, error) {
	return git.LastFetchTime(context.Background(), repoPath)
}

func (s *RemoteService) Pull(ctx context.Context, repoPath string, auth *gitexec.Credential) error {
	return syncWithRemote(ctx, repoPath, auth, func(ctx context.Context) error { return git.Pull(ctx, repoPath) })
}

func (s *RemoteService) PullMerge(ctx context.Context, repoPath string, auth *gitexec.Credential) error {
	return syncWithRemote(ctx, repoPath, auth, func(ctx context.Context) error { return git.PullMerge(ctx, repoPath) })
}

func (s *RemoteService) PullRebase(ctx context.Context, repoPath string, auth *gitexec.Credential) error {
	return syncWithRemote(ctx, repoPath, auth, func(ctx context.Context) error { return git.PullRebase(ctx, repoPath) })
}

// PushResult says what was pushed and, when the branch has no known merge
// request / pull request, where to create one: the link the host printed,
// else one built from the remote's address. Kind names the host's word.
type PushResult struct {
	Branch     string `json:"branch"`
	Upstream   string `json:"upstream"`
	CreateLink string `json:"createLink"`
	Kind       string `json:"kind"`
}

func (s *RemoteService) Push(ctx context.Context, repoPath string, auth *gitexec.Credential) (PushResult, error) {
	return s.push(ctx, repoPath, auth, func(ctx context.Context) (string, error) { return git.PushOutput(ctx, repoPath) })
}

func (s *RemoteService) PushSetUpstream(ctx context.Context, repoPath, remote, branch string, auth *gitexec.Credential) (PushResult, error) {
	return s.push(ctx, repoPath, auth, func(ctx context.Context) (string, error) {
		return git.PushSetUpstreamOutput(ctx, repoPath, remote, branch)
	})
}

// push runs a push with auth offered to Git and reads what the remote printed.
func (s *RemoteService) push(ctx context.Context, repoPath string, auth *gitexec.Credential, push func(context.Context) (string, error)) (PushResult, error) {
	ctx, err := withAuth(ctx, auth)
	if err != nil {
		return PushResult{}, err
	}
	output, err := push(ctx)
	if err != nil {
		return PushResult{}, err
	}
	status, err := git.CurrentBranchStatus(ctx, repoPath)
	if err != nil {
		return PushResult{}, err
	}
	result := PushResult{Branch: status.Current, Upstream: status.Upstream}
	if link, kind, err := s.createChangeLink(ctx, repoPath, status.Current); err == nil && link != "" {
		result.CreateLink, result.Kind = link, kind
		if printed := createLinkIn(output); printed != "" {
			result.CreateLink = printed
		}
	}
	return result, nil
}

// CreateChangeLink is the host's pre-filled form for a merge request / pull
// request from branch, built from the remote's address, no API. For a fork
// (a remote named upstream on the same host) the target is upstream's
// project. Empty when no remote points at a hosted project.
func (s *RemoteService) CreateChangeLink(repoPath, branch string) (string, error) {
	link, _, err := s.createChangeLink(context.Background(), repoPath, branch)
	return link, err
}

func (s *RemoteService) createChangeLink(ctx context.Context, repoPath, branch string) (link, kind string, err error) {
	remotes, err := git.ListRemotes(ctx, repoPath)
	if err != nil || len(remotes) == 0 {
		return "", "", err
	}
	primary := primaryRemote(remotes)
	host, ok := hostlink.Resolve(ctx, remoteURL(primary))
	if !ok {
		return "", "", nil
	}
	host = withHostConfig(host, configOrDefault().Hosts)
	kind = "pull request"
	if host.Style == hostlink.GitLab {
		kind = "merge request"
	}
	base, _ := git.DefaultBranch(ctx, repoPath)
	branch = remoteBranchOf(ctx, repoPath, branch)
	for _, r := range remotes {
		if r.Name != "upstream" || r.Name == primary.Name {
			continue
		}
		if upstream, ok := hostlink.Resolve(ctx, remoteURL(r)); ok && upstream.Hostname() == host.Hostname() {
			return host.NewChangeFromFork(upstream, branch, base), kind, nil
		}
	}
	return host.NewChange(branch, base), kind, nil
}

// remoteBranchOf is the branch's name on the remote: its upstream's, when
// it has one, without the remote's name.
func remoteBranchOf(ctx context.Context, repoPath, branch string) string {
	branches, err := git.ListBranches(ctx, repoPath)
	if err != nil {
		return branch
	}
	for _, b := range branches {
		if b.Name == branch && b.Upstream != "" {
			if _, rest, ok := strings.Cut(b.Upstream, "/"); ok {
				return rest
			}
		}
	}
	return branch
}

// createLinkIn finds the link a host prints after a push ("To create a
// merge request for feature, visit:", "Create a pull request for 'feature'
// on GitHub by visiting:").
func createLinkIn(output string) string {
	for _, line := range strings.Split(output, "\n") {
		for _, word := range strings.Fields(line) {
			if (strings.HasPrefix(word, "https://") || strings.HasPrefix(word, "http://")) &&
				(strings.Contains(word, "/merge_requests/new") || strings.Contains(word, "/pull/new/") || strings.Contains(word, "/compare/")) {
				return word
			}
		}
	}
	return ""
}

// ForcePush force-pushes the current branch to its configured upstream.
func (s *RemoteService) ForcePush(ctx context.Context, repoPath string, auth *gitexec.Credential) error {
	ctx, err := withAuth(ctx, auth)
	if err != nil {
		return err
	}
	return git.ForcePushUpstream(ctx, repoPath)
}

// PushWarning is what the host forbids on the branch a push would reach,
// for a confirmation before the push rather than a rejection after it.
type PushWarning struct {
	Host           string `json:"host"`
	Branch         string `json:"branch"`
	Pattern        string `json:"pattern"`
	NoPush         bool   `json:"noPush"`
	RestrictedPush bool   `json:"restrictedPush"`
	NoForcePush    bool   `json:"noForcePush"`
	NoDelete       bool   `json:"noDelete"`
}

// PushWarning reads the cache only and is nil when the host is off, has
// shown no rules, or none covers the branch. remoteBranch is "origin/main";
// empty means the current branch's upstream.
func (s *RemoteService) PushWarning(repoPath, remoteBranch string) (*PushWarning, error) {
	ctx := context.Background()
	if remoteBranch == "" {
		status, err := git.CurrentBranchStatus(ctx, repoPath)
		if err != nil || status.Upstream == "" {
			return nil, err
		}
		remoteBranch = status.Upstream
	}
	remotes, err := git.ListRemotes(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	remote, branch, ok := splitRemoteBranch(remotes, remoteBranch)
	if !ok {
		return nil, nil
	}
	return protectionOf(repoPath, remote, branch)
}

// DeleteRemoteBranch takes a remote-tracking branch as the UI shows it, such
// as "origin/feature".
func (s *RemoteService) DeleteRemoteBranch(ctx context.Context, repoPath, remoteBranch string, auth *gitexec.Credential) error {
	remotes, err := git.ListRemotes(ctx, repoPath)
	if err != nil {
		return err
	}
	remote, branch, ok := splitRemoteBranch(remotes, remoteBranch)
	if !ok {
		return &gitexec.AppError{Message: "That branch isn't on a configured remote.", Detail: remoteBranch}
	}
	ctx, err = withAuth(ctx, auth)
	if err != nil {
		return err
	}
	return git.DeleteRemoteBranch(ctx, repoPath, remote, branch)
}

// splitRemoteBranch matches the longest remote name, since remote names can
// contain slashes ("team/origin/feature").
func splitRemoteBranch(remotes []git.Remote, remoteBranch string) (remote, branch string, ok bool) {
	for _, r := range remotes {
		rest, found := strings.CutPrefix(remoteBranch, r.Name+"/")
		if found && rest != "" && len(r.Name) > len(remote) {
			remote, branch, ok = r.Name, rest, true
		}
	}
	return remote, branch, ok
}

// HostLinks are the web links for the repository's host, from origin's URL or
// the first remote's, in the style the user chose for that host. nil when no
// remote points at a hosted project.
func (s *RemoteService) HostLinks(repoPath string) (*hostlink.Links, error) {
	remotes, err := git.ListRemotes(context.Background(), repoPath)
	if err != nil {
		return nil, err
	}
	return hostLinksFor(remotes, configOrDefault().Hosts), nil
}

func hostLinksFor(remotes []git.Remote, hosts map[string]config.HostConfig) *hostlink.Links {
	if len(remotes) == 0 {
		return nil
	}
	host, ok := hostlink.Resolve(context.Background(), remoteURL(primaryRemote(remotes)))
	if !ok {
		return nil
	}
	links := withHostConfig(host, hosts).Links()
	return &links
}

func withHostConfig(host hostlink.Host, hosts map[string]config.HostConfig) hostlink.Host {
	if provider := hosts[host.Hostname()].Provider; provider != "" {
		host.Style = hostlink.Style(provider)
	}
	return host
}

func remoteURL(remote git.Remote) string {
	if remote.FetchURL != "" {
		return remote.FetchURL
	}
	return remote.PushURL
}

// primaryRemote is origin when there is one, else the first remote.
func primaryRemote(remotes []git.Remote) git.Remote {
	for _, r := range remotes {
		if r.Name == "origin" {
			return r
		}
	}
	return remotes[0]
}
