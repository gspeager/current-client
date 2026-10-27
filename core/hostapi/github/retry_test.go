package github

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
)

func TestRetryReRunsActionsJobsByRun(t *testing.T) {
	p, f := newProvider(t, map[string]string{
		"POST " + v3 + "/actions/jobs/11/rerun":            `{}`,
		"POST " + v3 + "/actions/runs/5/rerun-failed-jobs": `{}`,
		"POST " + v3 + "/actions/runs/6/rerun-failed-jobs": `{}`,
	})
	ctx := context.Background()
	job := func(id int64, run string, retryable bool) hostapi.Job {
		return hostapi.Job{ID: id, Retryable: retryable, WebURL: "https://github.com/cli/cli/actions/runs/" + run + "/job/" + strconv.FormatInt(id, 10)}
	}
	if err := p.RetryJob(ctx, project, job(11, "5", true)); err != nil {
		t.Fatal(err)
	}
	jobs := []hostapi.Job{job(11, "5", true), job(12, "5", true), job(13, "6", true), job(14, "7", false)}
	if err := p.RetryPipeline(ctx, project, 0, jobs); err != nil {
		t.Fatal(err)
	}
	if len(f.requests) != 3 || f.count("POST", v3+"/actions/runs/5/rerun-failed-jobs") != 1 || f.count("POST", v3+"/actions/runs/6/rerun-failed-jobs") != 1 {
		t.Fatalf("requests = %+v", f.requests)
	}
	status := hostapi.Job{ID: 7, Stage: "status", Status: hostapi.CIFailed, WebURL: "https://circleci.example.com/1"}
	if err := p.RetryJob(ctx, project, status); !errors.Is(err, errNotActions) {
		t.Errorf("a commit status was sent: %v", err)
	}
	if err := p.RetryPipeline(ctx, project, 0, []hostapi.Job{status}); !errors.Is(err, errNotActions) {
		t.Errorf("a pipeline with no Actions job: %v", err)
	}
}

func TestJobsMarkFailedActionsRunsRetryable(t *testing.T) {
	sha := "654d851d8145b2912fe474ceb0a47681854b37e2"
	p, _ := newProvider(t, map[string]string{
		"GET " + v3 + "/commits/" + sha + "/check-runs?per_page=100": `{"check_runs": [
			{"id": 1, "status": "completed", "conclusion": "failure", "app": {"slug": "github-actions"}},
			{"id": 2, "status": "completed", "conclusion": "success", "app": {"slug": "github-actions"}},
			{"id": 3, "status": "completed", "conclusion": "failure", "app": {"slug": "codecov"}}]}`,
		"GET " + v3 + "/commits/" + sha + "/status": `{"statuses": [{"id": 4, "state": "failure"}]}`,
	})
	jobs, err := p.commitJobs(context.Background(), project, sha)
	if err != nil || len(jobs) != 4 {
		t.Fatalf("jobs = %+v, %v", jobs, err)
	}
	for i, want := range []bool{true, false, false, false} {
		if jobs[i].Retryable != want {
			t.Errorf("job %d retryable = %v, want %v", jobs[i].ID, jobs[i].Retryable, want)
		}
	}
}
