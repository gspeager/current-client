//go:build !nohost

package app

import (
	"context"
	"strconv"
	"strings"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/hostapi"
)

// A change the host reports as merged or closed never changes again, so
// one is asked for once and kept for the session.
const cacheMergedChange = "merged"

// refreshMergedChanges asks the host about the changes of local branches
// that are no longer open: one request per such branch the first time, none
// after. A change is known either from the last refresh, where it was still
// open, or from the heads Git publishes while its branch exists on the
// remote.
func refreshMergedChanges(ctx context.Context, provider hostapi.Provider, info HostInfo, repoPath, remote string, before []hostapi.Change, open []hostapi.Change) {
	branches, err := git.ListBranches(ctx, repoPath)
	if err != nil {
		return
	}
	upstreams := map[string]bool{}
	for _, b := range branches {
		if branch, ok := cutRemote(b.Upstream, remote); ok {
			upstreams[branch] = true
		}
	}
	isOpen := map[int]bool{}
	for _, change := range open {
		isOpen[change.Number] = true
	}
	candidates := map[int]bool{}
	for _, change := range before {
		if upstreams[change.SourceBranch] && change.SourceProject == "" {
			candidates[change.Number] = true
		}
	}
	if cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheRemoteRefs, remote); ok {
		for branch, number := range cached.(git.RemoteChanges).ByBranch {
			if upstreams[branch] {
				candidates[number] = true
			}
		}
	}
	for number := range candidates {
		id := strconv.Itoa(number)
		if _, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheMergedChange, id); ok || isOpen[number] {
			continue
		}
		change, err := provider.Change(ctx, info.Project, number)
		if err != nil || change.State == hostapi.ChangeOpen {
			continue
		}
		hostClient.Cache.Put(info.Hostname, repoPath, cacheMergedChange, id, change)
	}
}

// openChangesByBranch reads the cache only: each branch's open change, such as "!12".
func openChangesByBranch(repoPath string) map[string]string {
	_, info, ok, err := enabledHost(repoPath)
	if err != nil || !ok {
		return nil
	}
	cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheChanges, info.Project)
	if !ok {
		return nil
	}
	open := map[string]string{}
	for _, change := range cached.(cachedChanges).changes {
		if change.SourceProject == "" {
			open[change.SourceBranch] = changePrefix(info) + strconv.Itoa(change.Number)
		}
	}
	return open
}

// mergedOnHost reads the cache only.
func mergedOnHost(repoPath string) map[string]hostMergedChange {
	remote, info, ok, err := enabledHost(repoPath)
	if err != nil || !ok {
		return nil
	}
	merged := map[string]hostMergedChange{}
	for _, value := range hostClient.Cache.Values(info.Hostname, repoPath, cacheMergedChange) {
		change := value.(hostapi.Change)
		if change.State != hostapi.ChangeMerged || change.SourceProject != "" {
			continue
		}
		merged[remote.Name+"/"+change.SourceBranch] = hostMergedChange{Number: change.Number, Prefix: changePrefix(info), HeadSHA: change.HeadSHA}
	}
	return merged
}

// finishedChanges is the cached Closed list, if it was opened, and the
// changes merged from the app.
func finishedChanges(info HostInfo, repoPath string) []hostapi.Change {
	var closed ChangeList
	fillChangeList(&closed, info, repoPath, cacheClosed)
	changes := append([]hostapi.Change{}, closed.Changes...)
	for _, value := range hostClient.Cache.Entries(info.Hostname, repoPath, cacheMergedChange) {
		changes = append(changes, value.(hostapi.Change))
	}
	return changes
}

func cutRemote(upstream, remote string) (string, bool) {
	return strings.CutPrefix(upstream, remote+"/")
}
