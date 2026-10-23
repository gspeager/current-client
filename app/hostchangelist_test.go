//go:build !nohost

package app

import (
	"context"
	"net/http/cgi"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostapi/hostapitest"
	"github.com/gspeager/current-client/internal/config"
)

// hostedRemote serves a bare repository over smart HTTP, so its URL parses
// as a host the way a real one does, and returns the clone plus the host
// name to enable.
func hostedRemote(t *testing.T) (repo, hostname, remoteDir string) {
	t.Helper()
	execPath, err := exec.Command("git", "--exec-path").Output()
	if err != nil {
		t.Skipf("git --exec-path: %v", err)
	}
	backend := filepath.Join(strings.TrimSpace(string(execPath)), "git-http-backend")
	if _, err := os.Stat(backend); err != nil {
		t.Skip("git-http-backend not available")
	}
	root := t.TempDir()
	remoteDir = filepath.Join(root, "group", "app.git")
	gitIn(t, root, "init", "-q", "--bare", "-b", "main", remoteDir)
	gitIn(t, remoteDir, "config", "http.receivepack", "true")
	server := httptest.NewServer(&cgi.Handler{Path: backend, Env: []string{"GIT_PROJECT_ROOT=" + root, "GIT_HTTP_EXPORT_ALL=1", "GIT_HTTP_RECEIVE_PACK=1"}})
	t.Cleanup(server.Close)
	hostname = strings.TrimPrefix(server.URL, "http://")

	repo = t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	gitIn(t, repo, "config", "user.name", "T")
	gitIn(t, repo, "config", "user.email", "t@example.com")
	gitIn(t, repo, "commit", "-q", "--allow-empty", "-m", "first")
	gitIn(t, repo, "remote", "add", "origin", server.URL+"/group/app.git")
	gitIn(t, repo, "push", "-q", "origin", "main")
	return repo, hostname, remoteDir
}

func TestChangesPanelIsAbsentUntilTheHostIsOn(t *testing.T) {
	useTempSettings(t)
	swapHostClient(t, hostapitest.Forbid(t))
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q")
	gitIn(t, repo, "remote", "add", "origin", "git@gitlab.example.com:group/app.git")
	var s HostService

	list, err := s.Changes(repo)
	if err != nil || list.Enabled || len(list.Changes) != 0 {
		t.Fatalf("with the host off: %+v, %v", list, err)
	}
	if err := s.SetEnabled("gitlab.example.com", true); err != nil {
		t.Fatal(err)
	}
	list, err = s.Changes(repo)
	if err != nil || !list.Enabled || list.FetchedAt != nil || list.Prefix != "!" || list.Project != "group/app" {
		t.Fatalf("enabled but not refreshed: %+v, %v", list, err)
	}

	hostClient.Cache.Put("gitlab.example.com", repo, cacheChanges, "group/app", cachedChanges{changes: []hostapi.Change{
		{Number: 7, Title: "Add the thing", Description: "See [the issue](/group/app/-/issues/4) <script>x()</script>", SourceBranch: "feature", Author: hostapi.User{Username: "ada"}},
	}, truncated: true})
	hostClient.Cache.Put("gitlab.example.com", repo, cacheUser, "origin", hostapi.User{Username: "ada"})
	list, _ = s.Changes(repo)
	if list.FetchedAt == nil || !list.Truncated || list.Me != "ada" || len(list.Changes) != 1 {
		t.Fatalf("after a refresh: %+v", list)
	}
	details, err := s.ChangeDetails(repo, 7)
	if err != nil || !strings.Contains(details.DescriptionHTML, `href="https://gitlab.example.com/group/app/-/issues/4"`) || strings.Contains(details.DescriptionHTML, "<script") {
		t.Fatalf("ChangeDetails = %+v, %v", details, err)
	}
	if _, err := s.ChangeDetails(repo, 99); err == nil {
		t.Fatal("a change outside the last refresh should be refused")
	}
}

