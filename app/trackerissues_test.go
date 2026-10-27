//go:build !nohost

package app

import (
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/hostapi/hostapitest"
	"github.com/gspeager/current-client/internal/policy"
)

func jiraRepo(t *testing.T) (repo, site string) {
	t.Helper()
	useTempSettings(t)
	repo = t.TempDir()
	gitIn(t, repo, "init", "-q")
	var s TrackerService
	added, err := s.Add("https://example.atlassian.net", "jira-cloud", []string{"PROJ"})
	if err != nil {
		t.Fatal(err)
	}
	s.SetRepos(added.Site, []string{repo})
	return repo, added.Site
}

func TestJiraIssuesReadOnlyWhileTheSiteIsOn(t *testing.T) {
	repo, site := jiraRepo(t)
	swapHostClient(t, hostapitest.Forbid(t))
	var hosts HostService
	if _, err := hosts.IssueDetails(repo, "PROJ-12"); err == nil || !strings.Contains(err.Error(), "issue tracker") {
		t.Fatalf("with the site off: %v", err)
	}
	if titles := hosts.FetchTrackerIssueTitles(repo, []string{"PROJ-12"}); len(titles) != 0 {
		t.Fatalf("titles with the site off = %v", titles)
	}
	if links := (&TrackerService{}).Links(repo); links.Reading {
		t.Fatal("reading with the site off")
	}

	var trackers TrackerService
	if err := trackers.SetEnabled(site, true); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(policy.Path(), []byte(`{"hosts": {"example.atlassian.net": {"enabled": false}}}`), 0o644)
	policy.Load()
	if _, err := hosts.IssueDetails(repo, "PROJ-12"); err == nil {
		t.Fatal("a locked site was asked")
	}
	if err := trackers.SetEnabled(site, true); err == nil {
		t.Fatal("a locked switch was changed")
	}
	if list := trackers.List(); !list.Trackers[0].Locked || list.Trackers[0].Enabled {
		t.Fatalf("locked row = %+v", list.Trackers[0])
	}
}

func TestJiraIssuePanelTitlesAssignedAndInbox(t *testing.T) {
	repo, site := jiraRepo(t)
	(&TrackerService{}).SetEnabled(site, true)
	api := "/rest/api/2"
	fields := url.Values{"fields": {"summary,status,description,labels,fixVersions,reporter,created,updated"}}.Encode()
	assigned := url.Values{
		"jql":        {"assignee = currentUser() AND statusCategory != Done AND project in (PROJ) ORDER BY updated DESC"},
		"maxResults": {"50"},
		"fields":     {"summary,status"},
	}.Encode()
	transport := &routeTransport{routes: map[string]string{
		api + "/issue/PROJ-12?" + fields:              `{"key": "PROJ-12", "fields": {"summary": "Login fails", "description": "Steps *here*", "status": {"statusCategory": {"key": "new"}}, "reporter": {"displayName": "Ada"}}}`,
		api + "/issue/PROJ-12/comment?maxResults=100": `{"comments": [{"author": {"displayName": "Grace", "accountId": "a1"}, "body": "Same"}]}`,
		api + "/issue/PROJ-13?" + fields:              "404",
		api + "/search/jql?" + assigned:               `{"issues": [{"key": "PROJ-20", "fields": {"summary": "Mine", "status": {"statusCategory": {"key": "indeterminate"}}}}]}`,
	}, hits: map[string]int{}}
	swapHostClient(t, transport)
	var hosts HostService

	details, err := hosts.IssueDetails(repo, "PROJ-12")
	if err != nil || details.Reference != "PROJ-12" || details.Writable || len(details.Comments) != 1 ||
		!strings.Contains(details.DescriptionHTML, "Steps") || details.Issue.WebURL != "https://example.atlassian.net/browse/PROJ-12" {
		t.Fatalf("IssueDetails = %+v, %v", details, err)
	}
	hosts.IssueDetails(repo, "PROJ-12")
	if n := transport.hits[api+"/issue/PROJ-12?"+fields]; n != 1 {
		t.Fatalf("issue asked %d times, want once", n)
	}

	titles := hosts.FetchTrackerIssueTitles(repo, []string{"PROJ-12", "PROJ-13", "PROJ-12"})
	if titles["PROJ-12"].Title != "Login fails" || !titles["PROJ-13"].Known || titles["PROJ-13"].Title != "" {
		t.Fatalf("titles = %+v", titles)
	}
	if cached := hosts.TrackerIssueTitles(repo, []string{"PROJ-13"}); !cached["PROJ-13"].Known {
		t.Fatal("a missing key should be remembered")
	}

	issues, err := hosts.AssignedIssues(repo)
	if err != nil || len(issues) != 1 || issues[0].Key != "PROJ-20" {
		t.Fatalf("AssignedIssues = %+v, %v", issues, err)
	}
	var inbox InboxService
	rows := inbox.Rows([]string{repo}, false)[0].Rows
	found := false
	for _, r := range rows {
		found = found || (r.Kind == "assigned" && r.Key == "PROJ-20" && r.FetchedAt != nil)
	}
	if !found {
		t.Fatalf("Inbox rows = %+v", rows)
	}

	if err := hosts.Refresh(repo); err != nil {
		t.Fatal(err)
	}
	if n := transport.hits[api+"/search/jql?"+assigned]; n != 2 {
		t.Fatalf("a refresh should read the assigned issues again: %d", n)
	}
	if cached := hosts.TrackerIssueTitles(repo, []string{"PROJ-12"}); len(cached) != 0 {
		t.Fatal("a refresh should clear the tracker's titles")
	}
}
