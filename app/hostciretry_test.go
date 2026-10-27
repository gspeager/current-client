//go:build !nohost

package app

import "testing"

func TestRetryReadsTheJobsAgainAndMarksThePipelineRunning(t *testing.T) {
	repo := enabledGitLab(t)
	api := "/api/v4/projects/group%2Fapp"
	transport := &routeTransport{routes: map[string]string{
		api + "/pipelines?per_page=1&sha=bbb":   `[{"id": 12, "status": "failed"}]`,
		api + "/pipelines/12/jobs?per_page=100": `[{"id": 5, "name": "test", "status": "failed"}]`,
		api + "/jobs/5/retry":                   `{"id": 6}`,
		api + "/pipelines/12/retry":             `{"id": 12}`,
	}, hits: map[string]int{}}
	swapHostClient(t, transport)
	var s HostService

	if _, err := s.RetryJob(repo, 12, 5); err == nil {
		t.Fatal("a retry before the jobs were read should be refused")
	}
	s.FetchCIStatus(repo, []string{"bbb"})
	if jobs, _ := s.PipelineJobs(repo, 12); !jobs[0].Retryable {
		t.Fatalf("a failed job should be retryable: %+v", jobs)
	}
	transport.routes[api+"/pipelines/12/jobs?per_page=100"] = `[{"id": 5, "name": "test", "status": "failed"}, {"id": 6, "name": "test", "status": "pending"}]`
	jobs, err := s.RetryJob(repo, 12, 5)
	if err != nil || len(jobs) != 2 || jobs[1].Status != "pending" {
		t.Fatalf("RetryJob = %+v, %v", jobs, err)
	}
	if marks, _ := s.CIStatus(repo, []string{"bbb"}); marks["bbb"].Status != "running" {
		t.Fatalf("the mark should read running after a retry: %+v", marks)
	}
	if cached, _ := s.PipelineJobs(repo, 12); len(cached) != 2 {
		t.Fatalf("the jobs read after the retry should be cached: %+v", cached)
	}
	if _, err := s.RetryPipeline(repo, 12); err != nil {
		t.Fatal(err)
	}
	if transport.hits[api+"/jobs/5/retry"] != 1 || transport.hits[api+"/pipelines/12/retry"] != 1 {
		t.Fatalf("hits = %v", transport.hits)
	}
	if _, err := s.RetryJob(repo, 12, 99); err == nil {
		t.Fatal("a job outside the list should be refused")
	}
}
