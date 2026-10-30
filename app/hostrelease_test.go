//go:build !nohost

package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReleaseDraftFromTheChangelogAtTheTag(t *testing.T) {
	repo := enabledGitLab(t)
	commit := func(message string) {
		gitIn(t, repo, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", message)
	}
	commit("feat: first")
	gitIn(t, repo, "tag", "v0.1.0")
	commit("feat: forms")
	commit("fix: login")
	gitIn(t, repo, "tag", "v0.2.0")
	os.WriteFile(filepath.Join(repo, "CHANGELOG.md"), []byte("# Changelog\n\n## [0.3.0] - 2026-11-01\n\n### Added\n\n- Issues.\n\n## [0.2.0]\n\n- Old.\n"), 0o644)
	gitIn(t, repo, "add", "CHANGELOG.md")
	commit("docs: changelog")
	gitIn(t, repo, "tag", "-a", "v0.3.0", "-m", "0.3.0")
	var s HostService

	draft, err := s.ReleaseDraft(repo, "v0.3.0")
	if err != nil || draft.NotesFrom != "changelog" || draft.Notes != "### Added\n\n- Issues." || draft.Name != "v0.3.0" || draft.Remote != "origin" {
		t.Fatalf("ReleaseDraft(v0.3.0) = %+v, %v", draft, err)
	}
	// The changelog written after v0.2.0 doesn't count for it: its notes come
	// from its own commits.
	draft, err = s.ReleaseDraft(repo, "v0.2.0")
	if err != nil || draft.NotesFrom != "commits" || draft.Since != "v0.1.0" || draft.Notes != "### Added\n\n- forms\n\n### Fixed\n\n- login" {
		t.Fatalf("ReleaseDraft(v0.2.0) = %+v, %v", draft, err)
	}
	if _, err := s.ReleaseDraft(repo, "v9"); err == nil {
		t.Fatal("a draft for a tag that doesn't exist")
	}
}

func TestPublishReleaseSendsOneRequest(t *testing.T) {
	repo := enabledGitLab(t)
	api := "/api/v4/projects/group%2Fapp"
	transport := &routeTransport{routes: map[string]string{
		api + "/releases": `{"tag_name": "v0.3.0", "name": "v0.3.0", "_links": {"self": "https://gitlab.example.com/group/app/-/releases/v0.3.0"}}`,
	}, hits: map[string]int{}}
	swapHostClient(t, transport)
	var s HostService
	if !s.CanPublishReleases(repo) {
		t.Fatal("GitLab should publish releases")
	}
	published, err := s.PublishRelease(repo, NewReleaseInput{Tag: "v0.3.0", Name: " ", Notes: "- Issues."})
	if err != nil || published.Name != "v0.3.0" || transport.hits[api+"/releases"] != 1 {
		t.Fatalf("PublishRelease = %+v, %v, hits %v", published, err, transport.hits)
	}
}
