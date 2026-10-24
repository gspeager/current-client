//go:build !nohost

package app

import (
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostapi/hostapitest"
	"github.com/gspeager/current-client/internal/config"
)

func TestNoticesNameWhatARefreshChangedOnTheBranch(t *testing.T) {
	useTempSettings(t)
	swapHostClient(t, hostapitest.Forbid(t))
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "feature")
	gitIn(t, repo, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "first")
	gitIn(t, repo, "remote", "add", "origin", "git@gitlab.example.com:group/app.git")
	gitIn(t, repo, "update-ref", "refs/remotes/origin/feature", "HEAD")
	gitIn(t, repo, "branch", "--set-upstream-to=origin/feature", "feature")
	if err := updateConfig(func(cfg *config.Config) {
		cfg.Hosts = map[string]config.HostConfig{"gitlab.example.com": {Provider: "gitlab", Enabled: true}}
	}); err != nil {
		t.Fatal(err)
	}
	hosts := configOrDefault().Hosts
	info := infoFor(hostByName("gitlab.example.com"), hosts)
	host, project := "gitlab.example.com", "group/app"
	put := func(state hostapi.ChangeState, ci hostapi.CIStatus, threads []hostapi.Thread) {
		hostClient.Cache.Put(host, repo, cacheChanges, project, cachedChanges{changes: []hostapi.Change{
			{Number: 7, SourceBranch: "feature", TargetBranch: "main", State: state, CI: ci, HeadSHA: "abc"},
		}})
		hostClient.Cache.Put(host, repo, cacheThreads, "7", threads)
	}
	note := hostapi.Note{Body: "x"}

	// Nothing known before the first refresh: no notices.
	before := snapshotCurrentBranch(info, repo)
	put(hostapi.ChangeOpen, hostapi.CISuccess, []hostapi.Thread{{ID: "d1", Path: "app/x.go", Notes: []hostapi.Note{note}}})
	if got := noticesSince(before, info, repo); len(got) != 0 {
		t.Fatalf("first refresh gave %+v", got)
	}

	// An unchanged answer: nothing.
	before = snapshotCurrentBranch(info, repo)
	if got := noticesSince(before, info, repo); len(got) != 0 {
		t.Fatalf("unchanged answer gave %+v", got)
	}

	// A new thread and a reply, CI turning to failed, and the change merged.
	before = snapshotCurrentBranch(info, repo)
	put(hostapi.ChangeMerged, hostapi.CIFailed, []hostapi.Thread{
		{ID: "d1", Path: "app/x.go", Notes: []hostapi.Note{note, note}},
		{ID: "d2", Path: "README.md", Notes: []hostapi.Note{note}},
	})
	hostClient.Cache.Put(host, repo, cachePipeline, "abc", CIMark{Status: "failed", PipelineID: 12})
	hostClient.Cache.Put(host, repo, cacheJobs, "12", []hostapi.Job{{Name: "test", Status: hostapi.CIFailed}})
	got := noticesSince(before, info, repo)
	if len(got) != 3 {
		t.Fatalf("notices = %+v", got)
	}
	if got[0].Kind != "threads" || got[0].Text != "2 new comments on !7" || got[0].Path != "app/x.go" {
		t.Errorf("threads notice = %+v", got[0])
	}
	if got[1].Kind != "ci" || got[1].Text != "Pipeline failed: test on !7" {
		t.Errorf("ci notice = %+v", got[1])
	}
	if got[2].Kind != "state" || got[2].Text != "!7 was merged" || got[2].Number != 7 {
		t.Errorf("state notice = %+v", got[2])
	}

	hostClient.Cache.Put(host, repo, cacheNotices, "current", got)
	var s HostService
	taken, err := s.TakeNotices(repo)
	if err != nil || len(taken) != 3 {
		t.Fatalf("TakeNotices = %+v, %v", taken, err)
	}
	if again, _ := s.TakeNotices(repo); len(again) != 0 {
		t.Fatalf("notices should be taken once, got %+v", again)
	}
}
