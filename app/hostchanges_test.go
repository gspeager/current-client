//go:build !nohost

package app

import (
	"context"
	"testing"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostapi/hostapitest"
)

func TestBranchChangesReadOnlyTheCache(t *testing.T) {
	useTempSettings(t)
	swapHostClient(t, hostapitest.Forbid(t))
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q")
	gitIn(t, repo, "remote", "add", "origin", "git@gitlab.example.com:group/app.git")
	var s HostService

	hostClient.Cache.Put("gitlab.example.com", repo, cacheRemoteRefs, "origin", git.RemoteChanges{
		Kind: git.MergeRequestRefs, ByBranch: map[string]int{"feature": 7, "older": 9},
	})
	hostClient.Cache.Put("gitlab.example.com", repo, cacheChanges, "group/app", cachedChanges{changes: []hostapi.Change{
		{Number: 7, Title: "Add the thing", State: hostapi.ChangeOpen, Draft: true, SourceBranch: "feature", CI: hostapi.CIFailed, Author: hostapi.User{Name: "Ada"}, WebURL: "https://gitlab.example.com/group/app/-/merge_requests/7"},
		{Number: 8, Title: "From a fork", State: hostapi.ChangeOpen, SourceBranch: "feature", SourceProject: "someone/app"},
	}})

	changes, err := s.BranchChanges(repo)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]BranchChange{
		"origin/feature": {Number: 7, Prefix: "!", State: "open", Draft: true, CI: "failed", Title: "Add the thing", Author: "Ada", WebURL: "https://gitlab.example.com/group/app/-/merge_requests/7", SourceBranch: "feature", UpdatedAt: "0001-01-01T00:00:00Z"},
		"origin/older":   {Number: 9, Prefix: "!"},
	}
	if len(changes) != len(want) {
		t.Fatalf("BranchChanges = %+v", changes)
	}
	for key, w := range want {
		if changes[key] != w {
			t.Errorf("%s = %+v, want %+v", key, changes[key], w)
		}
	}
}

func TestFetchRefreshesChangeNumbersFromGit(t *testing.T) {
	useTempSettings(t)
	swapHostClient(t, hostapitest.Forbid(t))
	remoteDir := t.TempDir() + "/remote.git"
	gitIn(t, t.TempDir(), "init", "-q", "--bare", "-b", "main", remoteDir)
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	gitIn(t, repo, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "first")
	gitIn(t, repo, "remote", "add", "origin", remoteDir)
	gitIn(t, repo, "push", "-q", "origin", "main")
	gitIn(t, repo, "push", "-q", "origin", "main:refs/heads/feature", "main:refs/pull/3/head", "main:refs/pull/3/merge")

	var remotes RemoteService
	if err := remotes.FetchAll(context.Background(), repo, nil); err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	var s HostService
	changes, err := s.BranchChanges(repo)
	if err != nil {
		t.Fatal(err)
	}
	if got := changes["origin/feature"]; got != (BranchChange{Number: 3, Prefix: "#"}) || changes["origin/main"] != got {
		t.Fatalf("after a fetch, BranchChanges = %+v", changes)
	}
}
