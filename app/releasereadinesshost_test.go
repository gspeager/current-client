//go:build !nohost

package app

import (
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostapi/hostapitest"
)

func TestReleaseReadinessHostRowsComeFromTheCache(t *testing.T) {
	repo := enabledGitLab(t)
	gitIn(t, repo, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "feat: first")
	swapHostClient(t, hostapitest.Forbid(t))
	_, info, _, _ := primaryHost(repo)
	sha := strings.TrimSpace(gitOut(t, repo, "rev-parse", "HEAD"))
	var s ChangelogService

	r, err := s.ReleaseReadiness(repo, "", "0.1.0", "main")
	if err != nil {
		t.Fatal(err)
	}
	if c := checkNamed(t, r, "CI on the candidate"); c.Level != "unknown" {
		t.Errorf("CI before a refresh = %+v", c)
	}
	hostClient.Cache.Put(info.Hostname, repo, cachePipeline, sha, CIMark{Status: "failed", PipelineID: 4})
	hostClient.Cache.Put(info.Hostname, repo, cacheRules, "all", []hostapi.BranchRule{{Pattern: "ma*", NoForcePush: true, NoDelete: true}})
	r, _ = s.ReleaseReadiness(repo, "", "0.1.0", "main")
	if c := checkNamed(t, r, "CI on the candidate"); c.Level != "warn" || c.Value != "Failed" || c.AsOf == nil {
		t.Errorf("CI = %+v", c)
	}
	if c := checkNamed(t, r, "Protection on main"); c.Level != "ok" || c.Value != "Protected" || len(c.Evidence) != 3 || c.Evidence[0] != "by the rule ma*" || c.AsOf == nil {
		t.Errorf("protection = %+v", c)
	}
	r, _ = s.ReleaseReadiness(repo, "", "0.1.0", "release")
	if c := checkNamed(t, r, "Protection on release"); c.Level != "warn" {
		t.Errorf("unprotected target = %+v", c)
	}
}
