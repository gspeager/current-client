package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
)

func TestReviewWritesSendWhatGitLabExpects(t *testing.T) {
	mr := api + "/merge_requests/7"
	p, f := newProvider(t, map[string]string{
		"GET " + mr:                            `{"iid": 7, "diff_refs": {"base_sha": "b", "head_sha": "h", "start_sha": "s"}}`,
		"POST " + mr + "/discussions/d1/notes": `{"id": 1}`,
		"PUT " + mr + "/discussions/d1":        `{"id": "d1"}`,
		"POST " + mr + "/discussions":          `{"id": "d2"}`,
		"POST " + mr + "/notes":                `{"id": 2}`,
		"POST " + mr + "/approve":              `{"approved": true}`,
	})
	ctx := context.Background()
	if err := p.Reply(ctx, project, 7, "d1", "Agreed."); err != nil {
		t.Fatal(err)
	}
	if err := p.ResolveThread(ctx, project, 7, "d1", true); err != nil {
		t.Fatal(err)
	}
	if err := p.CommentOnLine(ctx, project, 7, "h", hostapi.NewComment{Path: "app/x.go", Line: 12, Body: "Rename this."}); err != nil {
		t.Fatal(err)
	}
	if err := p.CommentOnLine(ctx, project, 7, "stale", hostapi.NewComment{Path: "app/x.go", Line: 12, Body: "Late."}); err == nil || !strings.Contains(err.Error(), "moved since") {
		t.Errorf("a stale head is refused: %v", err)
	}
	err := p.SubmitReview(ctx, project, 7, "h", hostapi.ReviewSubmission{
		Verdict: hostapi.ReviewApprove, Summary: "Looks good.",
		Comments: []hostapi.NewComment{{Path: "app/y.go", OldLine: 3, Body: "Why remove this?"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var resolve, comment map[string]any
	json.Unmarshal([]byte(f.requests[1].Body), &resolve)
	json.Unmarshal([]byte(f.requests[3].Body), &comment)
	if resolve["resolved"] != true {
		t.Errorf("resolve = %v", resolve)
	}
	position := comment["position"].(map[string]any)
	if position["new_line"] != 12.0 || position["head_sha"] != "h" || position["base_sha"] != "b" || position["start_sha"] != "s" || position["new_path"] != "app/x.go" {
		t.Errorf("position = %v", position)
	}
	// The review: diff refs, the comment (old side), the summary note, the approval.
	tail := f.requests[len(f.requests)-4:]
	if tail[0].Method != "GET" || tail[1].Path != mr+"/discussions" || tail[2].Path != mr+"/notes" || tail[3].Path != mr+"/approve" {
		for _, r := range tail {
			t.Logf("%s %s", r.Method, r.Path)
		}
		t.Error("review requests in the wrong order")
	}
	var old map[string]any
	json.Unmarshal([]byte(tail[1].Body), &old)
	if old["position"].(map[string]any)["old_line"] != 3.0 {
		t.Errorf("old-side comment = %v", old)
	}
	if err := p.SubmitReview(ctx, project, 7, "h", hostapi.ReviewSubmission{Verdict: hostapi.ReviewRequestChanges}); !errors.Is(err, hostapi.ErrReviewUnsupported) {
		t.Errorf("request changes = %v, want unsupported", err)
	}
	if caps := p.ReviewCapabilities(); !caps.Resolve || caps.RequestChanges || !caps.Approve {
		t.Errorf("capabilities = %+v", caps)
	}
}
