package github

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gspeager/current-client/core/hostapi"
)

var _ hostapi.IssueWriter = (*Provider)(nil)

func (p *Provider) CreateIssue(ctx context.Context, project string, input hostapi.NewIssue) (hostapi.Issue, error) {
	body := map[string]any{"title": input.Title, "body": input.Description}
	if len(input.Labels) > 0 {
		body["labels"] = input.Labels
	}
	var created issue
	if err := p.post(ctx, scope, repoPath(project, "issues"), body, &created); err != nil {
		return hostapi.Issue{}, err
	}
	return toIssue(created), nil
}

func (p *Provider) CommentOnIssue(ctx context.Context, project, key, body string) (hostapi.Note, error) {
	number, err := hostapi.IssueNumber(key)
	if err != nil {
		return hostapi.Note{}, err
	}
	var c issueComment
	if err := p.post(ctx, scope, repoPath(project, "issues", strconv.Itoa(number), "comments"), map[string]string{"body": body}, &c); err != nil {
		return hostapi.Note{}, err
	}
	return hostapi.Note{Author: toUser(c.User), Body: c.Body, CreatedAt: c.CreatedAt}, nil
}

func (p *Provider) SetIssueState(ctx context.Context, project, key string, open bool) (hostapi.IssueDetails, error) {
	number, err := hostapi.IssueNumber(key)
	if err != nil {
		return hostapi.IssueDetails{}, err
	}
	state := "closed"
	if open {
		state = "open"
	}
	var is issue
	if err := p.send(ctx, scope, http.MethodPatch, repoPath(project, "issues", strconv.Itoa(number)), map[string]string{"state": state}, &is); err != nil {
		return hostapi.IssueDetails{}, err
	}
	return toIssueDetails(is), nil
}

func (p *Provider) Issue(ctx context.Context, project string, number int) (hostapi.Issue, error) {
	is, err := p.issue(ctx, project, number)
	if err != nil {
		return hostapi.Issue{}, err
	}
	return toIssue(is), nil
}

func (p *Provider) issue(ctx context.Context, project string, number int) (issue, error) {
	var is issue
	err := p.get(ctx, scope, repoPath(project, "issues", strconv.Itoa(number)), nil, &is)
	return is, err
}

func (p *Provider) AssignedIssues(ctx context.Context, project, username string) ([]hostapi.Issue, error) {
	return p.listIssues(ctx, project, url.Values{"assignee": {username}, "state": {"open"}})
}

func (p *Provider) ProjectIssues(ctx context.Context, project string, open bool) ([]hostapi.Issue, error) {
	state := "closed"
	if open {
		state = "open"
	}
	return p.listIssues(ctx, project, url.Values{"state": {state}})
}

// listIssues leaves out pull requests, which GitHub lists as issues too.
func (p *Provider) listIssues(ctx context.Context, project string, query url.Values) ([]hostapi.Issue, error) {
	query.Set("sort", "updated")
	query.Set("per_page", "50")
	var page []issue
	if err := p.get(ctx, scope, repoPath(project, "issues"), query, &page); err != nil {
		return nil, err
	}
	issues := make([]hostapi.Issue, 0, len(page))
	for _, is := range page {
		if is.PullRequest == nil {
			issues = append(issues, toIssue(is))
		}
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

func (p *Provider) IssueComments(ctx context.Context, project, key string) ([]hostapi.Note, error) {
	number, err := hostapi.IssueNumber(key)
	if err != nil {
		return nil, err
	}
	var page []issueComment
	query := url.Values{"per_page": {"100"}}
	if err := p.get(ctx, scope, repoPath(project, "issues", strconv.Itoa(number), "comments"), query, &page); err != nil {
		return nil, err
	}
	notes := make([]hostapi.Note, 0, len(page))
	for _, c := range page {
		notes = append(notes, hostapi.Note{Author: toUser(c.User), Body: c.Body, CreatedAt: c.CreatedAt})
	}
	return notes, nil
}

func toIssue(is issue) hostapi.Issue {
	return hostapi.Issue{Key: strconv.Itoa(is.Number), Number: is.Number, Title: is.Title, State: is.State, WebURL: is.HTMLURL}
}

func toIssueDetails(is issue) hostapi.IssueDetails {
	labels := make([]string, 0, len(is.Labels))
	for _, l := range is.Labels {
		labels = append(labels, l.Name)
	}
	out := hostapi.IssueDetails{Issue: toIssue(is), Description: is.Body, Labels: labels, Author: toUser(is.User), CreatedAt: is.CreatedAt, UpdatedAt: is.UpdatedAt}
	if is.Milestone != nil {
		out.Milestone = is.Milestone.Title
	}
	return out
}
