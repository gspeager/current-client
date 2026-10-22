package app

import (
	"testing"

	"github.com/gspeager/current-client/core/git"
)

func TestSplitRemoteBranch(t *testing.T) {
	remotes := []git.Remote{{Name: "origin"}, {Name: "team"}, {Name: "team/origin"}}
	tests := []struct {
		ref, remote, branch string
		ok                  bool
	}{
		{"origin/feature", "origin", "feature", true},
		{"origin/user/feature", "origin", "user/feature", true},
		{"team/origin/feature", "team/origin", "feature", true},
		{"team/feature", "team", "feature", true},
		{"upstream/feature", "", "", false},
		{"origin/", "", "", false},
	}
	for _, tt := range tests {
		remote, branch, ok := splitRemoteBranch(remotes, tt.ref)
		if remote != tt.remote || branch != tt.branch || ok != tt.ok {
			t.Errorf("splitRemoteBranch(%q) = %q, %q, %v; want %q, %q, %v", tt.ref, remote, branch, ok, tt.remote, tt.branch, tt.ok)
		}
	}
}

func TestHostLinksPrefersOrigin(t *testing.T) {
	remotes := []git.Remote{
		{Name: "fork", FetchURL: "git@github.com:me/repo.git"},
		{Name: "origin", FetchURL: "", PushURL: "https://gitlab.com/group/repo.git"},
	}
	links := hostLinksFor(remotes)
	if links == nil || links.Issue != "https://gitlab.com/group/repo/-/issues/" {
		t.Fatalf("links = %+v", links)
	}
	if links := hostLinksFor(remotes[:1]); links == nil || links.Issue != "https://github.com/me/repo/issues/" {
		t.Fatalf("first remote links = %+v", links)
	}
	if links := hostLinksFor([]git.Remote{{Name: "origin", FetchURL: "/srv/git/repo.git"}}); links != nil {
		t.Fatalf("local remote links = %+v", links)
	}
	if hostLinksFor(nil) != nil {
		t.Fatal("no remotes should give no links")
	}
}
