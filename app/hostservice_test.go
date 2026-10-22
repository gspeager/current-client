package app

import (
	"testing"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/internal/config"
)

func TestSetHostRoundTripsThroughSettings(t *testing.T) {
	useTempSettings(t)
	remote := git.Remote{Name: "origin", FetchURL: "https://git.example.com:8443/team/repo.git"}
	var s HostService

	info, ok := hostInfoFor(remote, configOrDefault().Hosts)
	if !ok || info.Provider != "github" || info.APIURL != "https://git.example.com:8443/api/v3" {
		t.Fatalf("before any choice: %+v, %v", info, ok)
	}

	if err := s.SetHost("git.example.com:8443", "gitlab", "https://git.example.com:8443/gitlab/api/v4/"); err != nil {
		t.Fatalf("SetHost: %v", err)
	}
	hosts := configOrDefault().Hosts
	info, _ = hostInfoFor(remote, hosts)
	want := HostInfo{
		Remote:           "origin",
		Hostname:         "git.example.com:8443",
		Project:          "team/repo",
		Provider:         "gitlab",
		DetectedProvider: "github",
		APIURL:           "https://git.example.com:8443/gitlab/api/v4",
		DefaultAPIURL:    "https://git.example.com:8443/api/v4",
	}
	if info != want {
		t.Fatalf("after choosing GitLab:\n got %+v\nwant %+v", info, want)
	}
	if links := hostLinksFor([]git.Remote{remote}, hosts); links == nil || links.Issue != "https://git.example.com:8443/team/repo/-/issues/" {
		t.Fatalf("links don't follow the choice: %+v", links)
	}

	if err := s.SetHost("git.example.com:8443", "", ""); err != nil {
		t.Fatalf("SetHost(clear): %v", err)
	}
	if hosts := configOrDefault().Hosts; len(hosts) != 0 {
		t.Fatalf("clearing left %+v", hosts)
	}
}

func TestSetHostRejectsUnknownProviderAndBadAddress(t *testing.T) {
	useTempSettings(t)
	var s HostService
	if err := s.SetHost("git.example.com", "bitbucket", ""); err == nil {
		t.Error("unknown provider accepted")
	}
	if err := s.SetHost("git.example.com", "gitlab", "git.example.com/api/v4"); err == nil {
		t.Error("address without a scheme accepted")
	}
	if hosts := configOrDefault().Hosts; len(hosts) != 0 {
		t.Fatalf("a rejected choice was saved: %+v", hosts)
	}
}

func TestHostInfoForSkipsLocalRemotes(t *testing.T) {
	if _, ok := hostInfoFor(git.Remote{Name: "backup", FetchURL: "/srv/git/repo.git"}, map[string]config.HostConfig{}); ok {
		t.Error("a local path was reported as a host")
	}
}
