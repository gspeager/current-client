package github

import (
	"context"
	"errors"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
)

func TestTodosReadTheNotifications(t *testing.T) {
	p, f := newProvider(t, map[string]string{
		"GET /api/v3/notifications?per_page=50": `[
			{"id": "901", "reason": "mention", "updated_at": "2026-10-20T10:00:00Z", "subject": {"title": "Add the thing", "url": "https://api.github.com/repos/cli/cli/pulls/7", "type": "PullRequest"}, "repository": {"full_name": "cli/cli", "html_url": "https://github.com/cli/cli"}},
			{"id": "902", "reason": "assign", "updated_at": "2026-10-20T10:00:00Z", "subject": {"title": "Login fails", "url": "https://api.github.com/repos/cli/cli/issues/12", "type": "Issue"}, "repository": {"full_name": "cli/cli", "html_url": "https://github.com/cli/cli"}},
			{"id": "903", "reason": "ci_activity", "updated_at": "2026-10-20T10:00:00Z", "subject": {"title": "Nightly", "url": "https://api.github.com/repos/cli/cli/commits/abc123", "type": "Commit"}, "repository": {"full_name": "cli/cli", "html_url": "https://github.com/cli/cli"}},
			{"id": "904", "reason": "subscribed", "updated_at": "2026-10-20T10:00:00Z", "subject": {"title": "v2.0", "url": null, "type": "Release"}, "repository": {"full_name": "cli/cli", "html_url": "https://github.com/cli/cli"}}
		]`,
		"PATCH /api/v3/notifications/threads/901": `{}`,
	})
	todos, err := p.Todos(context.Background())
	if err != nil || len(todos) != 4 {
		t.Fatalf("Todos = %+v, %v", todos, err)
	}
	want := hostapi.Todo{ID: "901", Project: "cli/cli", Reason: hostapi.TodoMentioned, Target: "change", Number: 7, Title: "Add the thing", WebURL: "https://github.com/cli/cli/pull/7", UpdatedAt: todos[0].UpdatedAt}
	if todos[0] != want {
		t.Errorf("mention = %+v, want %+v", todos[0], want)
	}
	if todos[1].Reason != hostapi.TodoAssigned || todos[1].Target != "issue" || todos[1].Number != 12 || todos[1].WebURL != "https://github.com/cli/cli/issues/12" {
		t.Errorf("assignment = %+v", todos[1])
	}
	if todos[2].Target != "" || todos[2].Number != 0 || todos[2].WebURL != "https://github.com/cli/cli/commit/abc123" || todos[2].Reason != "CI activity" {
		t.Errorf("commit = %+v", todos[2])
	}
	if todos[3].Target != "" || todos[3].WebURL != "https://github.com/cli/cli" {
		t.Errorf("release = %+v", todos[3])
	}
	if err := p.MarkTodoDone(context.Background(), "901"); err != nil {
		t.Fatal(err)
	}
	if last := f.requests[len(f.requests)-1]; last.Method != "PATCH" || last.Path != "/api/v3/notifications/threads/901" {
		t.Errorf("mark done sent %+v", last)
	}
}

func TestTodosNeedAClassicToken(t *testing.T) {
	p, _ := newProvider(t, map[string]string{"GET /api/v3/notifications?per_page=50": "403"})
	_, err := p.Todos(context.Background())
	if !errors.Is(err, hostapi.ErrTodosUnavailable) {
		t.Fatalf("err = %v, want ErrTodosUnavailable", err)
	}
}
