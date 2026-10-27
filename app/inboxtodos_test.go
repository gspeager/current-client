//go:build !nohost

package app

import (
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostapi/hostapitest"
	"github.com/gspeager/current-client/internal/config"
)

func enabledGitHub(t *testing.T) (repo string) {
	t.Helper()
	useTempSettings(t)
	repo = t.TempDir()
	gitIn(t, repo, "init", "-q")
	gitIn(t, repo, "remote", "add", "origin", "git@github.example.com:Owner/App.git")
	if err := updateConfig(func(cfg *config.Config) {
		cfg.Hosts = map[string]config.HostConfig{"github.example.com": {Enabled: true, Provider: "github"}}
	}); err != nil {
		t.Fatal(err)
	}
	return repo
}

func TestRefreshBringsTheHostsToDosForOpenRepositories(t *testing.T) {
	repo := enabledGitLab(t)
	api := "/api/v4/projects/group%2Fapp"
	todos := "/api/v4/todos?per_page=100&state=pending"
	transport := &routeTransport{routes: map[string]string{
		"/api/v4/user": `{"username": "ada", "name": "Ada"}`,
		api + "/merge_requests?per_page=100&state=opened": `[
			{"iid": 1, "title": "Bob's change", "source_branch": "bobs", "state": "opened", "author": {"username": "bob", "name": "Bob"}, "reviewers": [{"username": "ada", "name": "Ada"}], "web_url": "https://gitlab.example.com/group/app/-/merge_requests/1"},
			{"iid": 7, "title": "Add the thing", "source_branch": "thing", "state": "opened", "author": {"username": "bob", "name": "Bob"}, "web_url": "https://gitlab.example.com/group/app/-/merge_requests/7"}
		]`,
		api + "/merge_requests/1/reviewers":                                                `[{"user": {"username": "ada", "name": "Ada"}, "state": "unreviewed"}]`,
		api + "/merge_requests/1/approvals":                                                `{"approved_by": []}`,
		api + "/merge_requests/7/reviewers":                                                `[]`,
		api + "/merge_requests/7/approvals":                                                `{"approved_by": []}`,
		api + "/issues?assignee_username=ada&order_by=updated_at&per_page=50&state=opened": `[{"iid": 12, "title": "Login fails", "state": "opened", "web_url": "https://gitlab.example.com/group/app/-/issues/12"}]`,
		todos: `[
			{"id": 101, "project": {"path_with_namespace": "group/app"}, "action_name": "mentioned", "target_type": "MergeRequest", "target": {"iid": 7, "title": "Add the thing"}, "target_url": "https://gitlab.example.com/group/app/-/merge_requests/7", "author": {"username": "bob", "name": "Bob"}, "updated_at": "2026-10-20T10:00:00Z"},
			{"id": 102, "project": {"path_with_namespace": "group/app"}, "action_name": "review_requested", "target_type": "MergeRequest", "target": {"iid": 1, "title": "Bob's change"}, "target_url": "https://gitlab.example.com/group/app/-/merge_requests/1", "author": {"username": "bob", "name": "Bob"}, "updated_at": "2026-10-20T10:00:00Z"},
			{"id": 103, "project": {"path_with_namespace": "group/app"}, "action_name": "assigned", "target_type": "Issue", "target": {"iid": 12, "title": "Login fails"}, "target_url": "https://gitlab.example.com/group/app/-/issues/12", "author": {"username": "bob", "name": "Bob"}, "updated_at": "2026-10-20T10:00:00Z"},
			{"id": 104, "project": {"path_with_namespace": "group/other"}, "action_name": "mentioned", "target_type": "Issue", "target": {"iid": 3, "title": "Elsewhere"}, "target_url": "https://gitlab.example.com/group/other/-/issues/3", "author": {"username": "bob", "name": "Bob"}, "updated_at": "2026-10-20T10:00:00Z"}
		]`,
		"/api/v4/todos/101/mark_as_done": `{"id": 101, "state": "done"}`,
	}, hits: map[string]int{}}
	swapHostClient(t, transport)
	var s HostService
	if err := s.Refresh(repo); err != nil {
		t.Fatal(err)
	}
	if transport.hits[todos] != 1 {
		t.Fatalf("to-dos asked %d times on refresh, want once", transport.hits[todos])
	}
	for _, value := range hostClient.Cache.Values("gitlab.example.com", "", cacheTodos) {
		for _, todo := range value.([]hostapi.Todo) {
			if todo.Project != "group/app" {
				t.Errorf("a to-do about another repository was kept: %+v", todo)
			}
		}
	}

	var inbox InboxService
	repos := inbox.Rows([]string{repo}, true)
	rows := inboxKinds(repos[0].Rows)
	if r := rows["todo/"]; r.Number != 7 || r.Prefix != "!" || r.Title != "Add the thing" || r.Detail != "mentioned" || r.TodoID != "101" || r.FetchedAt == nil {
		t.Errorf("mention row = %+v", r)
	}
	if r := rows["review/bobs"]; r.TodoID != "102" {
		t.Errorf("the review request joins its to-do: %+v", r)
	}
	if r := rows["assigned/"]; r.TodoID != "103" {
		t.Errorf("the assignment joins its to-do: %+v", r)
	}
	if n := len(repos[0].Rows); n != 3 || repos[0].Notice != "" {
		t.Errorf("rows = %d %+v, notice %q", n, repos[0].Rows, repos[0].Notice)
	}
	if transport.hits[todos] != 1 {
		t.Fatal("opening the Inbox must not ask again")
	}

	if err := inbox.MarkDone(repo, "101"); err != nil {
		t.Fatal(err)
	}
	if transport.hits["/api/v4/todos/101/mark_as_done"] != 1 {
		t.Fatal("mark done should post once")
	}
	if rows := inboxKinds(inbox.Rows([]string{repo}, true)[0].Rows); rows["todo/"].TodoID != "" || rows["review/bobs"].TodoID != "102" {
		t.Errorf("after marking done the to-do row goes and the rest stay: %+v", rows)
	}
	if transport.hits[todos] != 1 {
		t.Fatal("marking done must not refresh")
	}
}

