//go:build !nohost

package app

import (
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/internal/config"
)

// The same refresh, cache and warnings as GitLab, against GitHub-shaped
// answers: the provider is the only thing that differs.
func TestGitHubRefreshFillsChangesCIAndRules(t *testing.T) {
	useTempSettings(t)
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	gitIn(t, repo, "remote", "add", "origin", "git@github.com:owner/app.git")
	gitIn(t, repo, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "first")
	gitIn(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	gitIn(t, repo, "branch", "-q", "-u", "origin/main", "main")
	if err := updateConfig(func(cfg *config.Config) {
		cfg.Hosts = map[string]config.HostConfig{"github.com": {Enabled: true}}
	}); err != nil {
		t.Fatal(err)
	}
	api := "/repos/owner/app"
	sha := "654d851d8145b2912fe474ceb0a47681854b37e2"
	transport := &routeTransport{routes: map[string]string{
		api + "/pulls?per_page=100&state=open":               `[{"number": 5, "title": "Add login", "state": "open", "head": {"ref": "feature", "sha": "` + sha + `", "repo": {"full_name": "owner/app"}}, "base": {"ref": "main", "repo": {"full_name": "owner/app"}}, "user": {"login": "ada"}, "html_url": "https://github.com/owner/app/pull/5"}]`,
		api + "/commits/" + sha + "/check-runs?per_page=100": `{"total_count": 1, "check_runs": [{"id": 9, "name": "test", "status": "completed", "conclusion": "failure", "html_url": "https://github.com/owner/app/runs/9", "app": {"slug": "github-actions"}}]}`,
		api + "/commits/" + sha + "/status":                  `{"state": "success", "statuses": [{"id": 1, "context": "lint", "state": "success"}]}`,
		api + "/rules/branches/main":                         `[{"type": "pull_request"}, {"type": "deletion"}]`,
		"/user":                                              `{"login": "ada", "name": "Ada"}`,
	}, hits: map[string]int{}}
	swapHostClient(t, transport)
	var s HostService

	if err := s.Refresh(repo); err != nil {
		t.Fatal(err)
	}
	changes, err := s.BranchChanges(repo)
	if err != nil {
		t.Fatal(err)
	}
	got := changes["origin/feature"]
	if got.Number != 5 || got.Prefix != "#" || got.State != string(hostapi.ChangeOpen) || got.CI != string(hostapi.CIFailed) || got.Author != "ada" || got.HeadSHA != sha {
		t.Fatalf("origin/feature = %+v", got)
	}
	var remotes RemoteService
	warning, err := remotes.PushWarning(repo, "")
	if err != nil || warning == nil || !warning.NoPush || !warning.NoDelete || warning.NoForcePush {
		t.Fatalf("PushWarning = %+v, %v", warning, err)
	}
	if transport.hits[api+"/pulls?per_page=100&state=open"] != 1 || transport.hits[api+"/rules/branches/main"] != 1 {
		t.Fatalf("hits = %v", transport.hits)
	}
	if got, err := s.TestConnection("github.com"); err != nil || got != "Connected to https://api.github.com as ada." {
		t.Fatalf("TestConnection = %q, %v", got, err)
	}
}
