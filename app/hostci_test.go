//go:build !nohost

package app

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/gspeager/current-client/core/hostapi/hostapitest"
	"github.com/gspeager/current-client/internal/config"
)

// routeTransport answers by URL path, counting requests per path.
type routeTransport struct {
	mu     sync.Mutex
	routes map[string]string
	hits   map[string]int
}

func (r *routeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := req.URL.EscapedPath()
	if req.URL.RawQuery != "" {
		key += "?" + req.URL.RawQuery
	}
	r.hits[key]++
	body, ok := r.routes[key]
	status := http.StatusOK
	switch {
	case !ok:
		body, status = `{"message":"404 Not Found"}`, http.StatusNotFound
	case body == "403":
		body, status = `{"message":"403 Forbidden"}`, http.StatusForbidden
	}
	return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
}

func enabledGitLab(t *testing.T) (repo string) {
	t.Helper()
	useTempSettings(t)
	repo = t.TempDir()
	gitIn(t, repo, "init", "-q")
	gitIn(t, repo, "remote", "add", "origin", "git@gitlab.example.com:group/app.git")
	if err := updateConfig(func(cfg *config.Config) {
		cfg.Hosts = map[string]config.HostConfig{"gitlab.example.com": {Enabled: true}}
	}); err != nil {
		t.Fatal(err)
	}
	return repo
}

func TestFetchCIStatusAsksOncePerSHAAndRemembersNone(t *testing.T) {
	repo := enabledGitLab(t)
	api := "/api/v4/projects/group%2Fapp"
	transport := &routeTransport{routes: map[string]string{
		api + "/pipelines?per_page=1&sha=aaa":   `[{"id": 11, "status": "success", "web_url": "https://gitlab.example.com/group/app/-/pipelines/11"}]`,
		api + "/pipelines?per_page=1&sha=bbb":   `[{"id": 12, "status": "failed", "web_url": "https://gitlab.example.com/group/app/-/pipelines/12"}]`,
		api + "/pipelines?per_page=1&sha=ccc":   `[]`,
		api + "/pipelines/12/jobs?per_page=100": `[{"id": 5, "name": "test", "stage": "test", "status": "failed", "duration": 42.5, "web_url": "https://gitlab.example.com/group/app/-/jobs/5"}]`,
	}, hits: map[string]int{}}
	swapHostClient(t, transport)
	var s HostService

	marks, err := s.FetchCIStatus(repo, []string{"aaa", "bbb", "ccc"})
	if err != nil {
		t.Fatal(err)
	}
	if marks["aaa"].Status != "success" || marks["bbb"].Status != "failed" || marks["bbb"].PipelineID != 12 || marks["ccc"].Status != "" {
		t.Fatalf("FetchCIStatus = %+v", marks)
	}
	if _, ok := marks["ccc"]; ok {
		t.Fatal("a commit without a pipeline should have no mark")
	}

	again, _ := s.FetchCIStatus(repo, []string{"aaa", "bbb", "ccc", "aaa"})
	if len(again) != 2 {
		t.Fatalf("second answer = %+v", again)
	}
	for _, sha := range []string{"aaa", "bbb", "ccc"} {
		if n := transport.hits[api+"/pipelines?per_page=1&sha="+sha]; n != 1 {
			t.Errorf("%s asked for %d times, want once", sha, n)
		}
	}

	cached, _ := s.CIStatus(repo, []string{"aaa", "zzz"})
	if len(cached) != 1 || cached["aaa"].Status != "success" || transport.hits[api+"/pipelines?per_page=1&sha=zzz"] != 0 {
		t.Fatalf("CIStatus should read the cache only: %+v, hits %v", cached, transport.hits)
	}

	jobs, err := s.PipelineJobs(repo, 12)
	if err != nil || len(jobs) != 1 || jobs[0].Name != "test" || jobs[0].Duration.Seconds() != 42.5 {
		t.Fatalf("PipelineJobs = %+v, %v", jobs, err)
	}
	s.PipelineJobs(repo, 12)
	if n := transport.hits[api+"/pipelines/12/jobs?per_page=100"]; n != 1 {
		t.Fatalf("jobs fetched %d times, want once", n)
	}

	if err := s.Refresh(repo); err == nil {
		t.Fatal("refresh against a transport without the merge request route should report the failure")
	}
	s.FetchCIStatus(repo, []string{"aaa"})
	if n := transport.hits[api+"/pipelines?per_page=1&sha=aaa"]; n != 2 {
		t.Fatalf("after a refresh the SHA should be asked for again, got %d", n)
	}
}

func TestCIStatusIsEmptyWhileTheHostIsOff(t *testing.T) {
	useTempSettings(t)
	swapHostClient(t, hostapitest.Forbid(t))
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q")
	gitIn(t, repo, "remote", "add", "origin", "git@gitlab.example.com:group/app.git")
	var s HostService
	marks, err := s.FetchCIStatus(repo, []string{"aaa"})
	if err != nil || len(marks) != 0 {
		t.Fatalf("FetchCIStatus while off = %+v, %v", marks, err)
	}
}
