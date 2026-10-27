package gitlab

import (
	"context"
	"errors"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
)

func TestRetrySendsWhatGitLabExpects(t *testing.T) {
	p, f := newProvider(t, map[string]string{
		"POST " + api + "/jobs/31/retry":                 `{"id": 32}`,
		"POST " + api + "/pipelines/12/retry":            `{"id": 12}`,
		"GET " + api + "/pipelines/12/jobs?per_page=100": `[{"id": 31, "status": "failed"}, {"id": 33, "status": "success"}, {"id": 34, "status": "canceled"}]`,
	})
	ctx := context.Background()
	jobs, err := p.Jobs(ctx, project, 12)
	if err != nil || !jobs[0].Retryable || jobs[1].Retryable || !jobs[2].Retryable {
		t.Fatalf("Jobs = %+v, %v", jobs, err)
	}
	if err := p.RetryJob(ctx, project, jobs[0]); err != nil {
		t.Fatal(err)
	}
	if err := p.RetryPipeline(ctx, project, 12, jobs); err != nil {
		t.Fatal(err)
	}
	if f.count("POST", api+"/jobs/31/retry") != 1 || f.count("POST", api+"/pipelines/12/retry") != 1 {
		t.Fatalf("requests = %+v", f.requests)
	}
}

func TestRetryRefusalNamesTheScope(t *testing.T) {
	p, _ := newProvider(t, map[string]string{"POST " + api + "/pipelines/12/retry": "403"})
	err := p.RetryPipeline(context.Background(), project, 12, nil)
	var hostErr *hostapi.HostError
	if !errors.As(err, &hostErr) || hostErr.Scope != writeScope {
		t.Fatalf("err = %v", err)
	}
}
