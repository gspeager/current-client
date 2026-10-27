// Package jira reads a Jira site's issues, Cloud or Data Center, through
// the one hostapi.Client: the same enabled set, trust, proxy and back-off as
// the code hosts. It only reads.
package jira

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostlink"
)

// Provider is one Jira site. It takes the site's tracker settings for its
// keys and links; the client decides whether it may be contacted.
type Provider struct {
	client  *hostapi.Client
	host    hostapi.Host
	tracker hostlink.Tracker
}

var _ hostapi.IssueTracker = (*Provider)(nil)

// HostName is how the client names the site: its host, with a port if it
// has one.
func HostName(tracker hostlink.Tracker) string {
	u, err := url.Parse(tracker.Site)
	if err != nil {
		return ""
	}
	return u.Host
}

// New reads version 2 of the REST API, which Cloud and Data Center both
// serve with descriptions as text.
func New(client *hostapi.Client, tracker hostlink.Tracker) *Provider {
	return &Provider{client: client, host: hostapi.Host{Name: HostName(tracker), API: tracker.Site + "rest/api/2"}, tracker: tracker}
}

type user struct {
	DisplayName string `json:"displayName"`
	Name        string `json:"name"`
	AccountID   string `json:"accountId"`
}

type issue struct {
	Key    string `json:"key"`
	Fields struct {
		Summary     string   `json:"summary"`
		Description string   `json:"description"`
		Labels      []string `json:"labels"`
		Status      struct {
			Name           string `json:"name"`
			StatusCategory struct {
				Key string `json:"key"`
			} `json:"statusCategory"`
		} `json:"status"`
		FixVersions []struct {
			Name string `json:"name"`
		} `json:"fixVersions"`
		Reporter *user  `json:"reporter"`
		Created  string `json:"created"`
		Updated  string `json:"updated"`
	} `json:"fields"`
}

type comment struct {
	Author  user   `json:"author"`
	Body    string `json:"body"`
	Created string `json:"created"`
}

var issueKey = regexp.MustCompile(`^[A-Z][A-Z0-9_]+-[1-9]\d*$`)

// key refuses anything that isn't shaped like an issue key before it goes
// into a path, as a numbered host refuses a key that isn't a number.
func key(raw string) (string, error) {
	if !issueKey.MatchString(raw) {
		return "", &hostapi.HostError{Status: 404}
	}
	return raw, nil
}

func (p *Provider) IssueByKey(ctx context.Context, _, rawKey string) (hostapi.Issue, error) {
	details, err := p.IssueDetails(ctx, "", rawKey)
	return details.Issue, err
}

// AssignedIssues asks for the user the token belongs to, so username is
// unused; with project keys set, only those projects.
func (p *Provider) AssignedIssues(ctx context.Context, _, _ string) ([]hostapi.Issue, error) {
	jql := "assignee = currentUser() AND statusCategory != Done"
	if len(p.tracker.Keys) > 0 {
		jql += " AND project in (" + strings.Join(p.tracker.Keys, ", ") + ")"
	}
	query := url.Values{"jql": {jql + " ORDER BY updated DESC"}, "maxResults": {"50"}, "fields": {"summary,status"}}
	path := "search"
	if p.tracker.Kind == hostlink.JiraCloud {
		path = "search/jql"
	}
	var page struct {
		Issues []issue `json:"issues"`
	}
	if err := p.client.GetJSON(ctx, p.host, path, query, &page); err != nil {
		return nil, err
	}
	issues := make([]hostapi.Issue, 0, len(page.Issues))
	for _, is := range page.Issues {
		issues = append(issues, p.toIssue(is))
	}
	return issues, nil
}

func (p *Provider) KeyPattern() *regexp.Regexp {
	return p.tracker.KeyPattern()
}

func (p *Provider) KeyInBranch(branch string) string {
	if m := p.tracker.BranchKeyPattern().FindStringSubmatch(branch); m != nil {
		return m[1]
	}
	return ""
}

func (p *Provider) Reference(key string) string {
	return key
}

func (p *Provider) IssueDetails(ctx context.Context, _, rawKey string) (hostapi.IssueDetails, error) {
	k, err := key(rawKey)
	if err != nil {
		return hostapi.IssueDetails{}, err
	}
	query := url.Values{"fields": {"summary,status,description,labels,fixVersions,reporter,created,updated"}}
	var is issue
	if err := p.client.GetJSON(ctx, p.host, "issue/"+k, query, &is); err != nil {
		return hostapi.IssueDetails{}, err
	}
	details := hostapi.IssueDetails{
		Issue:       p.toIssue(is),
		Description: is.Fields.Description,
		Labels:      is.Fields.Labels,
		CreatedAt:   parseTime(is.Fields.Created),
		UpdatedAt:   parseTime(is.Fields.Updated),
	}
	if details.Labels == nil {
		details.Labels = []string{}
	}
	if len(is.Fields.FixVersions) > 0 {
		details.Milestone = is.Fields.FixVersions[0].Name
	}
	if is.Fields.Reporter != nil {
		details.Author = toUser(*is.Fields.Reporter)
	}
	return details, nil
}

// IssueComments is the first 100, oldest first, as Jira lists them.
func (p *Provider) IssueComments(ctx context.Context, _, rawKey string) ([]hostapi.Note, error) {
	k, err := key(rawKey)
	if err != nil {
		return nil, err
	}
	var page struct {
		Comments []comment `json:"comments"`
	}
	if err := p.client.GetJSON(ctx, p.host, "issue/"+k+"/comment", url.Values{"maxResults": {"100"}}, &page); err != nil {
		return nil, err
	}
	notes := make([]hostapi.Note, 0, len(page.Comments))
	for _, c := range page.Comments {
		notes = append(notes, hostapi.Note{Author: toUser(c.Author), Body: c.Body, CreatedAt: parseTime(c.Created)})
	}
	return notes, nil
}

// CurrentUser is whom the token signs in as, for Test connection.
func (p *Provider) CurrentUser(ctx context.Context) (hostapi.User, error) {
	var me user
	if err := p.client.GetJSON(ctx, p.host, "myself", nil, &me); err != nil {
		return hostapi.User{}, err
	}
	return toUser(me), nil
}

// toIssue counts any status in the Done category as closed.
func (p *Provider) toIssue(is issue) hostapi.Issue {
	state := "open"
	if is.Fields.Status.StatusCategory.Key == "done" {
		state = "closed"
	}
	return hostapi.Issue{Key: is.Key, Title: is.Fields.Summary, State: state, WebURL: p.tracker.IssueURL(is.Key)}
}

// toUser keeps the name to show and an id for initials; Cloud has no
// username, only an account id.
func toUser(u user) hostapi.User {
	username := u.Name
	if username == "" {
		username = u.AccountID
	}
	return hostapi.User{Name: u.DisplayName, Username: username}
}

// parseTime reads Jira's "2026-10-20T10:00:00.000+0000"; anything else is
// the zero time.
func parseTime(raw string) time.Time {
	t, err := time.Parse("2006-01-02T15:04:05.000-0700", raw)
	if err != nil {
		return time.Time{}
	}
	return t
}
