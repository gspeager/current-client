//go:build !nohost

package app

import (
	"context"

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostapi/jira"
	"github.com/gspeager/current-client/core/hostlink"
)

// trackerAssignedID is the cache id of a tracker's assigned issues for a
// repository; the cache host is the tracker's.
const trackerAssignedID = "tracker"

// jiraFor is the repository's Jira site while its integration is on.
func jiraFor(repoPath string) (*jira.Provider, hostlink.Tracker, bool) {
	tracker, _, ok := trackerFor(repoPath)
	if !ok || !hostEnabled(hostlink.SiteHost(tracker.Site)) {
		return nil, hostlink.Tracker{}, false
	}
	return jira.New(hostClient, tracker), tracker, true
}

// trackerAssigned is the picker's list from the tracker, asked the first
// time and kept until a refresh, each issue kept for hover cards too.
func trackerAssigned(repoPath string, provider *jira.Provider, site hostlink.Tracker) ([]hostapi.Issue, error) {
	name := hostlink.SiteHost(site.Site)
	return cachedFrom(name, repoPath, cacheAssigned, trackerAssignedID, func(ctx context.Context) ([]hostapi.Issue, error) {
		issues, err := provider.AssignedIssues(ctx, "", "")
		if err == nil {
			trackerSource(provider, site).remember(repoPath, issues)
		}
		return issues, err
	})
}

// TrackerIssueTitles is what the last fetch learned about each tracker key, by key.
func (s *HostService) TrackerIssueTitles(repoPath string, keys []string) map[string]IssueInfo {
	_, site, ok := jiraFor(repoPath)
	if !ok {
		return map[string]IssueInfo{}
	}
	return issueSource{cacheHost: hostlink.SiteHost(site.Site)}.titles(repoPath, keys)
}

// FetchTrackerIssueTitles asks the tracker for the keys not yet known, as
// issueSource.fetchTitles does.
func (s *HostService) FetchTrackerIssueTitles(repoPath string, keys []string) map[string]IssueInfo {
	provider, site, ok := jiraFor(repoPath)
	if !ok {
		return map[string]IssueInfo{}
	}
	return trackerSource(provider, site).fetchTitles(repoPath, keys)
}

func trackerSource(provider *jira.Provider, site hostlink.Tracker) issueSource {
	return issueSource{cacheHost: hostlink.SiteHost(site.Site), webRoot: site.Site, tracker: provider}
}

// refreshTracker is the tracker's share of a refresh: what was cached is
// asked again, and the assigned issues are read once for the Inbox.
func refreshTracker(ctx context.Context, repoPath string) {
	provider, site, ok := jiraFor(repoPath)
	if !ok {
		return
	}
	name := hostlink.SiteHost(site.Site)
	clearKinds(name, repoPath, cacheIssue, cacheIssueDetails, cacheIssueComments, cacheAssigned, cacheProjectIssues)
	if issues, err := provider.AssignedIssues(ctx, "", ""); err == nil {
		cacheAssigned.put(name, repoPath, trackerAssignedID, issues)
		trackerSource(provider, site).remember(repoPath, issues)
	}
}

// trackerInboxRows is the tracker's issues assigned to the user, as of the
// refresh that brought them.
func trackerInboxRows(repoPath string) []InboxRow {
	_, site, ok := jiraFor(repoPath)
	if !ok {
		return nil
	}
	cached, at, ok := cacheAssigned.get(hostlink.SiteHost(site.Site), repoPath, trackerAssignedID)
	if !ok {
		return nil
	}
	var rows []InboxRow
	for _, issue := range cached {
		fetched := at
		rows = append(rows, InboxRow{ID: "assigned:" + issue.Key, Kind: "assigned", Key: issue.Key, Title: issue.Title, WebURL: issue.WebURL, FetchedAt: &fetched})
	}
	return rows
}
