//go:build !nohost

package app

import (
	"errors"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
)

func mergeFixture(t *testing.T) (repo, head string) {
	t.Helper()
	repo = enabledGitLab(t)
	gitIn(t, repo, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "base")
	gitIn(t, repo, "switch", "-q", "-c", "feature")
	gitIn(t, repo, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "feature")
	head = strings.TrimSpace(gitOut(t, repo, "rev-parse", "HEAD"))
	gitIn(t, repo, "update-ref", "refs/remotes/origin/feature", head)
	gitIn(t, repo, "branch", "-q", "--set-upstream-to=origin/feature", "feature")
	gitIn(t, repo, "switch", "-q", "main")
	gitIn(t, repo, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "main moved on")
	gitIn(t, repo, "update-ref", "refs/remotes/origin/main", strings.TrimSpace(gitOut(t, repo, "rev-parse", "HEAD")))
	return repo, head
}

func factsByLabel(facts []ReadinessFact) map[string]ReadinessFact {
	m := map[string]ReadinessFact{}
	for _, f := range facts {
		m[f.Label] = f
	}
	return m
}

func TestMergeReadinessComesFromTheCacheAndGit(t *testing.T) {
	repo, head := mergeFixture(t)
	_, info, _, err := primaryHost(repo)
	if err != nil {
		t.Fatal(err)
	}
	bob := hostapi.User{Name: "Bob", Username: "bob"}
	ann := hostapi.User{Name: "Ann", Username: "ann"}
	changes := []hostapi.Change{
		{Number: 7, Title: "Add the thing", SourceBranch: "feature", TargetBranch: "main", HeadSHA: head, State: hostapi.ChangeOpen, CI: hostapi.CISuccess, ConflictsKnown: true, Reviews: []hostapi.Review{{Reviewer: bob, State: hostapi.ReviewApproved}, {Reviewer: ann, State: hostapi.ReviewRequested}}},
		{Number: 8, Title: "Unknown head", SourceBranch: "other", TargetBranch: "main", HeadSHA: "0123456789abcdef0123456789abcdef01234567", State: hostapi.ChangeOpen, Draft: true, HasConflicts: true, ConflictsKnown: true, UnresolvedThreads: 2, Reviews: []hostapi.Review{{Reviewer: ann, State: hostapi.ReviewCommented}, {Reviewer: bob, State: hostapi.ReviewChangesRequested}}},
		{Number: 9, Title: "From GitHub's list", SourceBranch: "gh", TargetBranch: "main", HeadSHA: head, State: hostapi.ChangeOpen},
	}
	hostClient.Cache.Put(info.Hostname, repo, cacheChanges, info.Project, cachedChanges{changes, false})
	hostClient.Cache.Put(info.Hostname, repo, cacheRules, "all", []hostapi.BranchRule{{Pattern: "main", NoPush: true}})
	var s HostService

	ready, err := s.MergeReadiness(repo, 7)
	if err != nil || ready.FetchedAt == nil {
		t.Fatalf("MergeReadiness = %+v, %v", ready, err)
	}
	facts := factsByLabel(ready.Facts)
	if f := facts["Checks"]; f.Level != "ok" || f.Value != "passed" {
		t.Errorf("checks = %+v", f)
	}
	if f := facts["Reviews"]; f.Level != "warn" || f.Value != "approved by Bob; waiting on Ann" {
		t.Errorf("reviews = %+v (an approval and a pending request: the pending one wins)", f)
	}
	if f := facts["Conflicts"]; f.Level != "ok" {
		t.Errorf("conflicts = %+v", f)
	}
	if f := facts["Behind main"]; f.Level != "warn" || !strings.HasPrefix(f.Value, "1 commit") {
		t.Errorf("behind = %+v", f)
	}
	if f := facts["Protection"]; f.Level != "info" || !strings.HasPrefix(f.Value, "protected; the host applies its rules on merge") {
		t.Errorf("protection = %+v", f)
	}
	if _, ok := facts["Draft"]; ok {
		t.Error("not a draft")
	}

	ready, _ = s.MergeReadiness(repo, 8)
	facts = factsByLabel(ready.Facts)
	if facts["Draft"].Level != "warn" || facts["Checks"].Level != "unknown" || facts["Conflicts"].Level != "warn" || facts["Threads"].Value != "2 unresolved" || facts["Behind main"].Level != "unknown" {
		t.Errorf("facts = %+v", ready.Facts)
	}
	if f := facts["Reviews"]; f.Level != "warn" || f.Value != "changes requested by Bob; commented by Ann" {
		t.Errorf("reviews = %+v", f)
	}
	ready, _ = s.MergeReadiness(repo, 9)
	if f := factsByLabel(ready.Facts)["Conflicts"]; f.Level != "unknown" {
		t.Errorf("conflicts unknown for a list entry without an answer: %+v", f)
	}
}

func TestMergeChangeMergesOnTheHostAndMovesItToMerged(t *testing.T) {
	repo, head := mergeFixture(t)
	_, info, _, err := primaryHost(repo)
	if err != nil {
		t.Fatal(err)
	}
	api := "/api/v4/projects/group%2Fapp"
	transport := &routeTransport{routes: map[string]string{
		api:                             `{"merge_method": "merge", "squash_option": "default_on", "remove_source_branch_after_merge": true}`,
		api + "/merge_requests/7/merge": `{"iid": 7, "state": "merged"}`,
		api + "/merge_requests/8/merge": "405",
	}, hits: map[string]int{}}
	swapHostClient(t, transport)
	changes := []hostapi.Change{
		{Number: 7, SourceBranch: "feature", TargetBranch: "main", HeadSHA: head, State: hostapi.ChangeOpen},
		{Number: 8, SourceBranch: "other", TargetBranch: "main", HeadSHA: head, State: hostapi.ChangeOpen},
	}
	hostClient.Cache.Put(info.Hostname, repo, cacheChanges, info.Project, cachedChanges{changes, false})
	var s HostService

	options, err := s.MergeOptions(repo, 7)
	if err != nil || options.Methods[0] != "squash" || !options.DeleteBranch {
		t.Fatalf("MergeOptions = %+v, %v", options, err)
	}
	result, err := s.MergeChange(repo, 7, "squash", true)
	if err != nil || result.LocalBranch != "feature" || result.SourceBranch != "feature" {
		t.Fatalf("MergeChange = %+v, %v", result, err)
	}
	if transport.hits[api+"/merge_requests/7/merge"] != 1 {
		t.Fatal("merge should be one request")
	}
	list, _ := s.Changes(repo)
	if len(list.Changes) != 1 || list.Changes[0].Number != 8 {
		t.Errorf("the merged change leaves the open list: %+v", list.Changes)
	}
	if merged := mergedOnHost(repo); merged["origin/feature"].Number != 7 {
		t.Errorf("the clean-up should know the branch merged: %+v", merged)
	}

	_, err = s.MergeChange(repo, 8, "merge", false)
	var refused *hostapi.MergeRefusedError
	if !errors.As(err, &refused) {
		t.Fatalf("a refusal reaches the caller as one: %v", err)
	}
	if list, _ := s.Changes(repo); len(list.Changes) != 1 {
		t.Errorf("a refused merge changes nothing: %+v", list.Changes)
	}
}
