package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
)

func TestIssueWritesSendWhatGitLabExpects(t *testing.T) {
	p, f := newProvider(t, map[string]string{
		"POST " + api + "/issues":          `{"iid": 31, "title": "Flaky test", "state": "opened", "web_url": "https://gitlab.com/gitlab-org/cli/-/issues/31"}`,
		"POST " + api + "/issues/31/notes": `{"author": {"username": "dev"}, "body": "Seen again."}`,
		"PUT " + api + "/issues/31":        `{"iid": 31, "title": "Flaky test", "state": "closed", "labels": ["bug"]}`,
	})
	ctx := context.Background()
	created, err := p.CreateIssue(ctx, project, hostapi.NewIssue{Title: "Flaky test", Description: "From x.go", Labels: []string{"bug", "ci"}})
	if err != nil || created.Number != 31 || created.State != "open" || created.WebURL == "" {
		t.Fatalf("CreateIssue = %+v, %v", created, err)
	}
	note, err := p.CommentOnIssue(ctx, project, "31", "Seen again.")
	if err != nil || note.Author.Username != "dev" || note.Body != "Seen again." {
		t.Fatalf("CommentOnIssue = %+v, %v", note, err)
	}
	reopened, err := p.SetIssueState(ctx, project, "31", true)
	if err != nil || reopened.Number != 31 {
		t.Fatalf("SetIssueState = %+v, %v", reopened, err)
	}
	var create, comment, state map[string]any
	json.Unmarshal([]byte(f.requests[0].Body), &create)
	json.Unmarshal([]byte(f.requests[1].Body), &comment)
	json.Unmarshal([]byte(f.requests[2].Body), &state)
	if want := map[string]any{"title": "Flaky test", "description": "From x.go", "labels": "bug,ci"}; !reflect.DeepEqual(create, want) {
		t.Errorf("create = %v, want %v", create, want)
	}
	if comment["body"] != "Seen again." || state["state_event"] != "reopen" {
		t.Errorf("comment = %v, state = %v", comment, state)
	}
}

func TestIssueWriteRefusalNamesTheScope(t *testing.T) {
	p, _ := newProvider(t, map[string]string{"POST " + api + "/issues": "403"})
	_, err := p.CreateIssue(context.Background(), project, hostapi.NewIssue{Title: "x"})
	var hostErr *hostapi.HostError
	if !errors.As(err, &hostErr) || hostErr.Scope != writeScope {
		t.Fatalf("err = %v", err)
	}
}
