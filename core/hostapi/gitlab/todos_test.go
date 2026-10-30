package gitlab

import (
	"context"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
)

func TestTodosReadTheToDoList(t *testing.T) {
	p, f := newProvider(t, map[string]string{
		"GET /api/v4/todos?per_page=100&state=pending": `[
			{"id": 101, "project": {"path_with_namespace": "gitlab-org/cli"}, "action_name": "mentioned", "target_type": "MergeRequest", "target": {"iid": 7, "title": "Add the thing"}, "target_url": "https://gitlab.example.com/gitlab-org/cli/-/merge_requests/7", "author": {"username": "bob", "name": "Bob"}, "updated_at": "2026-10-20T10:00:00Z"},
			{"id": 102, "project": {"path_with_namespace": "gitlab-org/cli"}, "action_name": "assigned", "target_type": "Issue", "target": {"iid": 12, "title": "Login fails"}, "target_url": "https://gitlab.example.com/gitlab-org/cli/-/issues/12", "author": {"username": "bob", "name": "Bob"}, "updated_at": "2026-10-20T10:00:00Z"},
			{"id": 103, "project": {"path_with_namespace": "other/repo"}, "action_name": "build_failed", "target_type": "MergeRequest", "target": {"iid": 3, "title": "Elsewhere"}, "target_url": "https://gitlab.example.com/other/repo/-/merge_requests/3", "author": {"username": "bob", "name": "Bob"}, "updated_at": "2026-10-20T10:00:00Z"},
			{"id": 104, "project": {"path_with_namespace": "gitlab-org/cli"}, "action_name": "directly_addressed", "target_type": "DesignManagement::Design", "target": {"title": "A design"}, "target_url": "https://gitlab.example.com/gitlab-org/cli/-/designs/1", "author": {"username": "bob", "name": "Bob"}, "updated_at": "2026-10-20T10:00:00Z"}
		]`,
		"POST /api/v4/todos/101/mark_as_done": `{"id": 101, "state": "done"}`,
	})
	todos, err := p.Todos(context.Background())
	if err != nil || len(todos) != 4 {
		t.Fatalf("Todos = %+v, %v", todos, err)
	}
	want := hostapi.Todo{ID: "101", Project: "gitlab-org/cli", Reason: hostapi.TodoMentioned, Target: "change", Number: 7, Title: "Add the thing", Author: hostapi.User{Name: "Bob", Username: "bob"}, WebURL: "https://gitlab.example.com/gitlab-org/cli/-/merge_requests/7", UpdatedAt: todos[0].UpdatedAt}
	if todos[0] != want {
		t.Errorf("mention = %+v, want %+v", todos[0], want)
	}
	if todos[1].Reason != hostapi.TodoAssigned || todos[1].Target != "issue" || todos[1].Number != 12 {
		t.Errorf("assignment = %+v", todos[1])
	}
	if todos[2].Reason != "pipeline failed" || todos[2].Project != "other/repo" {
		t.Errorf("other project = %+v", todos[2])
	}
	if todos[3].Target != "" || todos[3].Reason != hostapi.TodoMentioned {
		t.Errorf("a design has no target kind: %+v", todos[3])
	}
	if err := p.MarkTodoDone(context.Background(), "101"); err != nil {
		t.Fatal(err)
	}
	if last := f.requests[len(f.requests)-1]; last.Method != "POST" || last.Path != "/api/v4/todos/101/mark_as_done" {
		t.Errorf("mark done sent %+v", last)
	}
}
