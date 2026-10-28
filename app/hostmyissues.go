//go:build !nohost

package app

import (
	"context"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostapi/jira"
)

// cacheProjectIssues is the project's open or closed issues, keyed "open"
// or "closed", asked when that filter is chosen and kept until a refresh.
const cacheProjectIssues = "projectIssues"

// IssueList is the Issues tab's list: the issues on the repository's host
// and on its tracker.
type IssueList struct {
	// Enabled says a host or tracker that lists issues is on.
	Enabled bool              `json:"enabled"`
	Sources []IssueListSource `json:"sources"`
	Issues  []ListedIssue     `json:"issues"`
}

// IssueListSource is one service the list reads, with when its part was
// read; FetchedAt is nil until it has been.
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

// issueService is one place the repository's issues live, while it is on.
type issueService struct {
	name, project, assignedID string
	tracker                   hostapi.IssueTracker
}

func issueServices(repoPath string) []issueService {
	var services []issueService
	if _, info, ok, err := primaryHost(repoPath); err == nil && ok && info.Enabled {
		if tracker := issueTrackerFor(info); tracker != nil {
			services = append(services, issueService{info.Hostname, info.Project, info.Project, tracker})
		}
	}
	if provider, site, ok := jiraFor(repoPath); ok {
		services = append(services, issueService{jira.HostName(site), "", trackerAssignedID, provider})
	}
	return services
}

// MyIssues reads the cache only; Refresh fills it, so opening the tab asks
// nothing until the user, or the tab's first open, refreshes.
func (s *HostService) MyIssues(repoPath string) IssueList {
	return listIssues(repoPath, func(svc issueService) ([]hostapi.Issue, *time.Time, error) {
		cached, at, ok := hostClient.Cache.Get(svc.name, repoPath, cacheAssigned, svc.assignedID)
		if !ok {
			return nil, nil, nil
		}
		return cached.([]hostapi.Issue), &at, nil
	})
}

// ProjectIssues is the Issues tab's All or Closed: each service's page of
// the project's issues in that state, asked the first time the filter is
// chosen and kept until a refresh. A service that can't list a project's
// issues is left out.
func (s *HostService) ProjectIssues(repoPath string, open bool) (IssueList, error) {
	state := "closed"
	if open {
		state = "open"
	}
	var failed error
	list := listIssues(repoPath, func(svc issueService) ([]hostapi.Issue, *time.Time, error) {
		lister, ok := svc.tracker.(hostapi.IssueLister)
		if !ok {
			return nil, nil, nil
		}
		issues, err := cachedFrom(svc.name, repoPath, cacheProjectIssues, state, func(ctx context.Context) ([]hostapi.Issue, error) {
			return lister.ProjectIssues(ctx, svc.project, open)
		})
		if err != nil {
			failed = err
			return nil, nil, err
		}
		_, at, _ := hostClient.Cache.Get(svc.name, repoPath, cacheProjectIssues, state)
		return issues, &at, nil
	})
	return list, failed
}

func listIssues(repoPath string, read func(issueService) ([]hostapi.Issue, *time.Time, error)) IssueList {
	var list IssueList
	branches, _ := git.ListBranches(context.Background(), repoPath)
	for _, svc := range issueServices(repoPath) {
		list.Enabled = true
		issues, at, _ := read(svc)
		list.Sources = append(list.Sources, IssueListSource{Name: svc.name, FetchedAt: at})
		for _, issue := range issues {
			list.Issues = append(list.Issues, ListedIssue{Issue: issue, Source: svc.name, Branch: branchNamedAfter(branches, svc.tracker, issue.Key)})
		}
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
