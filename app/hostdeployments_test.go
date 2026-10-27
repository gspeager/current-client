//go:build !nohost

package app

import (
	"strings"
	"testing"
	"time"

	"github.com/gspeager/current-client/core/hostapi"
)

func TestReachFollowsTheDeployedCommit(t *testing.T) {
	repo := enabledGitLab(t)
	commit := func(message string) string {
		gitIn(t, repo, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", message)
		return strings.TrimSpace(gitOut(t, repo, "rev-parse", "HEAD"))
	}
	first := commit("first")
	second := commit("second")
	api := "/api/v4/projects/group%2Fapp"
	deployments := func(sha string) string {
		return `[{"sha": "` + sha + `", "ref": "main", "status": "success", "created_at": "2026-10-26T10:00:00Z", "environment": {"id": 1, "name": "production", "tier": "production"}},
			{"sha": "0000000000000000000000000000000000000abc", "ref": "main", "status": "success", "environment": {"id": 2, "name": "staging", "tier": "staging"}}]`
	}
	transport := &routeTransport{routes: map[string]string{
		api + "/merge_requests?per_page=100&state=opened":       `[]`,
		api + "/deployments?order_by=id&per_page=100&sort=desc": deployments(first),
		"/api/v4/user": `{"username": "me"}`,
	}, hits: map[string]int{}}
	swapHostClient(t, transport)
	var s HostService

	s.Refresh(repo)
	if n := transport.hits[api+"/deployments?order_by=id&per_page=100&sort=desc"]; n != 1 {
		t.Fatalf("deployments asked %d times on a refresh, want once", n)
	}
	list, _ := s.Deployments(repo)
	if !list.Enabled || list.FetchedAt == nil || len(list.Environments) != 2 || !list.Environments[0].Fetched || list.Environments[1].Fetched {
		t.Fatalf("Deployments = %+v", list)
	}
	reach, err := s.Reach(repo, []string{first, second})
	if err != nil {
		t.Fatal(err)
	}
	if want := []Reach{{"production", "in", true}, {"staging", "unknown", false}}; !equalReach(reach[first], want) {
		t.Fatalf("first = %+v", reach[first])
	}
	if reach[second][0].State != "out" {
		t.Fatalf("a commit after the deployed one: %+v", reach[second])
	}

	transport.routes[api+"/deployments?order_by=id&per_page=100&sort=desc"] = deployments(second)
	s.Refresh(repo)
	if reach, _ := s.Reach(repo, []string{second}); reach[second][0].State != "in" {
		t.Fatalf("once deployed past it: %+v", reach[second])
	}
	if n := transport.hits["/api/v4/projects/group%2Fapp/merge_requests?per_page=100&state=opened"]; n != 2 {
		t.Fatalf("refreshes = %d", n)
	}
}

func equalReach(a, b []Reach) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestMergedWorkNotOnProductionYet(t *testing.T) {
	repo := enabledGitLab(t)
	commit := func(message string) string {
		gitIn(t, repo, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", message)
		return strings.TrimSpace(gitOut(t, repo, "rev-parse", "HEAD"))
	}
	deployed := commit("deployed")
	waiting := commit("merged after")
	remote, info, _, _ := primaryHost(repo)
	me := hostapi.User{Username: "me"}
	recent := time.Now().Add(-time.Hour)
	hostClient.Cache.Put(info.Hostname, repo, cacheUser, remote.Name, me)
	hostClient.Cache.Put(info.Hostname, repo, cacheChanges, info.Project, cachedChanges{nil, false})
	hostClient.Cache.Put(info.Hostname, repo, cacheClosed, info.Project, cachedChanges{[]hostapi.Change{
		{Number: 7, Title: "Deployed one", Author: me, State: hostapi.ChangeMerged, MergeCommitSHA: deployed, UpdatedAt: recent},
		{Number: 8, Title: "Waiting one", Author: me, State: hostapi.ChangeMerged, MergeCommitSHA: waiting, UpdatedAt: recent},
	}, false})
	hostClient.Cache.Put(info.Hostname, repo, cacheDeployments, info.Project, []hostapi.Deployment{{Environment: "production", SHA: deployed, Production: true}})

	var inbox InboxService
	var undeployed []InboxRow
	for _, r := range inbox.Rows([]string{repo}, false)[0].Rows {
		if r.Kind == "undeployed" {
			undeployed = append(undeployed, r)
		}
	}
	if len(undeployed) != 1 || undeployed[0].Number != 8 || undeployed[0].Detail != "production" {
		t.Fatalf("undeployed rows = %+v", undeployed)
	}

	var s HostService
	details, err := s.ChangeDetails(repo, 8)
	if err != nil || len(details.Environments) != 1 || details.Environments[0].State != "out" {
		t.Fatalf("ChangeDetails(8) = %+v, %v", details.Environments, err)
	}
	if details, _ := s.ChangeDetails(repo, 7); details.Environments[0].State != "in" {
		t.Fatalf("ChangeDetails(7) = %+v", details.Environments)
	}
}
