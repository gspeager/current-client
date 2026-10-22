//go:build !nohost

package app

import (
	"testing"
)

func TestFetchIssueTitlesAsksOnceAndRemembersMissingOnes(t *testing.T) {
	repo := enabledGitLab(t)
	api := "/api/v4/projects/group%2Fapp"
	transport := &routeTransport{routes: map[string]string{
		api + "/issues/4": `{"iid": 4, "title": "Login fails", "state": "opened", "web_url": "https://gitlab.example.com/group/app/-/issues/4"}`,
		api + "/issues/5": `{"iid": 5, "title": "Old one", "state": "closed", "web_url": "https://gitlab.example.com/group/app/-/issues/5"}`,
	}, hits: map[string]int{}}
	swapHostClient(t, transport)
	var s HostService

	issues, err := s.FetchIssueTitles(repo, []int{4, 5, 6, 4})
	if err != nil {
		t.Fatal(err)
	}
	if issues[4].Title != "Login fails" || issues[4].State != "open" || issues[5].State != "closed" || !issues[6].Known || issues[6].Title != "" {
		t.Fatalf("FetchIssueTitles = %+v", issues)
	}
	s.FetchIssueTitles(repo, []int{4, 5, 6})
	for _, n := range []string{"4", "5", "6"} {
		if hits := transport.hits[api+"/issues/"+n]; hits != 1 {
			t.Errorf("issue %s asked for %d times, want once", n, hits)
		}
	}
	cached, _ := s.IssueTitles(repo, []int{4, 7})
	if len(cached) != 1 || transport.hits[api+"/issues/7"] != 0 {
		t.Fatalf("IssueTitles should read the cache only: %+v", cached)
	}
	if jobs, err := s.CachedPipelineJobs(repo, 99); err != nil || jobs != nil {
		t.Fatalf("CachedPipelineJobs without a click = %+v, %v", jobs, err)
	}
}
