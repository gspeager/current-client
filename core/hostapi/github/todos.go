package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gspeager/current-client/core/hostapi"
)

var _ hostapi.TodoList = (*Provider)(nil)

// Notifications take a classic token with this scope (repo covers it); a
// fine-grained token can't read them at all.
const notificationsScope = "notifications"

type notification struct {
	ID        string    `json:"id"`
	Reason    string    `json:"reason"`
	UpdatedAt time.Time `json:"updated_at"`
	Subject   struct {
		Title string `json:"title"`
		URL   string `json:"url"`
		Type  string `json:"type"`
	} `json:"subject"`
	Repository struct {
		FullName string `json:"full_name"`
		HTMLURL  string `json:"html_url"`
	} `json:"repository"`
}

// Todos is the unread notifications, one page of 50. GitHub refuses the
// endpoint to a fine-grained token, which is ErrTodosUnavailable rather
// than a scope to add.
func (p *Provider) Todos(ctx context.Context) ([]hostapi.Todo, error) {
	var page []notification
	query := url.Values{"per_page": {"50"}}
	if err := p.client.GetJSON(ctx, p.host, "/notifications", query, &page); err != nil {
		var hostErr *hostapi.HostError
		if errors.As(err, &hostErr) && hostErr.Status == http.StatusForbidden {
			return nil, fmt.Errorf("%w: %s's notifications need a classic token with the notifications scope", hostapi.ErrTodosUnavailable, p.host.Name)
		}
		return nil, hostapi.WithScope(err, notificationsScope)
	}
	todos := make([]hostapi.Todo, 0, len(page))
	for _, n := range page {
		todos = append(todos, toTodo(n))
	}
	return todos, nil
}

func (p *Provider) MarkTodoDone(ctx context.Context, id string) error {
	return hostapi.WithScope(p.client.RequestJSON(ctx, p.host, http.MethodPatch, "/notifications/threads/"+url.PathEscape(id), map[string]any{}, nil), notificationsScope)
}

// toTodo reads the number from the subject's API address, its last segment,
// and builds the web address from the repository's, since the notification
// carries none.
func toTodo(n notification) hostapi.Todo {
	todo := hostapi.Todo{
		ID:        n.ID,
		Project:   n.Repository.FullName,
		Reason:    todoReason(n.Reason),
		Title:     n.Subject.Title,
		WebURL:    n.Repository.HTMLURL,
		UpdatedAt: n.UpdatedAt,
	}
	last := n.Subject.URL[strings.LastIndex(n.Subject.URL, "/")+1:]
	switch n.Subject.Type {
	case "PullRequest":
		todo.Target = "change"
		todo.Number, _ = strconv.Atoi(last)
		todo.WebURL = n.Repository.HTMLURL + "/pull/" + last
	case "Issue":
		todo.Target = "issue"
		todo.Number, _ = strconv.Atoi(last)
		todo.WebURL = n.Repository.HTMLURL + "/issues/" + last
	case "Commit":
		todo.WebURL = n.Repository.HTMLURL + "/commit/" + last
	}
	return todo
}

func todoReason(reason string) string {
	switch reason {
	case "assign":
		return hostapi.TodoAssigned
	case "mention", "team_mention":
		return hostapi.TodoMentioned
	case "review_requested":
		return hostapi.TodoReviewRequested
	case "comment":
		return "new comment"
	case "author":
		return "activity on yours"
	case "state_change":
		return "state changed"
	case "ci_activity":
		return "CI activity"
	case "subscribed", "manual":
		return "subscribed"
	case "security_alert":
		return "security alert"
	}
	return strings.ReplaceAll(reason, "_", " ")
}
