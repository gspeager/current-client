package gitlab

import (
	"context"
	"net/http"
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
	if err := p.client.PostJSON(ctx, p.host, projectPath(project, "issues"), body, &created); err != nil {
		return hostapi.Issue{}, hostapi.WithScope(err, writeScope)
	}
	return toIssue(created), nil
}

func (p *Provider) CommentOnIssue(ctx context.Context, project, key, body string) (hostapi.Note, error) {
	number, err := hostapi.IssueNumber(key)
	if err != nil {
		return hostapi.Note{}, err
	}
	var n issueNote
	if err := p.client.PostJSON(ctx, p.host, projectPath(project, "issues", strconv.Itoa(number), "notes"), map[string]string{"body": body}, &n); err != nil {
		return hostapi.Note{}, hostapi.WithScope(err, writeScope)
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
	if err := p.client.RequestJSON(ctx, p.host, http.MethodPut, projectPath(project, "issues", strconv.Itoa(number)), map[string]string{"state_event": event}, &is); err != nil {
		return hostapi.IssueDetails{}, hostapi.WithScope(err, writeScope)
	}
	return toIssueDetails(is), nil
}
