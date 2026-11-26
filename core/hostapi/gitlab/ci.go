package gitlab

import (
	"context"
	"net/url"
	"strconv"
	"time"

	"github.com/gspeager/current-client/core/hostapi"
)

func (p *Provider) LatestPipeline(ctx context.Context, project, sha string) (hostapi.Pipeline, bool, error) {
	var pipelines []pipeline
	err := p.get(ctx, readScope, projectPath(project, "pipelines"), url.Values{"sha": {sha}, "per_page": {"1"}}, &pipelines)
	if err != nil || len(pipelines) == 0 {
		return hostapi.Pipeline{}, false, err
	}
	return toPipeline(pipelines[0]), true, nil
}

func (p *Provider) Jobs(ctx context.Context, project string, pipelineID int64) ([]hostapi.Job, error) {
	var jobs []job
	err := p.get(ctx, readScope, projectPath(project, "pipelines", strconv.FormatInt(pipelineID, 10), "jobs"), url.Values{"per_page": {"100"}}, &jobs)
	if err != nil {
		return nil, err
	}
	out := make([]hostapi.Job, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, hostapi.Job{
			ID:        j.ID,
			Name:      j.Name,
			Stage:     j.Stage,
			Status:    ciStatus(j.Status),
			Duration:  time.Duration(j.Duration * float64(time.Second)),
			WebURL:    j.WebURL,
			Retryable: hostapi.Retryable(ciStatus(j.Status)),
		})
	}
	return out, nil
}

func (p *Provider) JobLog(ctx context.Context, project string, jobID int64, maxBytes int) (string, bool, error) {
	resp, err := p.getBody(ctx, readScope, projectPath(project, "jobs", strconv.FormatInt(jobID, 10), "trace"), nil)
	if err != nil {
		return "", false, err
	}
	log, truncated := hostapi.TailLog(resp.Body, maxBytes)
	return log, truncated, nil
}

func toPipeline(pl pipeline) hostapi.Pipeline {
	return hostapi.Pipeline{ID: pl.ID, Status: ciStatus(pl.Status), WebURL: pl.WebURL}
}

func ciStatus(status string) hostapi.CIStatus {
	switch status {
	case "success":
		return hostapi.CISuccess
	case "failed":
		return hostapi.CIFailed
	case "running":
		return hostapi.CIRunning
	case "canceled", "canceling":
		return hostapi.CICanceled
	case "skipped":
		return hostapi.CISkipped
	case "pending", "created", "waiting_for_resource", "preparing", "scheduled", "manual":
		return hostapi.CIPending
	}
	return hostapi.CIUnknown
}
