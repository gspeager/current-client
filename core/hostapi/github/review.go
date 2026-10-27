package github

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gspeager/current-client/core/hostapi"
)

var _ hostapi.Reviewer = (*Provider)(nil)

// ReviewCapabilities: the REST API has no way to resolve a review thread,
// which GitHub keeps for GraphQL.
func (p *Provider) ReviewCapabilities() hostapi.ReviewCapabilities {
	return hostapi.ReviewCapabilities{Reply: true, LineComments: true, Approve: true, RequestChanges: true}
}

// Reply answers a thread, whose id is its first comment's.
func (p *Provider) Reply(ctx context.Context, project string, number int, threadID, body string) error {
	return hostapi.WithScope(p.client.PostJSON(ctx, p.host, repoPath(project, "pulls", strconv.Itoa(number), "comments", threadID, "replies"), map[string]string{"body": body}, nil), scope)
}

func (p *Provider) ResolveThread(context.Context, string, int, string, bool) error {
	return fmt.Errorf("%w: %s's REST API can't resolve a thread; resolve it on the host", hostapi.ErrReviewUnsupported, p.host.Name)
}

func (p *Provider) CommentOnLine(ctx context.Context, project string, number int, headSHA string, comment hostapi.NewComment) error {
	body := reviewCommentBody(comment)
	body["commit_id"] = headSHA
	return hostapi.WithScope(p.client.PostJSON(ctx, p.host, repoPath(project, "pulls", strconv.Itoa(number), "comments"), body, nil), scope)
}

// SubmitReview sends the verdict, summary and drafted comments as one review.
func (p *Provider) SubmitReview(ctx context.Context, project string, number int, headSHA string, review hostapi.ReviewSubmission) error {
	event := map[string]string{hostapi.ReviewApprove: "APPROVE", hostapi.ReviewRequestChanges: "REQUEST_CHANGES", hostapi.ReviewComment: "COMMENT"}[review.Verdict]
	if event == "" {
		return fmt.Errorf("unknown review verdict %q", review.Verdict)
	}
	comments := make([]map[string]any, 0, len(review.Comments))
	for _, c := range review.Comments {
		comments = append(comments, reviewCommentBody(c))
	}
	body := map[string]any{"event": event, "body": review.Summary, "comments": comments}
	if headSHA != "" {
		body["commit_id"] = headSHA
	}
	return hostapi.WithScope(p.client.PostJSON(ctx, p.host, repoPath(project, "pulls", strconv.Itoa(number), "reviews"), body, nil), scope)
}

func reviewCommentBody(c hostapi.NewComment) map[string]any {
	body := map[string]any{"path": c.Path, "body": c.Body}
	if c.Line > 0 {
		body["line"], body["side"] = c.Line, "RIGHT"
	} else {
		body["line"], body["side"] = c.OldLine, "LEFT"
	}
	return body
}
