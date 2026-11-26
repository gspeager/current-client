package gitlab

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gspeager/current-client/core/hostapi"
)

var _ hostapi.IssueWriter = (*Provider)(nil)

func (p *Provider) CreateIssue(ctx context.Context, project string, input hostapi.NewIssue) (hostapi.Issue, error) {
	body := map[string]string{"title": input.Title, "description": input.Description}
	if len(input.Labels) > 0 {
		body["labels"] = strings.Join(input.Labels, ",")
	}
	var created issue
	if err := p.post(ctx, writeScope, projectPath(project, "issues"), body, &created); err != nil {
		return hostapi.Issue{}, err
	}
	return toIssue(created), nil
}

func (p *Provider) CommentOnIssue(ctx context.Context, project, key, body string) (hostapi.Note, error) {
	number, err := hostapi.IssueNumber(key)
	if err != nil {
		return hostapi.Note{}, err
	}
	var n issueNote
	if err := p.post(ctx, writeScope, projectPath(project, "issues", strconv.Itoa(number), "notes"), map[string]string{"body": body}, &n); err != nil {
		return hostapi.Note{}, err
	}
	return hostapi.Note{Author: toUser(n.Author), Body: n.Body, CreatedAt: n.CreatedAt}, nil
}

func (p *Provider) SetIssueState(ctx context.Context, project, key string, open bool) (hostapi.IssueDetails, error) {
	number, err := hostapi.IssueNumber(key)
	if err != nil {
		return hostapi.IssueDetails{}, err
	}
	event := "close"
	if open {
		event = "reopen"
	}
	var is issue
	if err := p.send(ctx, writeScope, http.MethodPut, projectPath(project, "issues", strconv.Itoa(number)), map[string]string{"state_event": event}, &is); err != nil {
		return hostapi.IssueDetails{}, err
	}
	return toIssueDetails(is), nil
}

func (p *Provider) issue(ctx context.Context, project string, number int) (issue, error) {
	var is issue
	err := p.get(ctx, readScope, projectPath(project, "issues", strconv.Itoa(number)), nil, &is)
	return is, err
}

func (p *Provider) Issue(ctx context.Context, project string, number int) (hostapi.Issue, error) {
	is, err := p.issue(ctx, project, number)
	if err != nil {
		return hostapi.Issue{}, err
	}
	return toIssue(is), nil
}

func (p *Provider) AssignedIssues(ctx context.Context, project, username string) ([]hostapi.Issue, error) {
	return p.listIssues(ctx, project, url.Values{"assignee_username": {username}, "state": {"opened"}})
}

func (p *Provider) ProjectIssues(ctx context.Context, project string, open bool) ([]hostapi.Issue, error) {
	state := "closed"
	if open {
		state = "opened"
	}
	return p.listIssues(ctx, project, url.Values{"state": {state}})
}

func (p *Provider) listIssues(ctx context.Context, project string, query url.Values) ([]hostapi.Issue, error) {
	query.Set("order_by", "updated_at")
	query.Set("per_page", "50")
	var page []issue
	if err := p.get(ctx, readScope, projectPath(project, "issues"), query, &page); err != nil {
		return nil, err
	}
	issues := make([]hostapi.Issue, 0, len(page))
	for _, is := range page {
		issues = append(issues, toIssue(is))
	}
	return issues, nil
}

func (p *Provider) IssueByKey(ctx context.Context, project, key string) (hostapi.Issue, error) {
	number, err := hostapi.IssueNumber(key)
	if err != nil {
		return hostapi.Issue{}, err
	}
	return p.Issue(ctx, project, number)
}

func (p *Provider) IssueDetails(ctx context.Context, project, key string) (hostapi.IssueDetails, error) {
	number, err := hostapi.IssueNumber(key)
	if err != nil {
		return hostapi.IssueDetails{}, err
	}
	is, err := p.issue(ctx, project, number)
	if err != nil {
		return hostapi.IssueDetails{}, err
	}
	return toIssueDetails(is), nil
}

// IssueComments leaves out system notes, GitLab's own record of label and
// state changes.
func (p *Provider) IssueComments(ctx context.Context, project, key string) ([]hostapi.Note, error) {
	number, err := hostapi.IssueNumber(key)
	if err != nil {
		return nil, err
	}
	var page []issueNote
	query := url.Values{"sort": {"asc"}, "order_by": {"created_at"}, "per_page": {"100"}}
	if err := p.get(ctx, readScope, projectPath(project, "issues", strconv.Itoa(number), "notes"), query, &page); err != nil {
		return nil, err
	}
	notes := make([]hostapi.Note, 0, len(page))
	for _, n := range page {
		if !n.System {
			notes = append(notes, hostapi.Note{Author: toUser(n.Author), Body: n.Body, CreatedAt: n.CreatedAt})
		}
	}
	return notes, nil
}

func toIssue(is issue) hostapi.Issue {
	state := "open"
	if is.State == "closed" {
		state = "closed"
	}
	return hostapi.Issue{Key: strconv.Itoa(is.IID), Number: is.IID, Title: is.Title, State: state, WebURL: is.WebURL}
}

func toIssueDetails(is issue) hostapi.IssueDetails {
	labels := is.Labels
	if labels == nil {
		labels = []string{}
	}
	out := hostapi.IssueDetails{Issue: toIssue(is), Description: is.Description, Labels: labels, Author: toUser(is.Author), CreatedAt: is.CreatedAt, UpdatedAt: is.UpdatedAt}
	if is.Milestone != nil {
		out.Milestone = is.Milestone.Title
	}
	return out
}
