package github

import (
	"context"
	"errors"
	"regexp"
	"strconv"

	"github.com/gspeager/current-client/core/hostapi"
)

var _ hostapi.CIRetrier = (*Provider)(nil)

// actionsApp is the check-run app GitHub Actions reports as; only its runs
// can be re-run through the API.
const actionsApp = "github-actions"

var runInURL = regexp.MustCompile(`/actions/runs/(\d+)/job/\d+`)

var errNotActions = errors.New("only GitHub Actions jobs can be re-run from the app; this check belongs to another app")

func (p *Provider) RetryJob(ctx context.Context, project string, job hostapi.Job) error {
	if !job.Retryable {
		return errNotActions
	}
	path := repoPath(project, "actions", "jobs", strconv.FormatInt(job.ID, 10), "rerun")
	return hostapi.WithScope(p.client.PostJSON(ctx, p.host, path, map[string]any{}, nil), scope)
}

// RetryPipeline re-runs the failed jobs of each workflow run that has one,
// found from the jobs' addresses: GitHub re-runs by run, and re-running one
// job of a run blocks the next until the run ends.
func (p *Provider) RetryPipeline(ctx context.Context, project string, _ int64, jobs []hostapi.Job) error {
	asked := map[string]bool{}
	for _, job := range jobs {
		m := runInURL.FindStringSubmatch(job.WebURL)
		if !job.Retryable || m == nil || asked[m[1]] {
			continue
		}
		asked[m[1]] = true
		path := repoPath(project, "actions", "runs", m[1], "rerun-failed-jobs")
		if err := p.client.PostJSON(ctx, p.host, path, map[string]any{}, nil); err != nil {
			return hostapi.WithScope(err, scope)
		}
	}
	if len(asked) == 0 {
		return errNotActions
	}
	return nil
}
