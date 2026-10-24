//go:build !nohost

package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostapi/gitlab"
	"github.com/gspeager/current-client/core/hostapi/hostapitest"
	"github.com/gspeager/current-client/internal/config"
)

func TestReviewThreadsOfTheCheckedOutBranchAreFetchedOnRefreshAndReadFromTheCache(t *testing.T) {
	useTempSettings(t)
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "feature")
	gitIn(t, repo, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "first")
	gitIn(t, repo, "remote", "add", "origin", "git@gitlab.example.com:group/app.git")
	gitIn(t, repo, "update-ref", "refs/remotes/origin/feature", "HEAD")
	gitIn(t, repo, "branch", "--set-upstream-to=origin/feature", "feature")
	if err := updateConfig(func(cfg *config.Config) {
		cfg.Hosts = map[string]config.HostConfig{"gitlab.example.com": {Provider: "gitlab", Enabled: true}}
	}); err != nil {
		t.Fatal(err)
	}
	api := "/api/v4/projects/group%2Fapp"
	transport := &routeTransport{routes: map[string]string{
		api + "/merge_requests/7/discussions?per_page=100": `[
			{"id": "d1", "notes": [{"body": "Rename this to **client**.", "author": {"name": "Ada", "username": "ada"}, "created_at": "2026-10-20T10:00:00Z", "resolvable": true, "resolved": false,
			  "position": {"head_sha": "abc", "new_path": "app/x.go", "old_path": "app/x.go", "new_line": 42}}]},
			{"id": "d2", "notes": [{"body": "Typo.", "author": {"name": "Ada", "username": "ada"}, "created_at": "2026-10-20T10:00:00Z", "resolvable": true, "resolved": true,
			  "position": {"head_sha": "abc", "new_path": "README.md", "old_path": "README.md", "new_line": 7}}]},
			{"id": "d3", "notes": [{"body": "Moved since <script>x()</script>", "author": {"name": "Bob", "username": "bob"}, "created_at": "2026-10-20T10:00:00Z", "resolvable": true, "resolved": false,
			  "position": {"head_sha": "old", "new_path": "app/x.go", "old_path": "app/x.go", "old_line": 15}}]},
			{"id": "d4", "notes": [{"body": "Looks good overall.", "author": {"name": "Bob", "username": "bob"}, "created_at": "2026-10-20T10:00:00Z", "resolvable": false, "resolved": false}]}
		]`,
	}, hits: map[string]int{}}
	swapHostClient(t, transport)
	var s HostService

	hosts := configOrDefault().Hosts
	info := infoFor(hostByName("gitlab.example.com"), hosts)
	info.Project = "group/app"
	changes := []hostapi.Change{{Number: 7, SourceBranch: "feature", TargetBranch: "main", HeadSHA: "abc", State: hostapi.ChangeOpen}}
	provider := gitlab.New(hostClient, hostapi.Host{Name: info.Hostname, API: info.APIURL})
	fillCurrentBranchThreads(context.Background(), provider, info, repo, "origin", changes)
	hostClient.Cache.Put(info.Hostname, repo, cacheChanges, "group/app", cachedChanges{changes: changes})
	if changes[0].UnresolvedThreads != 3 {
		t.Fatalf("UnresolvedThreads = %d, want 3", changes[0].UnresolvedThreads)
	}

	fetched, _, ok := hostClient.Cache.Get(info.Hostname, repo, cacheThreads, "7")
	if !ok {
		t.Fatal("threads weren't cached")
	}
	// The read half runs against a transport that refuses every request.
	swapHostClient(t, hostapitest.Forbid(t))
	hostClient.Cache.Put(info.Hostname, repo, cacheChanges, "group/app", cachedChanges{changes: changes})
	hostClient.Cache.Put(info.Hostname, repo, cacheThreads, "7", fetched)
	threads, err := s.ReviewThreads(repo)
	if err != nil || !threads.Enabled || threads.Number != 7 || len(threads.Threads) != 4 || threads.Total != 3 {
		t.Fatalf("ReviewThreads = %+v, %v", threads, err)
	}
	if threads.Unresolved["app/x.go"] != 1 || threads.Unresolved["README.md"] != 0 {
		t.Fatalf("Unresolved per path = %v, want app/x.go once (the outdated one doesn't count)", threads.Unresolved)
	}
	byID := map[string]ReviewThread{}
	for _, th := range threads.Threads {
		byID[th.ID] = th
	}
	if th := byID["d1"]; th.Line != 42 || th.Resolved || th.Outdated || th.Notes[0].BodyHTML != "<p>Rename this to <strong>client</strong>.</p>\n" {
		t.Errorf("d1 = %+v", th)
	}
	if th := byID["d2"]; !th.Resolved {
		t.Errorf("d2 = %+v", th)
	}
	if th := byID["d3"]; !th.Outdated || th.OldLine != 15 || strings.Contains(th.Notes[0].BodyHTML, "<script") || !strings.Contains(th.Notes[0].BodyHTML, "Moved since") {
		t.Errorf("d3 = %+v", th)
	}
	if th := byID["d4"]; th.Path != "" || th.Line != 0 {
		t.Errorf("d4 = %+v", th)
	}
}

