//go:build !nohost

package app

import (
	"testing"
	"time"

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostapi/hostapitest"
)

func TestInboxHostRowsComeFromTheCacheAlone(t *testing.T) {
	repo := enabledGitLab(t)
	gitIn(t, repo, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "first")
	gitIn(t, repo, "branch", "-q", "12-fix-login")
	swapHostClient(t, hostapitest.Forbid(t))
	remote, info, _, err := primaryHost(repo)
	if err != nil {
		t.Fatal(err)
	}
	me := hostapi.User{Name: "Ada", Username: "ada"}
	bob := hostapi.User{Name: "Bob", Username: "bob"}
	changes := []hostapi.Change{
		{Number: 1, Title: "Bob's change", SourceBranch: "bobs", Author: bob, Reviews: []hostapi.Review{{Reviewer: me, State: hostapi.ReviewRequested}}},
		{Number: 2, Title: "Mine, changes requested", SourceBranch: "mine", Author: me, Reviews: []hostapi.Review{{Reviewer: bob, State: hostapi.ReviewChangesRequested}}},
		{Number: 3, Title: "Mine, failing", SourceBranch: "mine-2", Author: me, CI: hostapi.CIFailed},
		{Number: 4, Title: "Fix login", SourceBranch: "12-fix-login", Author: bob, CI: hostapi.CIFailed},
		{Number: 5, Title: "Nothing for me", SourceBranch: "other", Author: bob},
	}
	hostClient.Cache.Put(info.Hostname, repo, cacheUser, remote.Name, me)
	hostClient.Cache.Put(info.Hostname, repo, cacheChanges, info.Project, cachedChanges{changes, false})
	hostClient.Cache.Put(info.Hostname, repo, cacheAssigned, info.Project, []hostapi.Issue{{Key: "12", Number: 12, Title: "Login fails", WebURL: "https://gitlab.example.com/group/app/-/issues/12"}})

	var inbox InboxService
	if n := inbox.Unseen([]string{repo}); n != 4 {
		t.Fatalf("Unseen before the Inbox opened = %d, want 4", n)
	}
	repos := inbox.Rows([]string{repo}, true)
	if !repos[0].HostEnabled || repos[0].FetchedAt == nil {
		t.Fatalf("a refreshed host: enabled with a time, got %+v", repos[0])
	}
	rows := inboxKinds(repos[0].Rows)
	for _, r := range repos[0].Rows {
		if !r.New || r.ID == "" {
			t.Errorf("first showing: every host row is new with an id: %+v", r)
		}
	}
	if n := inbox.Unseen([]string{repo}); n != 0 {
		t.Fatalf("Unseen after the Inbox showed them = %d, want 0", n)
	}
	changes = append(changes, hostapi.Change{Number: 6, Title: "Another for me", SourceBranch: "six", Author: bob, Reviews: []hostapi.Review{{Reviewer: me, State: hostapi.ReviewRequested}}})
	hostClient.Cache.Put(info.Hostname, repo, cacheChanges, info.Project, cachedChanges{changes, false})
	if n := inbox.Unseen([]string{repo}); n != 1 {
		t.Fatalf("Unseen after a refresh brought one more = %d, want 1", n)
	}
	for _, r := range inbox.Rows([]string{repo}, false)[0].Rows {
		if r.New != (r.ID == "review:6") {
			t.Errorf("only the newcomer is new: %+v", r)
		}
	}
	if r := rows["review/bobs"]; r.Number != 1 || r.Prefix != "!" || r.FetchedAt == nil || time.Since(*r.FetchedAt) > time.Minute {
		t.Errorf("review row = %+v", r)
	}
	if r := rows["changes-requested/mine"]; r.Number != 2 || r.Detail != "Bob" {
		t.Errorf("changes-requested row = %+v", r)
	}
	if r := rows["ci-failed/mine-2"]; r.Number != 3 {
		t.Errorf("ci-failed row = %+v", r)
	}
	if r := rows["assigned/"]; r.Key != "12" || r.Title != "Login fails" || r.Detail != "!4 failing CI" {
		t.Errorf("assigned row = %+v", r)
	}
	for kind := range rows {
		if kind == "review/other" || kind == "ci-failed/12-fix-login" {
			t.Errorf("unexpected row %s", kind)
		}
	}
}

func TestRefreshBringsTheAssignedIssues(t *testing.T) {
	repo := enabledGitLab(t)
	api := "/api/v4/projects/group%2Fapp"
	assigned := api + "/issues?assignee_username=ada&order_by=updated_at&per_page=50&state=opened"
	transport := &routeTransport{routes: map[string]string{
		"/api/v4/user": `{"username": "ada", "name": "Ada"}`,
		api + "/merge_requests?per_page=100&state=opened": `[]`,
		assigned: `[{"iid": 12, "title": "Login fails on Windows", "state": "opened", "web_url": "https://gitlab.example.com/group/app/-/issues/12"}]`,
	}, hits: map[string]int{}}
	swapHostClient(t, transport)
	var s HostService
	if before := (&InboxService{}).Rows([]string{repo}, false)[0]; !before.HostEnabled || before.FetchedAt != nil || len(before.Rows) != 0 {
		t.Fatalf("before the first refresh the host is on with no time and no rows: %+v", before)
	}
	if err := s.Refresh(repo); err != nil {
		t.Fatal(err)
	}
	if transport.hits[assigned] != 1 {
		t.Fatalf("assigned issues asked %d times on refresh, want once", transport.hits[assigned])
	}
	repos := (&InboxService{}).Rows([]string{repo}, true)
	if r := inboxKinds(repos[0].Rows)["assigned/"]; r.Title != "Login fails on Windows" {
		t.Fatalf("assigned row = %+v (rows %+v)", r, repos[0].Rows)
	}
	if transport.hits[assigned] != 1 {
		t.Fatal("opening the Inbox must not ask again")
	}
	if cached, _ := s.IssueTitles(repo, []int{12}); cached[12].Title != "Login fails on Windows" {
		t.Fatalf("hover cards should know the title: %+v", cached)
	}
}
