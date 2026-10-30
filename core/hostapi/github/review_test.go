package github

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
)

func TestReviewWritesSendWhatGitHubExpects(t *testing.T) {
	p, f := newProvider(t, map[string]string{
		"POST " + v3 + "/pulls/7/comments/55/replies": `{"id": 56}`,
		"POST " + v3 + "/pulls/7/comments":            `{"id": 57}`,
		"POST " + v3 + "/pulls/7/reviews":             `{"id": 9}`,
	})
	ctx := context.Background()
	if err := p.Reply(ctx, project, 7, "55", "Agreed."); err != nil {
		t.Fatal(err)
	}
	if err := p.CommentOnLine(ctx, project, 7, "abc", hostapi.NewComment{Path: "app/x.go", Line: 12, Body: "Rename this."}); err != nil {
		t.Fatal(err)
	}
	err := p.SubmitReview(ctx, project, 7, "abc", hostapi.ReviewSubmission{
		Verdict: hostapi.ReviewRequestChanges, Summary: "Two things.",
		Comments: []hostapi.NewComment{{Path: "app/y.go", OldLine: 3, Body: "Why remove this?"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var reply, comment, review map[string]any
	json.Unmarshal([]byte(f.requests[0].Body), &reply)
	json.Unmarshal([]byte(f.requests[1].Body), &comment)
	json.Unmarshal([]byte(f.requests[2].Body), &review)
	if reply["body"] != "Agreed." {
		t.Errorf("reply = %v", reply)
	}
	if want := map[string]any{"path": "app/x.go", "body": "Rename this.", "line": 12.0, "side": "RIGHT", "commit_id": "abc"}; !reflect.DeepEqual(comment, want) {
		t.Errorf("comment = %v, want %v", comment, want)
	}
	if review["event"] != "REQUEST_CHANGES" || review["body"] != "Two things." || review["commit_id"] != "abc" {
		t.Errorf("review = %v", review)
	}
	if comments := review["comments"].([]any); len(comments) != 1 || comments[0].(map[string]any)["side"] != "LEFT" || comments[0].(map[string]any)["line"] != 3.0 {
		t.Errorf("review comments = %v", review["comments"])
	}
	if err := p.ResolveThread(ctx, project, 7, "55", true); !errors.Is(err, hostapi.ErrReviewUnsupported) {
		t.Errorf("resolve = %v, want unsupported", err)
	}
	if caps := p.ReviewCapabilities(); caps.Resolve || !caps.RequestChanges || !caps.Reply || !caps.LineComments || !caps.Approve {
		t.Errorf("capabilities = %+v", caps)
	}
}