func TestReviewThreadsAtKeepOnlyFilesThatMatchTheChangeHead(t *testing.T) {
	useTempSettings(t)
	swapHostClient(t, hostapitest.Forbid(t))
	repo := t.TempDir()
	commit := func(name, content string) string {
		if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		gitIn(t, repo, "add", name)
		gitIn(t, repo, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "-m", name)
		out, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(out))
	}
	gitIn(t, repo, "init", "-q", "-b", "feature")
	commit("a.go", "a\n")
	first := commit("b.go", "b\n")
	head := commit("b.go", "b2\n")
	gitIn(t, repo, "remote", "add", "origin", "git@gitlab.example.com:group/app.git")
	gitIn(t, repo, "update-ref", "refs/remotes/origin/feature", "HEAD")
	gitIn(t, repo, "branch", "--set-upstream-to=origin/feature", "feature")
	if err := updateConfig(func(cfg *config.Config) {
		cfg.Hosts = map[string]config.HostConfig{"gitlab.example.com": {Provider: "gitlab", Enabled: true}}
	}); err != nil {
		t.Fatal(err)
	}
	info := infoFor(hostByName("gitlab.example.com"), configOrDefault().Hosts)
	put := func(headSHA string) {
		changes := []hostapi.Change{{Number: 7, SourceBranch: "feature", TargetBranch: "main", HeadSHA: headSHA, State: hostapi.ChangeOpen}}
		hostClient.Cache.Put(info.Hostname, repo, cacheChanges, "group/app", cachedChanges{changes: changes})
		hostClient.Cache.Put(info.Hostname, repo, cacheThreads, "7", []hostapi.Thread{
			{ID: "a", Path: "a.go", Line: 1},
			{ID: "b", Path: "b.go", Line: 1},
			{ID: "all"},
		})
	}
	ids := func(threads ReviewThreads) string {
		var got []string
		for _, th := range threads.Threads {
			got = append(got, th.ID)
		}
		return strings.Join(got, ",")
	}
	var s HostService

	put(head)
	for sha, want := range map[string]string{head: "a,b,all", first: "a,all"} {
		threads, err := s.ReviewThreadsAt(repo, sha)
		if err != nil || ids(threads) != want {
			t.Errorf("ReviewThreadsAt(%s) = %q, %v, want %q", sha[:7], ids(threads), err, want)
		}
	}
	put("0123456789abcdef0123456789abcdef01234567")
	if threads, err := s.ReviewThreadsAt(repo, head); err != nil || ids(threads) != "all" {
		t.Errorf("with the head not fetched = %q, %v, want only the thread on the whole change", ids(threads), err)
	}
}
