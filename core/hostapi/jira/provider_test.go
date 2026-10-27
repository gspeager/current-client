package jira

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostapi/hostapitest"
	"github.com/gspeager/current-client/core/hostlink"
)

type site struct {
	mu       sync.Mutex
	routes   map[string]string
	requests []*http.Request
}

func (s *site) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.requests = append(s.requests, r)
	s.mu.Unlock()
	key := r.URL.Path
	if r.URL.RawQuery != "" {
		key += "?" + r.URL.RawQuery
	}
	body, ok := s.routes[key]
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Write([]byte(body))
}

type headerSource struct{ header string }

func (h headerSource) Token(context.Context, string) (string, error) { return "", nil }
func (h headerSource) Authorization(context.Context, string) (string, error) {
	return h.header, nil
}

func newProvider(t *testing.T, kind hostlink.TrackerKind, keys []string, routes map[string]string) (*Provider, *site) {
	t.Helper()
	hostapitest.IsolateGitConfig(t)
	s := &site{routes: routes}
	server := httptest.NewServer(s)
	t.Cleanup(server.Close)
	tracker := hostlink.Tracker{Site: server.URL + "/jira/", Kind: kind, Keys: keys}
	client := hostapi.New(hostapi.Options{Enabled: hostapitest.AllowAll, Tokens: headerSource{"Basic dGVzdA=="}})
	return New(client, tracker), s
}

const issueJSON = `{"key": "PROJ-12", "fields": {"summary": "Login fails", "description": "Steps:\n* open *login*",
	"labels": ["auth"], "status": {"name": "In Progress", "statusCategory": {"key": "indeterminate"}},
	"fixVersions": [{"name": "0.5.0"}], "reporter": {"displayName": "Ada Lovelace", "accountId": "5b10a2"},
	"created": "2026-10-20T10:00:00.000+0000", "updated": "2026-10-21T09:30:00.000+0100"}}`

func TestIssueDetailsAndComments(t *testing.T) {
	p, s := newProvider(t, hostlink.JiraDataCenter, nil, map[string]string{
		"/jira/rest/api/2/issue/PROJ-12?fields=summary%2Cstatus%2Cdescription%2Clabels%2CfixVersions%2Creporter%2Ccreated%2Cupdated": issueJSON,
		"/jira/rest/api/2/issue/PROJ-12/comment?maxResults=100": `{"comments": [{"author": {"displayName": "Grace", "name": "grace"}, "body": "Same here", "created": "2026-10-21T10:00:00.000+0000"}]}`,
	})
	ctx := context.Background()
	details, err := p.IssueDetails(ctx, "", "PROJ-12")
	if err != nil {
		t.Fatal(err)
	}
	if details.Key != "PROJ-12" || details.Number != 0 || details.Title != "Login fails" || details.State != "open" ||
		details.Milestone != "0.5.0" || details.Author != (hostapi.User{Name: "Ada Lovelace", Username: "5b10a2"}) ||
		!strings.HasSuffix(details.WebURL, "/jira/browse/PROJ-12") || details.UpdatedAt.UTC().Hour() != 8 {
		t.Fatalf("IssueDetails = %+v", details)
	}
	notes, err := p.IssueComments(ctx, "", "PROJ-12")
	if err != nil || len(notes) != 1 || notes[0].Author.Username != "grace" || notes[0].Body != "Same here" {
		t.Fatalf("IssueComments = %+v, %v", notes, err)
	}
	if got := s.requests[0].Header.Get("Authorization"); got != "Basic dGVzdA==" {
		t.Errorf("Authorization = %q", got)
	}
	var hostErr *hostapi.HostError
	if _, err := p.IssueDetails(ctx, "", "../myself"); !errors.As(err, &hostErr) || hostErr.Status != 404 {
		t.Errorf("a key that isn't one = %v", err)
	}
	if len(s.requests) != 2 {
		t.Errorf("a bad key was sent: %d requests", len(s.requests))
	}
}

func TestAssignedIssuesOnCloudAndDataCenter(t *testing.T) {
	answer := `{"issues": [{"key": "PROJ-3", "fields": {"summary": "Done one", "status": {"statusCategory": {"key": "done"}}}}, {"key": "OPS-4", "fields": {"summary": "Open one", "status": {"statusCategory": {"key": "new"}}}}]}`
	query := url.Values{
		"jql":        {"assignee = currentUser() AND statusCategory != Done AND project in (PROJ, OPS) ORDER BY updated DESC"},
		"maxResults": {"50"},
		"fields":     {"summary,status"},
	}.Encode()
	for kind, path := range map[hostlink.TrackerKind]string{hostlink.JiraCloud: "/jira/rest/api/2/search/jql", hostlink.JiraDataCenter: "/jira/rest/api/2/search"} {
		p, _ := newProvider(t, kind, []string{"PROJ", "OPS"}, map[string]string{path + "?" + query: answer})
		issues, err := p.AssignedIssues(context.Background(), "", "")
		if err != nil || len(issues) != 2 || issues[0].State != "closed" || issues[1].Key != "OPS-4" || issues[1].State != "open" {
			t.Fatalf("%s AssignedIssues = %+v, %v", kind, issues, err)
		}
	}
}

func TestNoRequestWhileTheSiteIsOff(t *testing.T) {
	tracker := hostlink.Tracker{Site: "https://example.atlassian.net/", Kind: hostlink.JiraCloud}
	client := hostapi.New(hostapi.Options{Enabled: func(string) bool { return false }, Transport: hostapitest.Forbid(t)})
	p := New(client, tracker)
	if _, err := p.IssueDetails(context.Background(), "", "PROJ-1"); !errors.Is(err, hostapi.ErrNotEnabled) {
		t.Fatalf("err = %v", err)
	}
	if HostName(tracker) != "example.atlassian.net" {
		t.Fatalf("HostName = %q", HostName(tracker))
	}
}

func TestKeysAndBranches(t *testing.T) {
	p := New(hostapi.New(hostapi.Options{}), hostlink.Tracker{Site: "https://x/", Keys: []string{"PROJ"}})
	if p.KeyInBranch("feature/PROJ-9-fix") != "PROJ-9" || p.KeyInBranch("9-fix") != "" || p.Reference("PROJ-9") != "PROJ-9" {
		t.Fatal("branch keys")
	}
	if m := p.KeyPattern().FindStringSubmatch("fix (PROJ-3)"); m == nil || m[1] != "PROJ-3" {
		t.Fatal("key pattern")
	}
}
