//go:build !nohost

package app

import (
	"context"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/hostapi"
)

const cacheRules = "rules"

// refreshBranchRules asks for the host's protection rules, naming the local
// branches' upstreams for a host whose endpoint takes a branch. A failed
// request keeps the last answer.
func refreshBranchRules(ctx context.Context, provider hostapi.Provider, info HostInfo, repoPath, remote string) {
	branches, err := git.ListBranches(ctx, repoPath)
	if err != nil {
		return
	}
	var names []string
	for _, b := range branches {
		if name, ok := cutRemote(b.Upstream, remote); ok {
			names = append(names, name)
		}
	}
	if rules, err := provider.BranchRules(ctx, info.Project, names); err == nil {
		hostClient.Cache.Put(info.Hostname, repoPath, cacheRules, "all", rules)
	}
}

// protectionOf reads the cache only: the rule for a branch on the
// repository's host, when the host is on and a refresh brought rules.
func protectionOf(repoPath, remote, branch string) (*PushWarning, error) {
	primary, info, ok, err := primaryHost(repoPath)
	if err != nil || !ok || !info.Enabled || primary.Name != remote {
		return nil, err
	}
	cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheRules, "all")
	if !ok {
		return nil, nil
	}
	rule, ok := hostapi.MatchRule(cached.([]hostapi.BranchRule), branch)
	if !ok {
		return nil, nil
	}
	return &PushWarning{
		Host: info.Hostname, Branch: branch, Pattern: rule.Pattern,
		NoPush: rule.NoPush, RestrictedPush: rule.RestrictedPush, NoForcePush: rule.NoForcePush, NoDelete: rule.NoDelete,
	}, nil
}
