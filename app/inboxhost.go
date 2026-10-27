//go:build !nohost

package app

import (
	"context"
	"strconv"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/hostapi"
)

// hostInboxRows reads the cache only: review requests waiting on the user,
// their own changes with changes requested or failing CI, the issues
// assigned to them and the host's to-dos about this project, each with the
// age of the refresh that brought it. notice is what kept the to-dos from
// being listed, if anything.
func hostInboxRows(repoPath string) (rows []InboxRow, enabled bool, fetchedAt *time.Time, notice string) {
	list, info, _, err := changeList(repoPath)
	if err != nil || !list.Enabled {
		return trackerInboxRows(repoPath), false, nil, ""
	}
	fillChangeList(&list, info, repoPath, cacheChanges)
	for _, change := range list.Changes {
		row := InboxRow{Number: change.Number, Prefix: list.Prefix, Branch: change.SourceBranch, Title: change.Title, WebURL: change.WebURL, FetchedAt: list.FetchedAt}
		number := strconv.Itoa(change.Number)
		if list.Me != "" && change.Author.Username == list.Me {
			if hasReview(change.Reviews, hostapi.ReviewChangesRequested) {
				row.ID, row.Kind, row.Detail = "changes-requested:"+number, "changes-requested", reviewerNames(change.Reviews, hostapi.ReviewChangesRequested)
				rows = append(rows, row)
			}
			if change.CI == hostapi.CIFailed {
				row.ID, row.Kind, row.Detail = "ci-failed:"+number, "ci-failed", ""
				rows = append(rows, row)
			}
			continue
		}
		for _, r := range change.Reviews {
			if r.State == hostapi.ReviewRequested && r.Reviewer.Username == list.Me && list.Me != "" {
				row.ID, row.Kind = "review:"+number, "review"
				rows = append(rows, row)
				break
			}
		}
	}
	if cached, at, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheAssigned, info.Project); ok {
		fetched := at
		for _, issue := range cached.([]hostapi.Issue) {
			rows = append(rows, InboxRow{ID: "assigned:" + issue.Key, Kind: "assigned", Key: issue.Key, Title: issue.Title, Detail: issueBranchState(repoPath, info, list, issue.Key), WebURL: issue.WebURL, FetchedAt: &fetched})
		}
	}
	rows, notice = todoRows(rows, info, list.Prefix)
	return append(rows, trackerInboxRows(repoPath)...), true, list.FetchedAt, notice
}

// issueBranchState says what a local branch named after the issue is
// waiting for: a review from the user or a failed pipeline on its change.
func issueBranchState(repoPath string, info HostInfo, list ChangeList, key string) string {
	tracker := issueTrackerFor(info)
	if tracker == nil {
		return ""
	}
	branches, err := git.ListBranches(context.Background(), repoPath)
	if err != nil {
		return ""
	}
	for _, b := range branches {
		if tracker.KeyInBranch(b.Name) != key {
			continue
		}
		for _, change := range list.Changes {
			if change.SourceBranch != b.Name {
				continue
			}
			switch {
			case change.CI == hostapi.CIFailed:
				return list.Prefix + strconv.Itoa(change.Number) + " failing CI"
			case hasReview(change.Reviews, hostapi.ReviewRequested):
				return list.Prefix + strconv.Itoa(change.Number) + " review requested"
			}
		}
	}
	return ""
}

func hasReview(reviews []hostapi.Review, state hostapi.ReviewState) bool {
	for _, r := range reviews {
		if r.State == state {
			return true
		}
	}
	return false
}

func reviewerNames(reviews []hostapi.Review, state hostapi.ReviewState) string {
	names := ""
	for _, r := range reviews {
		if r.State != state {
			continue
		}
		if names != "" {
			names += ", "
		}
		names += r.Reviewer.Name
	}
	return names
}

// refreshAssignedIssues keeps the Inbox's assigned issues current: one
// request per refresh, the same page From an issue reads.
func refreshAssignedIssues(ctx context.Context, provider hostapi.Provider, info HostInfo, repoPath, remote string) {
	tracker, ok := provider.(hostapi.IssueTracker)
	if !ok {
		return
	}
	me, err := currentUser(ctx, provider, info, repoPath, remote)
	if err != nil {
		return
	}
	issues, err := tracker.AssignedIssues(ctx, info.Project, me.Username)
	if err != nil {
		return
	}
	hostClient.Cache.Put(info.Hostname, repoPath, cacheAssigned, info.Project, issues)
	rememberIssueTitles(info, repoPath, issues)
}
