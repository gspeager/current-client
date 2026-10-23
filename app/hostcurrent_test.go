//go:build !nohost

package app

import (
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostapi/hostapitest"
	"github.com/gspeager/current-client/internal/config"
)

func TestCurrentChangeReadsTheCheckedOutBranchFromTheCache(t *testing.T) {
	useTempSettings(t)
	swapHostClient(t, hostapitest.Forbid(t))
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "feature")
	gitIn(t, repo, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "first")
	gitIn(t, repo, "remote", "add", "origin", "git@gitlab.example.com:group/app.git")
	gitIn(t, repo, "update-ref", "refs/remotes/origin/feature", "HEAD")
	gitIn(t, repo, "branch", "--set-upstream-to=origin/feature", "feature")
	var s HostService

	off, err := s.CurrentChange(repo)
	if err != nil || off.Enabled || off.Branch != "feature" || off.Upstream != "origin/feature" {
		t.Fatalf("with the host off: %+v, %v", off, err)
	}

	if err := updateConfig(func(cfg *config.Config) {
		cfg.Hosts = map[string]config.HostConfig{"gitlab.example.com": {Provider: "gitlab", Enabled: true}}
	}); err != nil {
		t.Fatal(err)
	}
	none, _ := s.CurrentChange(repo)
	if !none.Enabled || none.Change != nil || none.Kind != "merge request" {
		t.Fatalf("enabled, nothing cached: %+v", none)
	}

	hostClient.Cache.Put("gitlab.example.com", repo, cacheChanges, "group/app", cachedChanges{changes: []hostapi.Change{
		{Number: 7, Title: "Add the thing", State: hostapi.ChangeOpen, SourceBranch: "feature", TargetBranch: "main", HeadSHA: "abc", CI: hostapi.CIFailed},
	}})
	hostClient.Cache.Put("gitlab.example.com", repo, cachePipeline, "abc", CIMark{Status: "failed", PipelineID: 12, WebURL: "https://gitlab.example.com/p/12"})
	hostClient.Cache.Put("gitlab.example.com", repo, cacheJobs, "12", []hostapi.Job{{Name: "test", Status: hostapi.CIFailed}, {Name: "build", Status: hostapi.CISuccess}})

	got, err := s.CurrentChange(repo)
	if err != nil || got.Change == nil || got.Change.Number != 7 || got.Change.TargetBranch != "main" {
		t.Fatalf("CurrentChange = %+v, %v", got, err)
	}
	if got.CI == nil || got.CI.Status != "failed" || len(got.FailedJobs) != 1 || got.FailedJobs[0] != "test" {
		t.Fatalf("CI of the current change = %+v, jobs %v", got.CI, got.FailedJobs)
	}
}
