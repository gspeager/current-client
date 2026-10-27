//go:build !nohost

package app

import (
	"sort"
	"time"

	"github.com/gspeager/current-client/core/hostapi"
)

// standupHost reads the cache only: the open list a refresh brought, the
// Closed list if it was opened, and changes merged from the app. Opened is
// the user's open changes updated since; the host gives no time a review
// was left, so Reviewed is the changes updated since on which the user's
// review stands.
func standupHost(repo *StandupRepo, since time.Time) {
	list, info, _, err := changeList(repo.Path)
	if err != nil || !list.Enabled {
		return
	}
	repo.HostEnabled, repo.Prefix = true, list.Prefix
	fillChangeList(&list, info, repo.Path, cacheChanges)
	repo.FetchedAt = list.FetchedAt
	if list.Me == "" {
		return
	}
	closed := list
	closed.Changes = nil
	fillChangeList(&closed, info, repo.Path, cacheClosed)
	changes := append(append([]hostapi.Change{}, list.Changes...), closed.Changes...)
	for _, value := range hostClient.Cache.Entries(info.Hostname, repo.Path, cacheMergedChange) {
		changes = append(changes, value.(hostapi.Change))
	}
	seen := map[int]bool{}
	for _, change := range changes {
		if seen[change.Number] || change.UpdatedAt.Before(since) {
			continue
		}
		seen[change.Number] = true
		item := StandupChange{Number: change.Number, Title: change.Title, WebURL: change.WebURL, UpdatedAt: change.UpdatedAt}
		if change.Author.Username == list.Me {
			switch change.State {
			case hostapi.ChangeOpen:
				repo.Opened = append(repo.Opened, item)
			case hostapi.ChangeMerged:
				repo.Merged = append(repo.Merged, item)
			}
			continue
		}
		for _, r := range change.Reviews {
			if r.Reviewer.Username == list.Me && r.State != hostapi.ReviewRequested {
				item.Review = string(r.State)
				repo.Reviewed = append(repo.Reviewed, item)
			}
		}
	}
	newestFirst := func(changes []StandupChange) {
		sort.SliceStable(changes, func(i, j int) bool { return changes[i].UpdatedAt.After(changes[j].UpdatedAt) })
	}
	newestFirst(repo.Opened)
	newestFirst(repo.Merged)
}
