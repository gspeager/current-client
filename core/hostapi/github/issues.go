package github

import (
	"context"
	"net/http"
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
	if err := p.client.PostJSON(ctx, p.host, repoPath(project, "issues"), body, &created); err != nil {
		return hostapi.Issue{}, hostapi.WithScope(err, scope)
	}
	return toIssue(created), nil
}

func (p *Provider) CommentOnIssue(ctx context.Context, project, key, body string) (hostapi.Note, error) {
	number, err := hostapi.IssueNumber(key)
	if err != nil {
		return hostapi.Note{}, err
	}
	var c issueComment
	if err := p.client.PostJSON(ctx, p.host, repoPath(project, "issues", strconv.Itoa(number), "comments"), map[string]string{"body": body}, &c); err != nil {
		return hostapi.Note{}, hostapi.WithScope(err, scope)
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
	if err := p.client.RequestJSON(ctx, p.host, http.MethodPatch, repoPath(project, "issues", strconv.Itoa(number)), map[string]string{"state": state}, &is); err != nil {
		return hostapi.IssueDetails{}, hostapi.WithScope(err, scope)
	}
	return toIssueDetails(is), nil
}