func TestCheckoutChangeFetchesTheHeadIntoItsOwnNamespace(t *testing.T) {
	useTempSettings(t)
	swapHostClient(t, hostapitest.Forbid(t))
	repo, hostname, _ := hostedRemote(t)
	gitIn(t, repo, "switch", "-q", "-c", "feature")
	gitIn(t, repo, "commit", "-q", "--allow-empty", "-m", "on the merge request")
	headSHA := strings.TrimSpace(string(mustOutput(t, repo, "rev-parse", "HEAD")))
	gitIn(t, repo, "push", "-q", "origin", "feature:refs/merge-requests/3/head")
	gitIn(t, repo, "switch", "-q", "main")
	gitIn(t, repo, "branch", "-q", "-D", "feature")

	if err := updateConfig(func(cfg *config.Config) {
		cfg.Hosts = map[string]config.HostConfig{hostname: {Provider: "gitlab", Enabled: true}}
	}); err != nil {
		t.Fatal(err)
	}
	hostClient.Cache.Put(hostname, repo, cacheChanges, "group/app", cachedChanges{changes: []hostapi.Change{
		{Number: 3, SourceBranch: "feature", HeadSHA: headSHA},
		{Number: 4, SourceBranch: "fix", SourceProject: "someone/app", HeadSHA: headSHA},
	}})
	var s HostService

	result, err := s.CheckoutChange(context.Background(), repo, 3, nil)
	if err != nil || result.Branch != "feature" || result.Existed {
		t.Fatalf("CheckoutChange = %+v, %v", result, err)
	}
	if got := strings.TrimSpace(string(mustOutput(t, repo, "rev-parse", "--abbrev-ref", "HEAD"))); got != "feature" {
		t.Fatalf("HEAD is %s", got)
	}
	if got := strings.TrimSpace(string(mustOutput(t, repo, "rev-parse", "refs/current-client/origin/mr/3"))); got != headSHA {
		t.Fatalf("fetched head = %s, want %s", got, headSHA)
	}
	again, err := s.CheckoutChange(context.Background(), repo, 3, nil)
	if err != nil || !again.Existed {
		t.Fatalf("second checkout = %+v, %v", again, err)
	}

	gitIn(t, repo, "push", "-q", "origin", "feature:refs/merge-requests/4/head")
	fork, err := s.CheckoutChange(context.Background(), repo, 4, nil)
	if err != nil || fork.Branch != "someone/fix" {
		t.Fatalf("fork checkout = %+v, %v", fork, err)
	}

	if err := s.SetEnabled(hostname, false); err != nil {
		t.Fatal(err)
	}
	if err := updateConfig(func(cfg *config.Config) { cfg.OpenTabs = []string{repo} }); err != nil {
		t.Fatal(err)
	}
	if err := updateConfig(func(cfg *config.Config) {
		cfg.Hosts = map[string]config.HostConfig{hostname: {Provider: "gitlab", Enabled: true}}
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetEnabled(hostname, false); err != nil {
		t.Fatal(err)
	}
	if out := strings.TrimSpace(string(mustOutput(t, repo, "for-each-ref", "refs/current-client"))); out != "" {
		t.Fatalf("turning the host off left fetched heads:\n%s", out)
	}
}

func mustOutput(t *testing.T, dir string, args ...string) []byte {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return out
}

func TestClosedChangesFilesCommitsAndThreadsAreAskedOnceUntilARefresh(t *testing.T) {
	useTempSettings(t)
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	gitIn(t, repo, "remote", "add", "origin", "git@github.com:owner/app.git")
	if err := updateConfig(func(cfg *config.Config) {
		cfg.Hosts = map[string]config.HostConfig{"github.com": {Enabled: true}}
	}); err != nil {
		t.Fatal(err)
	}
	api := "/repos/owner/app"
	closed := api + "/pulls?direction=desc&per_page=50&sort=updated&state=closed"
	transport := &routeTransport{routes: map[string]string{
		api + "/pulls?per_page=100&state=open": `[]`,
		closed:                                 `[{"number": 3, "title": "Done", "state": "closed", "merged_at": "2026-10-01T00:00:00Z", "head": {"ref": "done", "sha": "abc"}, "base": {"ref": "main"}, "user": {"login": "ada"}}]`,
		api + "/pulls/3/files?per_page=100":    `[{"filename": "a.go", "status": "modified", "additions": 1, "deletions": 2}]`,
		api + "/pulls/3/commits?per_page=100":  `[{"sha": "abc", "commit": {"message": "done", "author": {"name": "Ada", "date": "2026-10-02T03:04:05Z"}}}]`,
		api + "/pulls/3/comments?per_page=100": `[{"id": 1, "path": "a.go", "line": 2, "side": "RIGHT", "body": "Why?", "user": {"login": "bob"}, "commit_id": "abc", "position": 1}]`,
		"/user":                                `{"login": "ada"}`,
	}, hits: map[string]int{}}
	swapHostClient(t, transport)
	var s HostService
	if err := s.Refresh(repo); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ChangeFiles(repo, 3); err == nil {
		t.Fatal("a change the cache doesn't know should be refused")
	}
	for range 2 {
		list, err := s.ClosedChanges(repo)
		if err != nil || len(list.Changes) != 1 || list.Changes[0].Number != 3 || list.FetchedAt == nil || list.Me != "ada" {
			t.Fatalf("ClosedChanges = %+v, %v", list, err)
		}
		files, err := s.ChangeFiles(repo, 3)
		if err != nil || len(files) != 1 || files[0].Deletions != 2 {
			t.Fatalf("ChangeFiles = %+v, %v", files, err)
		}
		commits, err := s.ChangeCommits(repo, 3)
		if err != nil || len(commits) != 1 || commits[0].Author != "Ada" {
			t.Fatalf("ChangeCommits = %+v, %v", commits, err)
		}
		threads, err := s.ChangeThreads(repo, 3)
		if err != nil || threads.Number != 3 || threads.Total != 1 || threads.Unresolved["a.go"] != 1 || !strings.Contains(threads.Threads[0].Notes[0].BodyHTML, "Why?") {
			t.Fatalf("ChangeThreads = %+v, %v", threads, err)
		}
	}
	for _, key := range []string{closed, api + "/pulls/3/files?per_page=100", api + "/pulls/3/commits?per_page=100", api + "/pulls/3/comments?per_page=100"} {
		if transport.hits[key] != 1 {
			t.Fatalf("%s asked %d times, want once; hits = %v", key, transport.hits[key], transport.hits)
		}
	}
	if err := s.Refresh(repo); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ClosedChanges(repo); err != nil || transport.hits[closed] != 2 {
		t.Fatalf("after a refresh the closed list should be asked again: %v, hits %v", err, transport.hits)
	}
}
