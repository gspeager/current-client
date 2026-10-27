package github

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
)

func TestIssueWritesSendWhatGitHubExpects(t *testing.T) {
	p, f := newProvider(t, map[string]string{
		"POST " + v3 + "/issues":             `{"number": 31, "title": "Flaky test", "state": "open", "html_url": "https://github.com/cli/cli/issues/31"}`,
		"POST " + v3 + "/issues/31/comments": `{"user": {"login": "octo"}, "body": "Seen again."}`,
		"PATCH " + v3 + "/issues/31":         `{"number": 31, "title": "Flaky test", "state": "closed", "labels": [{"name": "bug"}]}`,
	})
	ctx := context.Background()
	created, err := p.CreateIssue(ctx, project, hostapi.NewIssue{Title: "Flaky test", Description: "From x.go", Labels: []string{"bug", "ci"}})
	if err != nil || created.Number != 31 || created.Key != "31" || created.WebURL == "" {
		t.Fatalf("CreateIssue = %+v, %v", created, err)
	}
	note, err := p.CommentOnIssue(ctx, project, "31", "Seen again.")
	if err != nil || note.Author.Username != "octo" || note.Body != "Seen again." {
		t.Fatalf("CommentOnIssue = %+v, %v", note, err)
	}
	closed, err := p.SetIssueState(ctx, project, "31", false)
	if err != nil || closed.State != "closed" || len(closed.Labels) != 1 {
		t.Fatalf("SetIssueState = %+v, %v", closed, err)
	}
	var create, comment, state map[string]any
	json.Unmarshal([]byte(f.requests[0].Body), &create)
	json.Unmarshal([]byte(f.requests[1].Body), &comment)
	json.Unmarshal([]byte(f.requests[2].Body), &state)
	if want := map[string]any{"title": "Flaky test", "body": "From x.go", "labels": []any{"bug", "ci"}}; !reflect.DeepEqual(create, want) {
		t.Errorf("create = %v, want %v", create, want)
	}
	if comment["body"] != "Seen again." || state["state"] != "closed" {
		t.Errorf("comment = %v, state = %v", comment, state)
	}
}

func TestIssueWriteRefusalNamesTheScope(t *testing.T) {
	p, _ := newProvider(t, map[string]string{"POST " + v3 + "/issues/31/comments": "403"})
	_, err := p.CommentOnIssue(context.Background(), project, "31", "Hi")
	var hostErr *hostapi.HostError
	if !errors.As(err, &hostErr) || hostErr.Scope != scope {
		t.Fatalf("err = %v", err)
	}
	if _, err := p.CommentOnIssue(context.Background(), project, "../31", "Hi"); err == nil {
		t.Error("a key that isn't a number was sent")
	}
}