func TestGitHubToDosNeedAClassicToken(t *testing.T) {
	repo := enabledGitHub(t)
	api := "/api/v3/repos/Owner/App"
	transport := &routeTransport{routes: map[string]string{
		"/api/v3/user":                         `{"login": "ada", "name": "Ada"}`,
		api + "/pulls?per_page=100&state=open": `[]`,
		api + "/issues?assignee=ada&per_page=50&sort=updated&state=open": `[]`,
		"/api/v3/notifications?per_page=50":                              "403",
	}, hits: map[string]int{}}
	swapHostClient(t, transport)
	var s HostService
	if err := s.Refresh(repo); err != nil {
		t.Fatal(err)
	}
	if transport.hits["/api/v3/notifications?per_page=50"] != 1 {
		t.Fatalf("notifications asked %d times, want once", transport.hits["/api/v3/notifications?per_page=50"])
	}
	repos := (&InboxService{}).Rows([]string{repo, repo}, true)
	if !strings.Contains(repos[0].Notice, "classic token") || !strings.Contains(repos[0].Notice, "notifications") {
		t.Fatalf("notice = %q", repos[0].Notice)
	}
	if repos[1].Notice != "" {
		t.Fatalf("the notice shows once per host, got %q again", repos[1].Notice)
	}
	if !repos[0].HostEnabled || repos[0].FetchedAt == nil {
		t.Fatalf("everything else keeps working: %+v", repos[0])
	}
}

func TestToDosAreNeverAskedWithTheHostOff(t *testing.T) {
	useTempSettings(t)
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q")
	gitIn(t, repo, "remote", "add", "origin", "git@gitlab.example.com:group/app.git")
	swapHostClient(t, hostapitest.Forbid(t))
	var s HostService
	if err := s.Refresh(repo); err != nil {
		t.Fatal(err)
	}
	repos := (&InboxService{}).Rows([]string{repo}, true)
	if repos[0].HostEnabled || repos[0].Notice != "" || len(repos[0].Rows) != 0 {
		t.Fatalf("host off: %+v", repos[0])
	}
	if err := (&InboxService{}).MarkDone(repo, "1"); err == nil {
		t.Fatal("marking done with the host off must fail without a request")
	}
}
