package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gspeager/current-client/core/hostapi"
)

var _ hostapi.Reviewer = (*Provider)(nil)

// ReviewCapabilities: the REST API approves and unapproves, but has no
// request-changes verdict.
func (p *Provider) ReviewCapabilities() hostapi.ReviewCapabilities {
	return hostapi.ReviewCapabilities{Reply: true, Resolve: true, LineComments: true, Approve: true}
}

func (p *Provider) Reply(ctx context.Context, project string, number int, threadID, body string) error {
	path := projectPath(project, "merge_requests", strconv.Itoa(number), "discussions", url.PathEscape(threadID), "notes")
	return p.post(ctx, writeScope, path, map[string]string{"body": body}, nil)
}

func (p *Provider) ResolveThread(ctx context.Context, project string, number int, threadID string, resolved bool) error {
	path := projectPath(project, "merge_requests", strconv.Itoa(number), "discussions", url.PathEscape(threadID))
	return p.send(ctx, writeScope, http.MethodPut, path, map[string]bool{"resolved": resolved}, nil)
}

type diffRefs struct {
	BaseSHA  string `json:"base_sha"`
	HeadSHA  string `json:"head_sha"`
	StartSHA string `json:"start_sha"`
}

// CommentOnLine needs the merge request's diff refs for the position, read
// once per call; headSHA is checked against them so a comment never lands
// on a head the user didn't see.
func (p *Provider) CommentOnLine(ctx context.Context, project string, number int, headSHA string, comment hostapi.NewComment) error {
	refs, err := p.diffRefs(ctx, project, number, headSHA)
	if err != nil {
		return err
	}
	return p.comment(ctx, project, number, refs, comment)
}

func (p *Provider) diffRefs(ctx context.Context, project string, number int, headSHA string) (diffRefs, error) {
	var mr struct {
		DiffRefs diffRefs `json:"diff_refs"`
	}
	if err := p.get(ctx, readScope, projectPath(project, "merge_requests", strconv.Itoa(number)), nil, &mr); err != nil {
		return diffRefs{}, err
	}
	if headSHA != "" && mr.DiffRefs.HeadSHA != headSHA {
		return diffRefs{}, fmt.Errorf("%s moved since the last refresh; refresh and look again", p.host.Name)
	}
	return mr.DiffRefs, nil
}

func (p *Provider) comment(ctx context.Context, project string, number int, refs diffRefs, c hostapi.NewComment) error {
	position := map[string]any{
		"position_type": "text",
		"base_sha":      refs.BaseSHA,
		"head_sha":      refs.HeadSHA,
		"start_sha":     refs.StartSHA,
		"new_path":      c.Path,
		"old_path":      c.Path,
	}
	if c.Line > 0 {
		position["new_line"] = c.Line
	} else {
		position["old_line"] = c.OldLine
	}
	path := projectPath(project, "merge_requests", strconv.Itoa(number), "discussions")
	return p.post(ctx, writeScope, path, map[string]any{"body": c.Body, "position": position}, nil)
}

// SubmitReview has no single request on GitLab: each drafted comment is a
// discussion, the summary a note, and approving its own call.
func (p *Provider) SubmitReview(ctx context.Context, project string, number int, headSHA string, review hostapi.ReviewSubmission) error {
	if review.Verdict == hostapi.ReviewRequestChanges {
		return fmt.Errorf("%w: %s's REST API has no request-changes verdict; comment instead", hostapi.ErrReviewUnsupported, p.host.Name)
	}
	if len(review.Comments) > 0 {
		refs, err := p.diffRefs(ctx, project, number, headSHA)
		if err != nil {
			return err
		}
		for _, c := range review.Comments {
			if err := p.comment(ctx, project, number, refs, c); err != nil {
				return err
			}
		}
	}
	if review.Summary != "" {
		path := projectPath(project, "merge_requests", strconv.Itoa(number), "notes")
		if err := p.post(ctx, writeScope, path, map[string]string{"body": review.Summary}, nil); err != nil {
			return err
		}
	}
	if review.Verdict == hostapi.ReviewApprove {
		path := projectPath(project, "merge_requests", strconv.Itoa(number), "approve")
		return p.post(ctx, writeScope, path, map[string]any{}, nil)
	}
	return nil
}
