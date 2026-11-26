package github

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/gspeager/current-client/core/hostapi"
)

// LatestPipeline is the commit's check runs and commit statuses read as one
// result. GitHub has no pipeline id, so the id is the start of the SHA,
// which Jobs turns back into a commit reference.
func (p *Provider) LatestPipeline(ctx context.Context, project, sha string) (hostapi.Pipeline, bool, error) {
	jobs, err := p.commitJobs(ctx, project, sha)
	if err != nil || len(jobs) == 0 {
		return hostapi.Pipeline{}, false, err
	}
	statuses := make([]hostapi.CIStatus, 0, len(jobs))
	for _, job := range jobs {
		statuses = append(statuses, job.Status)
	}
	id, err := pipelineID(sha)
	if err != nil {
		return hostapi.Pipeline{}, false, err
	}
	return hostapi.Pipeline{ID: id, Status: combine(statuses), WebURL: p.webRoot() + project + "/commit/" + sha + "/checks"}, true, nil
}

// Jobs gets back the abbreviated commit the pipeline id was made from; the
// host resolves it like any short SHA.
func (p *Provider) Jobs(ctx context.Context, project string, pipelineID int64) ([]hostapi.Job, error) {
	return p.commitJobs(ctx, project, fmt.Sprintf("%0*x", pipelineIDChars, pipelineID))
}

func (p *Provider) commitJobs(ctx context.Context, project, ref string) ([]hostapi.Job, error) {
	var runs checkRuns
	if err := p.get(ctx, scope, repoPath(project, "commits", ref, "check-runs"), url.Values{"per_page": {"100"}}, &runs); err != nil {
		return nil, err
	}
	var status commitStatus
	if err := p.get(ctx, scope, repoPath(project, "commits", ref, "status"), nil, &status); err != nil {
		return nil, err
	}
	jobs := make([]hostapi.Job, 0, len(runs.CheckRuns)+len(status.Statuses))
	for _, run := range runs.CheckRuns {
		jobs = append(jobs, hostapi.Job{
			ID:        run.ID,
			Name:      run.Name,
			Stage:     run.App.Slug,
			Status:    checkStatus(run),
			Duration:  duration(run.StartedAt, run.CompletedAt),
			WebURL:    run.HTMLURL,
			Retryable: run.App.Slug == actionsApp && hostapi.Retryable(checkStatus(run)),
		})
	}
	for _, s := range status.Statuses {
		jobs = append(jobs, hostapi.Job{ID: s.ID, Name: s.Context, Stage: "status", Status: statusState(s.State), WebURL: s.TargetURL})
	}
	return jobs, nil
}

// JobLog reads an Actions job's log; a check run made by Actions has the
// job's id. The host answers with a redirect to the log's storage, which the
// client follows without the token.
func (p *Provider) JobLog(ctx context.Context, project string, jobID int64, maxBytes int) (string, bool, error) {
	resp, err := p.getBody(ctx, scope, repoPath(project, "actions", "jobs", strconv.FormatInt(jobID, 10), "logs"), nil)
	if err != nil {
		return "", false, err
	}
	log, truncated := hostapi.TailLog(resp.Body, maxBytes)
	return log, truncated, nil
}

func checkStatus(run checkRun) hostapi.CIStatus {
	if run.Status != "completed" {
		if run.Status == "in_progress" {
			return hostapi.CIRunning
		}
		return hostapi.CIPending
	}
	switch run.Conclusion {
	case "success":
		return hostapi.CISuccess
	case "failure", "timed_out", "action_required", "startup_failure":
		return hostapi.CIFailed
	case "cancelled":
		return hostapi.CICanceled
	case "skipped", "neutral", "stale":
		return hostapi.CISkipped
	}
	return hostapi.CIUnknown
}

func statusState(state string) hostapi.CIStatus {
	switch state {
	case "success":
		return hostapi.CISuccess
	case "failure", "error":
		return hostapi.CIFailed
	case "pending":
		return hostapi.CIPending
	}
	return hostapi.CIUnknown
}

// combine reads several results as one, the way the host's own merge box
// does: a failure first, then anything still going, then a cancellation.
func combine(statuses []hostapi.CIStatus) hostapi.CIStatus {
	seen := map[hostapi.CIStatus]bool{}
	for _, s := range statuses {
		seen[s] = true
	}
	for _, s := range []hostapi.CIStatus{hostapi.CIFailed, hostapi.CIRunning, hostapi.CIPending, hostapi.CICanceled, hostapi.CISuccess, hostapi.CISkipped} {
		if seen[s] {
			return s
		}
	}
	return hostapi.CIUnknown
}

func duration(started, completed *string) time.Duration {
	if started == nil || completed == nil {
		return 0
	}
	from, err1 := time.Parse(time.RFC3339, *started)
	to, err2 := time.Parse(time.RFC3339, *completed)
	if err1 != nil || err2 != nil || to.Before(from) {
		return 0
	}
	return to.Sub(from)
}

func pipelineID(sha string) (int64, error) {
	if len(sha) < pipelineIDChars {
		return 0, fmt.Errorf("%q is not a full commit id", sha)
	}
	return strconv.ParseInt(sha[:pipelineIDChars], 16, 64)
}
