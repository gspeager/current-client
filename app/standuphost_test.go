//go:build !nohost

package app

import (
	"strconv"
	"testing"
	"time"

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostapi/hostapitest"
)

func TestStandupHostRowsComeFromTheCacheAlone(t *testing.T) {
	repo := enabledGitLab(t)
	swapHostClient(t, hostapitest.Forbid(t))
	remote, info, _, err := primaryHost(repo)
	if err != nil {
		t.Fatal(err)
	}
	me := hostapi.User{Username: "ada"}
	bob := hostapi.User{Username: "bob"}
	since := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	recent, old := since.Add(time.Hour), since.Add(-time.Hour)
	open := []hostapi.Change{
		{Number: 1, Title: "Mine", Author: me, State: hostapi.ChangeOpen, UpdatedAt: recent},
		{Number: 2, Title: "Mine, untouched", Author: me, State: hostapi.ChangeOpen, UpdatedAt: old},
		{Number: 3, Title: "Bob's, approved", Author: bob, State: hostapi.ChangeOpen, UpdatedAt: recent, Reviews: []hostapi.Review{{Reviewer: me, State: hostapi.ReviewApproved}}},
		{Number: 4, Title: "Bob's, asked", Author: bob, State: hostapi.ChangeOpen, UpdatedAt: recent, Reviews: []hostapi.Review{{Reviewer: me, State: hostapi.ReviewRequested}}},
	}
	closed := []hostapi.Change{{Number: 5, Title: "Mine, merged", Author: me, State: hostapi.ChangeMerged, UpdatedAt: recent}}
	hostClient.Cache.Put(info.Hostname, repo, cacheUser, remote.Name, me)
	hostClient.Cache.Put(info.Hostname, repo, cacheChanges, info.Project, cachedChanges{open, false})
	hostClient.Cache.Put(info.Hostname, repo, cacheClosed, info.Project, cachedChanges{closed, false})
	merged := hostapi.Change{Number: 6, Title: "Merged here", Author: me, State: hostapi.ChangeMerged, UpdatedAt: recent.Add(time.Hour)}
	hostClient.Cache.Put(info.Hostname, repo, cacheMergedChange, strconv.Itoa(merged.Number), merged)

	var inbox InboxService
	repos, err := inbox.Standup([]string{repo}, since.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	r := repos[0]
	if !r.HostEnabled || r.FetchedAt == nil || r.Prefix != "!" {
		t.Fatalf("host = %+v", r)
	}
	if len(r.Opened) != 1 || r.Opened[0].Number != 1 {
		t.Errorf("opened = %+v", r.Opened)
	}
	if len(r.Merged) != 2 || r.Merged[0].Number != 6 || r.Merged[1].Number != 5 {
		t.Errorf("merged = %+v", r.Merged)
	}
	if len(r.Reviewed) != 1 || r.Reviewed[0].Number != 3 || r.Reviewed[0].Review != string(hostapi.ReviewApproved) {
		t.Errorf("reviewed = %+v", r.Reviewed)
	}
}
