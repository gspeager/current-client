//go:build !nohost

package app

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/hostapi/gitlab"
)

func TestIssueDetailsAsksOnceAndRendersComments(t *testing.T) {
	repo := enabledGitLab(t)
	api := "/api/v4/projects/group%2Fapp"
	transport := &routeTransport{routes: map[string]string{
		api + "/issues/4": `{"iid": 4, "title": "Login fails", "state": "opened", "description": "Steps:\n\n- open **login**", "labels": ["bug", "auth"], "milestone": {"title": "0.5.0"}, "author": {"name": "Ada", "username": "ada"}, "web_url": "https://gitlab.example.com/group/app/-/issues/4"}`,
		api + "/issues/4/notes?order_by=created_at&per_page=100&sort=asc": `[{"author": {"name": "Bot", "username": "bot"}, "body": "changed the description", "system": true}, {"author": {"name": "Grace", "username": "grace"}, "body": "Same on *Linux*", "created_at": "2026-10-20T10:00:00Z"}]`,
	}, hits: map[string]int{}}
	swapHostClient(t, transport)
	var s HostService

	details, err := s.IssueDetails(repo, "4")
	if err != nil {
		t.Fatal(err)
	}
	if details.Issue.Title != "Login fails" || details.Issue.Milestone != "0.5.0" || !reflect.DeepEqual(details.Issue.Labels, []string{"bug", "auth"}) || details.Reference != "#4" {
		t.Fatalf("IssueDetails = %+v", details)
	}
	if !strings.Contains(details.DescriptionHTML, "<strong>login</strong>") {
		t.Fatalf("description not rendered: %q", details.DescriptionHTML)
	}
	if len(details.Comments) != 1 || details.Comments[0].Author.Name != "Grace" || !strings.Contains(details.Comments[0].BodyHTML, "<em>Linux</em>") {
		t.Fatalf("comments = %+v", details.Comments)
	}
	s.IssueDetails(repo, "4")
	for path, hits := range transport.hits {
		if hits != 1 {
			t.Errorf("%s asked %d times, want once", path, hits)
		}
	}
	if cached, _ := s.IssueTitles(repo, []int{4}); cached[4].Title != "Login fails" {
		t.Fatalf("the panel should fill the hover card: %+v", cached)
	}
}

func TestClosedIssuesFromADescription(t *testing.T) {
	var tracker gitlab.Provider
	got := closedIssues("Adds the form.\n\nCloses #12, fixes #7 and Resolves: #12. See #99 and foo#3.", &tracker)
	if want := []string{"#12", "#7"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("closedIssues = %v, want %v", got, want)
	}
}
