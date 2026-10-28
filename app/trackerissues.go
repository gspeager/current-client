//go:build !nohost

package app

import (
	"context"
	"errors"
	"time"

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
	if !ok || !hostEnabled(jira.HostName(tracker)) {
		return nil, hostlink.Tracker{}, false
	}
	return jira.New(hostClient, tracker), tracker, true
}

// trackerAssigned is the picker's list from the tracker, asked the first
// time and kept until a refresh, each issue kept for hover cards too.
func trackerAssigned(repoPath string, provider *jira.Provider, site hostlink.Tracker) ([]hostapi.Issue, error) {
	name := jira.HostName(site)
	return cachedFrom(name, repoPath, cacheAssigned, trackerAssignedID, func(ctx context.Context) ([]hostapi.Issue, error) {
		issues, err := provider.AssignedIssues(ctx, "", "")
		if err == nil {
			rememberTrackerIssues(name, repoPath, issues)
		}
		return issues, err
	})
}

func rememberTrackerIssues(cacheHost, repoPath string, issues []hostapi.Issue) {
	for _, is := range issues {
		hostClient.Cache.Put(cacheHost, repoPath, cacheIssue, is.Key, IssueInfo{Key: is.Key, Title: is.Title, State: is.State, WebURL: is.WebURL, Known: true})
	}
}

// TrackerIssueTitles reads the cache only: what the last fetch learned
// about each tracker key, keyed by key.
func (s *HostService) TrackerIssueTitles(repoPath string, keys []string) map[string]IssueInfo {
	_, site, ok := jiraFor(repoPath)
	if !ok {
		return map[string]IssueInfo{}
	}
	return cachedTrackerIssues(jira.HostName(site), repoPath, keys)
}

// FetchTrackerIssueTitles asks the tracker for the keys not yet known, up to
// the batch cap, four at a time, as FetchIssueTitles does for a host; a key
// the tracker hasn't is remembered as known with no title.
func (s *HostService) FetchTrackerIssueTitles(repoPath string, keys []string) map[string]IssueInfo {
	provider, site, ok := jiraFor(repoPath)
	if !ok {
		return map[string]IssueInfo{}
	}
	name := jira.HostName(site)
	var missing []string
	seen := map[string]bool{}
	for _, key := range keys {
		if seen[key] {
			continue
		}
		seen[key] = true
		if _, _, ok := hostClient.Cache.Get(name, repoPath, cacheIssue, key); !ok {
			missing = append(missing, key)
		}
	}
	if len(missing) > issueBatchCap {
		missing = missing[:issueBatchCap]
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	inParallel(len(missing), func(i int) {
		key := missing[i]
		issue, err := provider.IssueByKey(ctx, "", key)
		var hostErr *hostapi.HostError
		switch {
		case err == nil:
			rememberTrackerIssues(name, repoPath, []hostapi.Issue{issue})
		case errors.As(err, &hostErr) && hostErr.Status == 404:
			hostClient.Cache.Put(name, repoPath, cacheIssue, key, IssueInfo{Key: key, Known: true})
		}
	})
	return cachedTrackerIssues(name, repoPath, keys)
}

func cachedTrackerIssues(cacheHost, repoPath string, keys []string) map[string]IssueInfo {
	issues := map[string]IssueInfo{}
	for _, key := range keys {
		if cached, _, ok := hostClient.Cache.Get(cacheHost, repoPath, cacheIssue, key); ok {
			issues[key] = cached.(IssueInfo)
		}
	}
	return issues
}

// refreshTracker is the tracker's share of a refresh: what was cached is
// asked again, and the assigned issues are read once for the Inbox.
func refreshTracker(ctx context.Context, repoPath string) {
	provider, site, ok := jiraFor(repoPath)
	if !ok {
		return
	}
	name := jira.HostName(site)
	for _, kind := range []string{cacheIssue, cacheIssueDetails, cacheIssueComments, cacheAssigned, cacheProjectIssues} {
		hostClient.Cache.ClearKind(name, repoPath, kind)
	}
	if issues, err := provider.AssignedIssues(ctx, "", ""); err == nil {
		hostClient.Cache.Put(name, repoPath, cacheAssigned, trackerAssignedID, issues)
		rememberTrackerIssues(name, repoPath, issues)
	}
}

// trackerInboxRows reads the cache only: the tracker's issues assigned to
// the user, as of the refresh that brought them.
func trackerInboxRows(repoPath string) []InboxRow {
	_, site, ok := jiraFor(repoPath)
	if !ok {
		return nil
	}
	cached, at, ok := hostClient.Cache.Get(jira.HostName(site), repoPath, cacheAssigned, trackerAssignedID)
	if !ok {
		return nil
	}
	var rows []InboxRow
	for _, issue := range cached.([]hostapi.Issue) {
		fetched := at
		rows = append(rows, InboxRow{ID: "assigned:" + issue.Key, Kind: "assigned", Key: issue.Key, Title: issue.Title, WebURL: issue.WebURL, FetchedAt: &fetched})
	}
	return rows
}
