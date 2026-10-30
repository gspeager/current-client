//go:build !nohost

package app

import (
	"testing"

	"github.com/gspeager/current-client/internal/config"
)

func TestIssueWritesKeepThePanelCurrent(t *testing.T) {
	repo := enabledGitLab(t)
	api := "/api/v4/projects/group%2Fapp"
	transport := &routeTransport{routes: map[string]string{
		api + "/issues":   `{"iid": 9, "title": "TODO: drop the shim", "state": "opened"}`,
		api + "/issues/4": `{"iid": 4, "title": "Login fails", "state": "opened"}`,
		api + "/issues/4/notes?order_by=created_at&per_page=100&sort=asc": `[]`,
		api + "/issues/4/notes": `{"author": {"username": "me"}, "body": "Fixed on main."}`,
		api + "/issues?assignee_username=me&order_by=updated_at&per_page=50&state=opened": `[{"iid": 4, "title": "Login fails", "state": "opened"}]`,
		"/api/v4/user": `{"username": "me"}`,
	}, hits: map[string]int{}}
	swapHostClient(t, transport)
	var s HostService

	if !s.CanWriteIssues(repo) {
		t.Fatal("GitLab issues should be writable")
	}
	created, err := s.CreateIssue(repo, NewIssueInput{Title: "  TODO: drop the shim "})
	if err != nil || created.Number != 9 {
		t.Fatalf("CreateIssue = %+v, %v", created, err)
	}
	if titles, _ := s.IssueTitles(repo, []int{9}); titles[9].Title != "TODO: drop the shim" {
		t.Fatalf("the new issue should have a hover card: %+v", titles)
	}
	if _, err := s.AssignedIssues(repo); err != nil {
		t.Fatal(err)
	}
	if _, err := s.IssueDetails(repo, "4"); err != nil {
		t.Fatal(err)
	}
	details, err := s.CommentOnIssue(repo, "4", "Fixed on main.")
	if err != nil || len(details.Comments) != 1 || details.Comments[0].Author.Username != "me" {
		t.Fatalf("CommentOnIssue = %+v, %v", details, err)
	}
	transport.routes[api+"/issues/4"] = `{"iid": 4, "title": "Login fails", "state": "closed"}`
	details, err = s.SetIssueState(repo, "4", false)
	if err != nil || details.Issue.State != "closed" {
		t.Fatalf("SetIssueState = %+v, %v", details.Issue, err)
	}
	if assigned, _ := s.AssignedIssues(repo); len(assigned) != 0 {
		t.Fatalf("a closed issue should leave the assigned list: %+v", assigned)
	}
	if hits := transport.hits[api+"/issues/4/notes?order_by=created_at&per_page=100&sort=asc"]; hits != 1 {
		t.Fatalf("comments asked %d times, want once", hits)
	}
	if _, err := s.CreateIssue(repo, NewIssueInput{Title: " "}); err == nil {
		t.Fatal("an issue without a title was sent")
	}
}

func TestIssueWritesNeedTheHostOn(t *testing.T) {
	repo := enabledGitLab(t)
	if err := updateConfig(func(cfg *config.Config) {
		cfg.Hosts = map[string]config.HostConfig{"gitlab.example.com": {Enabled: false}}
	}); err != nil {
		t.Fatal(err)
	}
	var s HostService
	if s.CanWriteIssues(repo) {
		t.Fatal("writes offered with the host off")
	}
	if _, err := s.CommentOnIssue(repo, "4", "Hi"); err == nil {
		t.Fatal("a comment was sent with the host off")
	}
}
