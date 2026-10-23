//go:build !nohost

package app

import (
	"context"
	"strings"
	"testing"
)

func TestCreateLinkInFindsTheHostsLink(t *testing.T) {
	gitlab := "remote:\nremote: To create a merge request for feature, visit:\nremote:   https://gitlab.com/g/r/-/merge_requests/new?merge_request%5Bsource_branch%5D=feature\nremote:\nTo gitlab.com:g/r.git\n * [new branch]      feature -> feature\n"
	github := "remote:\nremote: Create a pull request for 'feature' on GitHub by visiting:\nremote:      https://github.com/o/r/pull/new/feature\nremote:\n"
	if got := createLinkIn(gitlab); got != "https://gitlab.com/g/r/-/merge_requests/new?merge_request%5Bsource_branch%5D=feature" {
		t.Errorf("gitlab link = %q", got)
	}
	if got := createLinkIn(github); got != "https://github.com/o/r/pull/new/feature" {
		t.Errorf("github link = %q", got)
	}
	if got := createLinkIn("To gitlab.com:g/r.git\n   abc..def  feature -> feature\n"); got != "" {
		t.Errorf("a push without a link gave %q", got)
	}
}

func TestCreateChangeLinkUsesTheUpstreamProjectForAFork(t *testing.T) {
	useTempSettings(t)
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	gitIn(t, repo, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "first")
	gitIn(t, repo, "remote", "add", "origin", "git@github.com:someone/repo.git")
	gitIn(t, repo, "switch", "-q", "-c", "fix")
	gitIn(t, repo, "update-ref", "refs/remotes/origin/fix", "HEAD")
	gitIn(t, repo, "branch", "--set-upstream-to=origin/fix", "fix")
	var s RemoteService

	link, err := s.CreateChangeLink(repo, "fix")
	if err != nil || link != "https://github.com/someone/repo/compare/main...fix?expand=1" {
		t.Fatalf("plain link = %q, %v", link, err)
	}
	gitIn(t, repo, "remote", "add", "upstream", "https://github.com/owner/repo.git")
	link, _ = s.CreateChangeLink(repo, "fix")
	if link != "https://github.com/owner/repo/compare/main...someone:fix?expand=1" {
		t.Fatalf("fork link = %q", link)
	}

	local := t.TempDir()
	gitIn(t, local, "init", "-q")
	gitIn(t, local, "remote", "add", "origin", "/srv/git/repo.git")
	if link, err := s.CreateChangeLink(local, "fix"); err != nil || link != "" {
		t.Fatalf("a local remote gave %q, %v", link, err)
	}
}

func TestPushSetUpstreamReportsWhereToCreateTheChange(t *testing.T) {
	useTempSettings(t)
	repo, hostname, _ := hostedRemote(t)
	gitIn(t, repo, "switch", "-q", "-c", "feature")
	gitIn(t, repo, "commit", "-q", "--allow-empty", "-m", "work")
	var s RemoteService

	result, err := s.PushSetUpstream(context.Background(), repo, "origin", "feature", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Branch != "feature" || result.Upstream != "origin/feature" || result.Kind != "pull request" ||
		!strings.HasPrefix(result.CreateLink, "http://"+hostname+"/group/app/compare/main...feature") {
		t.Fatalf("PushSetUpstream = %+v", result)
	}
}
