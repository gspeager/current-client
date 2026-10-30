package gitlab

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gspeager/current-client/core/hostapi"
)

var _ hostapi.TodoList = (*Provider)(nil)

type todo struct {
	ID      int64 `json:"id"`
	Project struct {
		PathWithNamespace string `json:"path_with_namespace"`
	} `json:"project"`
	ActionName string `json:"action_name"`
	TargetType string `json:"target_type"`
	Target     struct {
		IID   int    `json:"iid"`
		Title string `json:"title"`
	} `json:"target"`
	TargetURL string    `json:"target_url"`
	Author    user      `json:"author"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Todos is the pending To-Do List, one page of 100, across every project
// the token sees.
func (p *Provider) Todos(ctx context.Context) ([]hostapi.Todo, error) {
	var page []todo
	query := url.Values{"state": {"pending"}, "per_page": {"100"}}
	if err := p.client.GetJSON(ctx, p.host, "/todos", query, &page); err != nil {
		return nil, hostapi.WithScope(err, readScope)
	}
	todos := make([]hostapi.Todo, 0, len(page))
	for _, td := range page {
		todos = append(todos, hostapi.Todo{
			ID:        strconv.FormatInt(td.ID, 10),
			Project:   td.Project.PathWithNamespace,
			Reason:    todoReason(td.ActionName),
			Target:    todoTarget(td.TargetType),
			Number:    td.Target.IID,
			Title:     td.Target.Title,
			Author:    toUser(td.Author),
			WebURL:    td.TargetURL,
			UpdatedAt: td.UpdatedAt,
		})
	}
	return todos, nil
}

func (p *Provider) MarkTodoDone(ctx context.Context, id string) error {
	return hostapi.WithScope(p.client.PostJSON(ctx, p.host, "/todos/"+url.PathEscape(id)+"/mark_as_done", map[string]any{}, nil), writeScope)
}

func todoReason(action string) string {
	switch action {
	case "assigned":
		return hostapi.TodoAssigned
	case "mentioned", "directly_addressed":
		return hostapi.TodoMentioned
	case "review_requested":
		return hostapi.TodoReviewRequested
	case "build_failed":
		return "pipeline failed"
	case "approval_required":
		return "approval required"
	case "unmergeable":
		return "can't be merged"
	case "marked":
		return "marked for later"
	case "review_submitted":
		return "review submitted"
	case "member_access_requested":
		return "access requested"
	}
	return strings.ReplaceAll(action, "_", " ")
}

func todoTarget(targetType string) string {
	switch targetType {
	case "MergeRequest":
		return "change"
	case "Issue":
		return "issue"
	}
	return ""
}
