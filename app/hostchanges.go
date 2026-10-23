//go:build !nohost

package app

import (
	"context"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/hostapi"
)

// BranchChange is what a branch row or graph ref shows for its merge
// request / pull request. State is empty when only the number is known,
// from Git alone, with the host's API off.
type BranchChange struct {
	Number            int    `json:"number"`
	Prefix            string `json:"prefix"`
	State             string `json:"state"`
	Draft             bool   `json:"draft"`
	CI                string `json:"ci"`
	Title             string `json:"title"`
	Author            string `json:"author"`
	WebURL            string `json:"webUrl"`
	SourceBranch      string `json:"sourceBranch"`
	TargetBranch      string `json:"targetBranch"`
	UnresolvedThreads int    `json:"unresolvedThreads"`
	UpdatedAt         string `json:"updatedAt"`
	HeadSHA           string `json:"headSha"`
}

const (
	cacheRemoteRefs = "ls-remote"
	cacheChanges    = "changes"
	cacheUser       = "user"
)

type cachedChanges struct {
	changes   []hostapi.Change
	truncated bool
}

// BranchChanges maps each remote branch ("origin/feature") to its change,
// reading only what the last fetch and refresh left in the cache: drawing
// the sidebar or the graph never makes a request.
func (s *HostService) BranchChanges(repoPath string) (map[string]BranchChange, error) {
	remotes, err := git.ListRemotes(context.Background(), repoPath)
	if err != nil {
		return nil, err
	}
	hosts := configOrDefault().Hosts
	changes := map[string]BranchChange{}
	for _, remote := range remotes {
		info, _ := hostInfoFor(remote, hosts)
		prefix := changePrefix(info)
		if cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheRemoteRefs, remote.Name); ok {
			refs := cached.(git.RemoteChanges)
			if refs.Kind == git.MergeRequestRefs {
				prefix = "!"
			}
			for branch, number := range refs.ByBranch {
				changes[remote.Name+"/"+branch] = BranchChange{Number: number, Prefix: prefix}
			}
		}
		if cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheChanges, info.Project); ok {
			for _, change := range cached.(cachedChanges).changes {
				if change.SourceProject != "" && change.SourceProject != info.Project {
					continue
				}
				changes[remote.Name+"/"+change.SourceBranch] = BranchChange{
					Number:            change.Number,
					Prefix:            prefix,
					State:             string(change.State),
					Draft:             change.Draft,
					CI:                string(change.CI),
					Title:             change.Title,
					Author:            change.Author.Name,
					WebURL:            change.WebURL,
					SourceBranch:      change.SourceBranch,
					TargetBranch:      change.TargetBranch,
					UnresolvedThreads: change.UnresolvedThreads,
					UpdatedAt:         change.UpdatedAt.UTC().Format(time.RFC3339),
					HeadSHA:           change.HeadSHA,
				}
			}
		}
	}
	return changes, nil
}

// Refresh is what a fetch or pull does for the host side: one ls-remote per
// remote for the change numbers Git publishes, and the open changes from
// each host that is on. A failed host request is returned so the panel can
// show it; the last answer stays in the cache.
func (s *HostService) Refresh(repoPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return refreshHostChanges(ctx, repoPath)
}

func refreshHostChanges(ctx context.Context, repoPath string) error {
	remotes, err := git.ListRemotes(ctx, repoPath)
	if err != nil {
		return err
	}
	hosts := configOrDefault().Hosts
	for _, remote := range remotes {
		info, _ := hostInfoFor(remote, hosts)
		if refs, err := git.ListRemoteChanges(ctx, repoPath, remote.Name); err == nil {
			hostClient.Cache.Put(info.Hostname, repoPath, cacheRemoteRefs, remote.Name, refs)
		}
		if !info.Enabled {
			continue
		}
		provider := providerFor(info)
		if provider == nil {
			continue
		}
		before := snapshotCurrentBranch(info, repoPath)
		// Pipelines are asked for again after a refresh; the heads of open
		// changes first, since the list endpoint carries no CI result.
		hostClient.Cache.ClearKind(info.Hostname, repoPath, cachePipeline)
		hostClient.Cache.ClearKind(info.Hostname, repoPath, cacheJobs)
		hostClient.Cache.ClearKind(info.Hostname, repoPath, cacheIssue)
		hostClient.Cache.ClearKind(info.Hostname, repoPath, cacheThreads)
		var lastOpen []hostapi.Change
		if cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheChanges, info.Project); ok {
			lastOpen = cached.(cachedChanges).changes
		}
		if changes, truncated, err := provider.OpenChanges(ctx, info.Project); err == nil {
			fillChangeCI(ctx, provider, info, repoPath, changes)
			refreshMergedChanges(ctx, provider, info, repoPath, remote.Name, lastOpen, changes)
			fillCurrentBranchThreads(ctx, provider, info, repoPath, remote.Name, changes)
			hostClient.Cache.Put(info.Hostname, repoPath, cacheChanges, info.Project, cachedChanges{changes, truncated})
			fetchCurrentBranchJobs(ctx, provider, info, repoPath, remote.Name, changes)
			refreshBranchRules(ctx, provider, info, repoPath, remote.Name)
			refreshForecast(ctx, info, repoPath, remote.Name, changes)
			if notices := noticesSince(before, info, repoPath); len(notices) > 0 {
				hostClient.Cache.Put(info.Hostname, repoPath, cacheNotices, "current", notices)
			}
		} else {
			return err
		}
		if _, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheUser, remote.Name); !ok {
			if user, err := provider.CurrentUser(ctx); err == nil {
				hostClient.Cache.Put(info.Hostname, repoPath, cacheUser, remote.Name, user)
			}
		}
	}
	return nil
}
