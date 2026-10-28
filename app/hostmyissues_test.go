//go:build !nohost

package app

import (
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostapi/hostapitest"
)

func TestMyIssuesComeFromTheCacheWithTheirBranch(t *testing.T) {
	repo := enabledGitLab(t)
	gitIn(t, repo, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "first")
	gitIn(t, repo, "branch", "-q", "12-fix-login")
	swapHostClient(t, hostapitest.Forbid(t))
	_, info, _, err := primaryHost(repo)
	if err != nil {
		t.Fatal(err)
	}
	var host HostService

	before := host.MyIssues(repo)
	if !before.Enabled || len(before.Sources) != 1 || before.Sources[0].FetchedAt != nil || len(before.Issues) != 0 {
		t.Fatalf("before a refresh: on, one source not yet read, no issues; got %+v", before)
	}

	hostClient.Cache.Put(info.Hostname, repo, cacheAssigned, info.Project, []hostapi.Issue{
		{Key: "12", Number: 12, Title: "Login fails"},
		{Key: "14", Number: 14, Title: "Slow start"},
	})
	list := host.MyIssues(repo)
	if list.Sources[0].Name != info.Hostname || list.Sources[0].FetchedAt == nil {
		t.Fatalf("source = %+v", list.Sources[0])
	}
	if len(list.Issues) != 2 || list.Issues[0].Branch != "12-fix-login" || list.Issues[1].Branch != "" || list.Issues[0].Source != info.Hostname {
		t.Fatalf("issues = %+v", list.Issues)
	}
}

func TestMyIssuesIsOffWithNoHostOrTracker(t *testing.T) {
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q")
	var host HostService
	if list := host.MyIssues(repo); list.Enabled || len(list.Issues) != 0 {
		t.Fatalf("no host, no tracker: off and empty, got %+v", list)
	}
}

func TestProjectIssuesAskOncePerStateUntilARefresh(t *testing.T) {
	repo := enabledGitLab(t)
	gitIn(t, repo, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "first")
	closed := "/api/v4/projects/group%2Fapp/issues?order_by=updated_at&per_page=50&state=closed"
	transport := &routeTransport{routes: map[string]string{
		closed: `[{"iid": 7, "title": "Old one", "state": "closed"}]`,
	}, hits: map[string]int{}}
	swapHostClient(t, transport)
	var host HostService
	for range 2 {
		list, err := host.ProjectIssues(repo, false)
		if err != nil || len(list.Issues) != 1 || list.Issues[0].Issue.Number != 7 || list.Sources[0].FetchedAt == nil {
			t.Fatalf("ProjectIssues = %+v, %v", list, err)
		}
	}
	if transport.hits[closed] != 1 {
		t.Fatalf("asked %d times, want once until a refresh", transport.hits[closed])
	}
}
