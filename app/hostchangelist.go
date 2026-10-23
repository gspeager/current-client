//go:build !nohost

package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostcontent"
	"github.com/gspeager/current-client/core/hostlink"
)

// ChangeList is the merge request / pull request panel's data for the
// repository's primary remote, read from the cache. Enabled is false, and
// the rest empty, when the host is off or has no provider; the panel is then
// absent. FetchedAt is nil before the first refresh.
type ChangeList struct {
	Host      string           `json:"host"`
	Provider  string           `json:"provider"`
	Project   string           `json:"project"`
	Enabled   bool             `json:"enabled"`
	Prefix    string           `json:"prefix"`
	Me        string           `json:"me"`
	Changes   []hostapi.Change `json:"changes"`
	Truncated bool             `json:"truncated"`
	FetchedAt *time.Time       `json:"fetchedAt"`
}

type ChangeDetails struct {
	Change          hostapi.Change `json:"change"`
	DescriptionHTML string         `json:"descriptionHtml"`
}

type CheckoutResult struct {
	Branch  string `json:"branch"`
	Existed bool   `json:"existed"`
}

func (s *HostService) Changes(repoPath string) (ChangeList, error) {
	remote, info, ok, err := primaryHost(repoPath)
	if err != nil || !ok || !info.Enabled || providerFor(info) == nil {
		return ChangeList{}, err
	}
	list := ChangeList{Host: info.Hostname, Provider: info.Provider, Project: info.Project, Enabled: true, Prefix: changePrefix(info), Changes: []hostapi.Change{}}
	if cached, at, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheChanges, info.Project); ok {
		entry := cached.(cachedChanges)
		list.Changes, list.Truncated, list.FetchedAt = entry.changes, entry.truncated, &at
	}
	if cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheUser, remote.Name); ok {
		list.Me = cached.(hostapi.User).Username
	}
	return list, nil
}

func (s *HostService) ChangeDetails(repoPath string, number int) (ChangeDetails, error) {
	_, info, change, err := cachedChange(repoPath, number)
	if err != nil {
		return ChangeDetails{}, err
	}
	webRoot := "https://" + info.Hostname + "/"
	return ChangeDetails{Change: change, DescriptionHTML: hostcontent.RenderMarkdown(change.Description, webRoot)}, nil
}

// CheckoutChange fetches the change's head into refs/current-client/ and
// checks out a local branch at it: the source branch's name, prefixed with
// the fork's owner when it comes from one. An existing branch of that name
// is checked out as it is.
func (s *HostService) CheckoutChange(ctx context.Context, repoPath string, number int, auth *gitexec.Credential) (CheckoutResult, error) {
	remote, info, change, err := cachedChange(repoPath, number)
	if err != nil {
		return CheckoutResult{}, err
	}
	ctx, err = withAuth(ctx, auth)
	if err != nil {
		return CheckoutResult{}, err
	}
	name := change.SourceBranch
	if change.SourceProject != "" {
		owner, _, _ := strings.Cut(change.SourceProject, "/")
		name = owner + "/" + change.SourceBranch
	}
	branches, err := git.ListBranches(ctx, repoPath)
	if err != nil {
		return CheckoutResult{}, err
	}
	for _, b := range branches {
		if b.Name == name {
			return CheckoutResult{Branch: name, Existed: true}, git.CheckoutBranch(ctx, repoPath, name)
		}
	}
	if err := git.FetchHostChangeHead(ctx, repoPath, remote.Name, changeHeadRef(info, number), number); err != nil {
		return CheckoutResult{}, err
	}
	if err := git.CreateBranchAt(ctx, repoPath, name, git.HostChangeRef(remote.Name, number)); err != nil {
		return CheckoutResult{}, err
	}
	if err := git.CheckoutBranch(ctx, repoPath, name); err != nil {
		return CheckoutResult{}, err
	}
	if change.SourceProject == "" {
		// The remote-tracking branch may not be fetched yet; pushing later sets it then.
		_ = git.SetUpstream(ctx, repoPath, name, remote.Name+"/"+change.SourceBranch)
	}
	return CheckoutResult{Branch: name}, nil
}

func isGitLab(info HostInfo) bool {
	return hostlink.Style(info.Provider) == hostlink.GitLab
}

// changeHeadRef is where the host publishes a change's head.
func changeHeadRef(info HostInfo, number int) string {
	if isGitLab(info) {
		return fmt.Sprintf("refs/merge-requests/%d/head", number)
	}
	return fmt.Sprintf("refs/pull/%d/head", number)
}

func changePrefix(info HostInfo) string {
	if isGitLab(info) {
		return "!"
	}
	return "#"
}

func changeKind(info HostInfo) string {
	if isGitLab(info) {
		return "merge request"
	}
	return "pull request"
}

func primaryHost(repoPath string) (git.Remote, HostInfo, bool, error) {
	remotes, err := git.ListRemotes(context.Background(), repoPath)
	if err != nil || len(remotes) == 0 {
		return git.Remote{}, HostInfo{}, false, err
	}
	remote := primaryRemote(remotes)
	info, ok := hostInfoFor(remote, configOrDefault().Hosts)
	return remote, info, ok, nil
}

func cachedChange(repoPath string, number int) (git.Remote, HostInfo, hostapi.Change, error) {
	remote, info, ok, err := primaryHost(repoPath)
	if err != nil {
		return remote, info, hostapi.Change{}, err
	}
	if !ok || !info.Enabled {
		return remote, info, hostapi.Change{}, errors.New("integration is off for this repository's host")
	}
	cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheChanges, info.Project)
	if ok {
		for _, change := range cached.(cachedChanges).changes {
			if change.Number == number {
				return remote, info, change, nil
			}
		}
	}
	return remote, info, hostapi.Change{}, fmt.Errorf("%s%d isn't in the last refresh; refresh and try again", changePrefix(info), number)
}

// deleteFetchedHeads removes refs/current-client/ heads for the host's
// remotes in every repository the app knows, as the guide promises when a
// host is turned off.
func deleteFetchedHeads(hostname string) {
	cfg := configOrDefault()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	seen := map[string]bool{}
	for _, repoPath := range append(append([]string(nil), cfg.OpenTabs...), cfg.RecentRepositories...) {
		if seen[repoPath] {
			continue
		}
		seen[repoPath] = true
		remotes, err := git.ListRemotes(ctx, repoPath)
		if err != nil {
			continue
		}
		for _, remote := range remotes {
			if info, ok := hostInfoFor(remote, cfg.Hosts); ok && info.Hostname == hostname {
				_ = git.DeleteHostRefs(ctx, repoPath, remote.Name)
			}
		}
	}
}
