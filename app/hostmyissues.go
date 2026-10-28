//go:build !nohost

package app

import (
	"context"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostapi/jira"
)

// IssueList is the Issues tab's Mine list: the issues assigned to the user
// on the repository's host and on its tracker, from the last refresh.
type IssueList struct {
	// Enabled says a host or tracker that lists issues is on.
	Enabled bool              `json:"enabled"`
	Sources []IssueListSource `json:"sources"`
	Issues  []ListedIssue     `json:"issues"`
}

// IssueListSource is one service the list reads, with when it was last
// refreshed; FetchedAt is nil until a refresh has read it.
type IssueListSource struct {
	Name      string     `json:"name"`
	FetchedAt *time.Time `json:"fetchedAt"`
}

type ListedIssue struct {
	Issue  hostapi.Issue `json:"issue"`
	Source string        `json:"source"`
	// Branch is a local branch named after the issue, if there is one.
	Branch string `json:"branch"`
}

// MyIssues reads the cache only; Refresh fills it, so opening the tab asks
// nothing until the user, or the tab's first open, refreshes.
func (s *HostService) MyIssues(repoPath string) IssueList {
	var list IssueList
	branches, _ := git.ListBranches(context.Background(), repoPath)
	add := func(name string, tracker hostapi.IssueTracker, cached any, at time.Time, ok bool) {
		list.Enabled = true
		source := IssueListSource{Name: name}
		if ok {
			source.FetchedAt = &at
			for _, issue := range cached.([]hostapi.Issue) {
				list.Issues = append(list.Issues, ListedIssue{Issue: issue, Source: name, Branch: branchNamedAfter(branches, tracker, issue.Key)})
			}
		}
		list.Sources = append(list.Sources, source)
	}
	if _, info, ok, err := primaryHost(repoPath); err == nil && ok && info.Enabled {
		if tracker := issueTrackerFor(info); tracker != nil {
			cached, at, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheAssigned, info.Project)
			add(info.Hostname, tracker, cached, at, ok)
		}
	}
	if provider, site, ok := jiraFor(repoPath); ok {
		name := jira.HostName(site)
		cached, at, ok := hostClient.Cache.Get(name, repoPath, cacheAssigned, trackerAssignedID)
		add(name, provider, cached, at, ok)
	}
	return list
}

func branchNamedAfter(branches []git.Branch, tracker hostapi.IssueTracker, key string) string {
	for _, b := range branches {
		if tracker.KeyInBranch(b.Name) == key {
			return b.Name
		}
	}
	return ""
}
