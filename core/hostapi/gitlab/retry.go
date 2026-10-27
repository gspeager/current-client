package gitlab

import (
	"context"
	"strconv"

	"github.com/gspeager/current-client/core/hostapi"
)

var _ hostapi.CIRetrier = (*Provider)(nil)

func (p *Provider) RetryJob(ctx context.Context, project string, job hostapi.Job) error {
	path := projectPath(project, "jobs", strconv.FormatInt(job.ID, 10), "retry")
	return hostapi.WithScope(p.client.PostJSON(ctx, p.host, path, map[string]any{}, nil), writeScope)
}

// RetryPipeline retries the pipeline's failed and canceled jobs in one request.
func (p *Provider) RetryPipeline(ctx context.Context, project string, pipelineID int64, _ []hostapi.Job) error {
	path := projectPath(project, "pipelines", strconv.FormatInt(pipelineID, 10), "retry")
	return hostapi.WithScope(p.client.PostJSON(ctx, p.host, path, map[string]any{}, nil), writeScope)
}
