//go:build !nohost

package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gspeager/current-client/internal/config"
)

func TestChangeDraftComesFromTheCommitsAndTheTemplate(t *testing.T) {
	useTempSettings(t)
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	gitIn(t, repo, "config", "user.name", "T")
	gitIn(t, repo, "config", "user.email", "t@example.com")
	gitIn(t, repo, "commit", "-q", "--allow-empty", "-m", "chore: start")
	gitIn(t, repo, "remote", "add", "origin", "git@gitlab.example.com:group/app.git")
	gitIn(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	gitIn(t, repo, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	gitIn(t, repo, "switch", "-q", "-c", "feature")
	gitIn(t, repo, "commit", "-q", "--allow-empty", "-m", "feat(login): add the login form")
	gitIn(t, repo, "commit", "-q", "--allow-empty", "-m", "fix(login): handle an empty password")
	os.MkdirAll(filepath.Join(repo, ".gitlab", "merge_request_templates"), 0o755)
	os.WriteFile(filepath.Join(repo, ".gitlab", "merge_request_templates", "Default.md"), []byte("## Checklist\n\n- [ ] Tests\n"), 0o644)
	if err := updateConfig(func(cfg *config.Config) {
		cfg.Hosts = map[string]config.HostConfig{"gitlab.example.com": {Provider: "gitlab", Enabled: true}}
	}); err != nil {
		t.Fatal(err)
	}
	var s HostService

	draft, err := s.ChangeDraft(repo, "feature")
	if err != nil {
		t.Fatal(err)
	}
	if draft.Title != "feat(login): add the login form" || draft.Target != "main" || draft.HasUpstream || draft.Kind != "merge request" || draft.Remote != "origin" {
		t.Fatalf("draft = %+v", draft)
	}
	want := "- feat(login): add the login form\n- fix(login): handle an empty password\n\n## Checklist\n\n- [ ] Tests"
	if draft.Description != want {
		t.Fatalf("description =\n%s\nwant\n%s", draft.Description, want)
	}
}

func TestCreateChangeGoesToTheHostAndIntoTheCache(t *testing.T) {
	repo := enabledGitLab(t)
	gitIn(t, repo, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "first")
	gitIn(t, repo, "switch", "-q", "-c", "feature")
	api := "/api/v4/projects/group%2Fapp"
	transport := &routeTransport{routes: map[string]string{
		"POST " + api + "/merge_requests": `{"iid": 9, "title": "Draft: Add the login form", "state": "opened", "draft": true, "source_branch": "feature", "target_branch": "main", "sha": "abc", "web_url": "https://gitlab.example.com/group/app/-/merge_requests/9", "author": {"name": "Ada", "username": "ada"}}`,
	}, hits: map[string]int{}}
	transport.routes[api+"/merge_requests"] = transport.routes["POST "+api+"/merge_requests"]
	swapHostClient(t, transport)
	var s HostService

	change, err := s.CreateChange(repo, "feature", NewChangeInput{Target: "main", Title: "Add the login form", Description: "- feat", Draft: true})
	if err != nil || change.Number != 9 || !change.Draft {
		t.Fatalf("CreateChange = %+v, %v", change, err)
	}
	changes, _ := s.BranchChanges(repo)
	if got := changes["origin/feature"]; got.Number != 9 || got.State != "open" {
		t.Fatalf("the created change should be in the cache: %+v", changes)
	}

	transport.routes[api+"/merge_requests"] = "403"
	_, err = s.CreateChange(repo, "feature", NewChangeInput{Target: "main", Title: "x"})
	if err == nil || !strings.Contains(err.Error(), "api scope") {
		t.Fatalf("a token without write scope should name the scope: %v", err)
	}
}
