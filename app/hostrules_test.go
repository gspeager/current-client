//go:build !nohost

package app

import (
	"context"
	"testing"

	"github.com/gspeager/current-client/core/hostapi/hostapitest"
	"github.com/gspeager/current-client/internal/config"
)

func TestPushWarningComesFromTheCachedRulesOnly(t *testing.T) {
	repo := enabledGitLab(t)
	gitIn(t, repo, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "first")
	for _, name := range []string{"main", "release/1.0", "feature"} {
		gitIn(t, repo, "update-ref", "refs/remotes/origin/"+name, "HEAD")
	}
	gitIn(t, repo, "branch", "-q", "-u", "origin/main", "main")
	api := "/api/v4/projects/group%2Fapp"
	transport := &routeTransport{routes: map[string]string{
		api + "/merge_requests?per_page=100&state=opened": `[]`,
		api + "/protected_branches?per_page=100": `[{"name": "main", "push_access_levels": [{"access_level": 0}], "allow_force_push": false},
			{"name": "release/*", "push_access_levels": [{"access_level": 40}], "allow_force_push": true}]`,
	}, hits: map[string]int{}}
	swapHostClient(t, transport)
	var remotes RemoteService

	warning, err := remotes.PushWarning(repo, "")
	if err != nil || warning != nil {
		t.Fatalf("before a refresh: %+v, %v", warning, err)
	}
	_, info, _, _ := primaryHost(repo)
	refreshBranchRules(context.Background(), providerFor(info), info, repo, "origin")

	warning, err = remotes.PushWarning(repo, "")
	if err != nil || warning == nil || *warning != (PushWarning{Host: "gitlab.example.com", Branch: "main", Pattern: "main", NoPush: true, NoForcePush: true, NoDelete: true}) {
		t.Fatalf("the current branch's upstream: %+v, %v", warning, err)
	}
	warning, err = remotes.PushWarning(repo, "origin/release/1.0")
	if err != nil || warning == nil || warning.Pattern != "release/*" || !warning.RestrictedPush || warning.NoForcePush {
		t.Fatalf("a wildcard rule: %+v, %v", warning, err)
	}
	warning, err = remotes.PushWarning(repo, "origin/feature")
	if err != nil || warning != nil {
		t.Fatalf("an unprotected branch: %+v, %v", warning, err)
	}
	if transport.hits[api+"/protected_branches?per_page=100"] != 1 {
		t.Fatalf("asked at push time: %v", transport.hits)
	}

	swapHostClient(t, hostapitest.Forbid(t))
	if err := updateConfig(func(cfg *config.Config) {
		cfg.Hosts = map[string]config.HostConfig{"gitlab.example.com": {Enabled: false}}
	}); err != nil {
		t.Fatal(err)
	}
	warning, err = remotes.PushWarning(repo, "")
	if err != nil || warning != nil {
		t.Fatalf("with the host off: %+v, %v", warning, err)
	}
}
