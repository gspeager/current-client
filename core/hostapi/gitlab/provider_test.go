package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostapi/hostapitest"
)

const project = "gitlab-org/cli"

type request struct {
	Method, Path, Query, Body string
}

type fixtures struct {
	mu       sync.Mutex
	requests []request
	// routes map "METHOD /api/v4/<escaped path>?<query>" to a testdata file,
	// a literal body when it starts with "[" or "{", or a status such as
	// "403" for a refusal.
	routes map[string]string
}

func (f *fixtures) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	f.requests = append(f.requests, request{r.Method, r.URL.EscapedPath(), r.URL.RawQuery, string(body)})
	f.mu.Unlock()
	key := r.Method + " " + r.URL.EscapedPath()
	if r.URL.RawQuery != "" {
		key += "?" + r.URL.RawQuery
	}
	route, ok := f.routes[key]
	switch {
	case !ok:
		http.Error(w, `{"message":"404 Not Found"}`, http.StatusNotFound)
	case len(route) == 3 && route >= "400" && route <= "599":
		// A refusal: the status alone, as "403" or "405".
		status, _ := strconv.Atoi(route)
		http.Error(w, `{"message":"`+route+` `+http.StatusText(status)+`"}`, status)
	case strings.HasPrefix(route, "[") || strings.HasPrefix(route, "{"):
		w.Write([]byte(route))
	default:
		data, err := os.ReadFile(filepath.Join("testdata", route))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(data)
	}
}

func (f *fixtures) count(method, path string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, r := range f.requests {
		if r.Method == method && r.Path == path {
			n++
		}
	}
	return n
}

const api = "/api/v4/projects/gitlab-org%2Fcli"

func newProvider(t *testing.T, routes map[string]string) (*Provider, *fixtures) {
	t.Helper()
	hostapitest.IsolateGitConfig(t)
	f := &fixtures{routes: routes}
	server := httptest.NewServer(f)
	t.Cleanup(server.Close)
	host := hostapi.Host{Name: "gitlab.example.com", API: server.URL + "/api/v4"}
	client := hostapi.New(hostapi.Options{Enabled: hostapitest.AllowAll, PageCap: 3})
	return New(client, host), f
}

func TestOpenChangesFromRecordedFixture(t *testing.T) {
	p, f := newProvider(t, map[string]string{
		"GET " + api + "/merge_requests?per_page=100&state=opened": "merge_requests.json",
	})
	changes, truncated, err := p.OpenChanges(context.Background(), project)
	if err != nil || truncated || len(changes) != 2 {
		t.Fatalf("OpenChanges = %d changes, truncated=%v, err=%v", len(changes), truncated, err)
	}
	first := changes[0]
	if len(first.Reviews) != 1 || first.Reviews[0].Reviewer.Username != "GitLabDuo" || first.Reviews[0].State != hostapi.ReviewRequested {
		t.Fatalf("first change's reviews = %+v", first.Reviews)
	}
	if first.Number != 4179 || !first.Draft || first.State != hostapi.ChangeOpen || first.SourceProject != "" ||
		first.Author.Username == "" || first.Author.Name == "" || len(first.HeadSHA) != 40 || first.TargetBranch != "main" ||
		first.UpdatedAt.IsZero() || !strings.HasPrefix(first.WebURL, "https://gitlab.com/gitlab-org/cli/-/merge_requests/") {
		t.Fatalf("first change = %+v", first)
	}
	encoded, _ := json.Marshal(changes)
	if strings.Contains(string(encoded), "avatar") || strings.Contains(string(encoded), "uploads") {
		t.Fatalf("avatars leaked into the model: %s", encoded)
	}
	if n := f.count("GET", api+"/merge_requests"); n != 1 {
		t.Fatalf("merge requests fetched %d times", n)
	}
}

func TestReviewsReadReviewersThenApprovals(t *testing.T) {
	p, _ := newProvider(t, map[string]string{
		"GET " + api + "/merge_requests/4178/reviewers": "reviewers.json",
		"GET " + api + "/merge_requests/4178/approvals": "approvals.json",
	})
	reviews, err := p.Reviews(context.Background(), project, hostapi.Change{Number: 4178})
	if err != nil || len(reviews) != 2 || reviews[0].Reviewer.Username != "jhebden" || reviews[0].State != hostapi.ReviewApproved ||
		reviews[1].Reviewer.Username != "GitLabDuo" || reviews[1].State != hostapi.ReviewApproved {
		t.Fatalf("Reviews = %+v, %v", reviews, err)
	}

	p, _ = newProvider(t, map[string]string{
		"GET " + api + "/merge_requests/1/reviewers": `[
			{"user": {"username": "ada", "name": "Ada"}, "state": "unreviewed"},
			{"user": {"username": "bo", "name": "Bo"}, "state": "requested_changes"},
			{"user": {"username": "cy", "name": "Cy"}, "state": "reviewed"}]`,
		"GET " + api + "/merge_requests/1/approvals": `{"approved_by": [{"user": {"username": "ada", "name": "Ada"}}, {"user": {"username": "di", "name": "Di"}}]}`,
	})
	reviews, err = p.Reviews(context.Background(), project, hostapi.Change{Number: 1})
	want := []hostapi.Review{
		{Reviewer: hostapi.User{Name: "Ada", Username: "ada"}, State: hostapi.ReviewApproved},
		{Reviewer: hostapi.User{Name: "Bo", Username: "bo"}, State: hostapi.ReviewChangesRequested},
		{Reviewer: hostapi.User{Name: "Cy", Username: "cy"}, State: hostapi.ReviewCommented},
		{Reviewer: hostapi.User{Name: "Di", Username: "di"}, State: hostapi.ReviewApproved},
	}
	if err != nil || !reflect.DeepEqual(reviews, want) {
		t.Fatalf("Reviews = %+v, %v", reviews, err)
	}
}

func TestChangeReadsCIAndNamesAFork(t *testing.T) {
	p, f := newProvider(t, map[string]string{
		"GET " + api + "/merge_requests/4179": "merge_request.json",
		"GET " + api + "/merge_requests/4200": "merge_request_fork.json",
		"GET /api/v4/projects/777777":         "project_777777.json",
	})
	same, err := p.Change(context.Background(), project, 4179)
	if err != nil || same.CI != hostapi.CISuccess || same.SourceProject != "" || same.HasConflicts {
		t.Fatalf("Change(4179) = %+v, %v", same, err)
	}
	for range 2 {
		fork, err := p.Change(context.Background(), project, 4200)
		if err != nil || fork.SourceProject != "someone/cli" || fork.CI != hostapi.CIFailed || !fork.HasConflicts || fork.SourceBranch != "fix-typo" {
			t.Fatalf("Change(4200) = %+v, %v", fork, err)
		}
	}
	if n := f.count("GET", "/api/v4/projects/777777"); n != 1 {
		t.Fatalf("the fork's project was looked up %d times, want once", n)
	}
}

func TestThreadsKeepResolvedAndOutdatedApart(t *testing.T) {
	p, _ := newProvider(t, map[string]string{
		"GET " + api + "/merge_requests/4179/discussions?per_page=100": "discussions.json",
	})
	threads, err := p.Threads(context.Background(), project, 4179, "bef4c38e0000000000000000000000000000000000")
	if err != nil || len(threads) != 4 {
		t.Fatalf("Threads = %d, %v", len(threads), err)
	}
	byID := map[string]hostapi.Thread{}
	for _, th := range threads {
		byID[th.ID] = th
	}
	if th := byID["d1"]; th.Resolved || th.Outdated || th.Path != "app/remoteservice.go" || th.Line != 42 || len(th.Notes) != 2 || th.Notes[0].Author.Username != "ada" {
		t.Errorf("d1 = %+v", th)
	}
	if th := byID["d2"]; !th.Resolved || th.Outdated || th.Path != "README.md" {
		t.Errorf("d2 = %+v", th)
	}
	if th := byID["d3"]; !th.Outdated || th.Resolved || th.OldLine != 15 || th.Line != 0 {
		t.Errorf("d3 = %+v", th)
	}
	if th := byID["d4"]; th.Path != "" || th.Line != 0 || th.Notes[0].Body != "Overall looks good." {
		t.Errorf("d4 = %+v", th)
	}
	if _, ok := byID["d5"]; ok {
		t.Error("a system-only discussion became a thread")
	}
	unresolved := 0
	for _, th := range threads {
		if !th.Resolved {
			unresolved++
		}
	}
	if unresolved != 3 {
		t.Errorf("unresolved = %d, want 3", unresolved)
	}
}

func TestLatestPipelineAndJobsFromRecordedFixtures(t *testing.T) {
	sha := "bef4c38e0000000000000000000000000000000000"
	p, _ := newProvider(t, map[string]string{
		"GET " + api + "/pipelines?per_page=1&sha=" + sha:        "pipelines.json",
		"GET " + api + "/pipelines?per_page=1&sha=none":          "[]",
		"GET " + api + "/pipelines/2977141150/jobs?per_page=100": "jobs.json",
	})
	pipeline, ok, err := p.LatestPipeline(context.Background(), project, sha)
	if err != nil || !ok || pipeline.ID != 2977141150 || pipeline.Status != hostapi.CISuccess || pipeline.WebURL == "" {
		t.Fatalf("LatestPipeline = %+v, %v, %v", pipeline, ok, err)
	}
	if _, ok, err := p.LatestPipeline(context.Background(), project, "none"); ok || err != nil {
		t.Fatalf("a commit without pipelines: ok=%v err=%v", ok, err)
	}
	jobs, err := p.Jobs(context.Background(), project, pipeline.ID)
	if err != nil || len(jobs) == 0 || jobs[0].Name != "workload" || jobs[0].Stage != "build" || jobs[0].Status != hostapi.CISuccess ||
		jobs[0].Duration < 825*time.Second || jobs[0].Duration > 826*time.Second {
		t.Fatalf("Jobs = %+v, %v", jobs, err)
	}
}

func TestJobLogKeepsTheTailOnWholeLines(t *testing.T) {
	p, _ := newProvider(t, map[string]string{
		"GET " + api + "/jobs/17387105784/trace": "trace.txt",
	})
	log, truncated, err := p.JobLog(context.Background(), project, 17387105784, 100)
	if err != nil || !truncated || len(log) > 100 || !strings.HasPrefix(log, "line ") || !strings.Contains(log, "ERROR: Job failed") {
		t.Fatalf("JobLog = %q, truncated=%v, err=%v", log, truncated, err)
	}
	whole, truncated, _ := p.JobLog(context.Background(), project, 17387105784, 0)
	if truncated || !strings.HasPrefix(whole, "\x1b[0KRunning") {
		t.Fatalf("whole log: truncated=%v, starts %q", truncated, whole[:20])
	}
}

func TestIssueFromRecordedFixture(t *testing.T) {
	p, _ := newProvider(t, map[string]string{"GET " + api + "/issues/8666": "issue.json"})
	issue, err := p.Issue(context.Background(), project, 8666)
	if err != nil || issue.Number != 8666 || issue.State != "open" || !strings.HasPrefix(issue.Title, "mr create:") || issue.WebURL == "" {
		t.Fatalf("Issue = %+v, %v", issue, err)
	}
}

func TestAssignedIssuesFromRecordedFixture(t *testing.T) {
	p, _ := newProvider(t, map[string]string{"GET " + api + "/issues?assignee_username=jhebden&order_by=updated_at&per_page=50&state=opened": "assigned_issues.json"})
	issues, err := p.AssignedIssues(context.Background(), project, "jhebden")
	if err != nil || len(issues) != 3 || issues[0].Number != 8493 || issues[0].State != "open" || !strings.HasSuffix(issues[0].WebURL, "/8493") {
		t.Fatalf("AssignedIssues = %+v, %v", issues, err)
	}
}

func TestCreateChangeSendsADraftTitle(t *testing.T) {
	p, f := newProvider(t, map[string]string{"POST " + api + "/merge_requests": "created_merge_request.json"})
	change, err := p.CreateChange(context.Background(), project, hostapi.NewChange{
		SourceBranch: "feat/thing", TargetBranch: "main", Title: "Add the thing", Description: "Body", Draft: true,
	})
	if err != nil || change.Number != 4300 || !change.Draft {
		t.Fatalf("CreateChange = %+v, %v", change, err)
	}
	var sent map[string]string
	json.Unmarshal([]byte(f.requests[0].Body), &sent)
	want := map[string]string{"source_branch": "feat/thing", "target_branch": "main", "title": "Draft: Add the thing", "description": "Body"}
	for k, v := range want {
		if sent[k] != v {
			t.Errorf("sent %s = %q, want %q", k, sent[k], v)
		}
	}
}

func TestCurrentUserKeepsNameAndUsernameOnly(t *testing.T) {
	p, _ := newProvider(t, map[string]string{"GET /api/v4/user": "user.json"})
	user, err := p.CurrentUser(context.Background())
	if err != nil || user != (hostapi.User{Name: "Ada Lovelace", Username: "ada"}) {
		t.Fatalf("CurrentUser = %+v, %v", user, err)
	}
}

func TestRefusalsNameTheScope(t *testing.T) {
	p, _ := newProvider(t, map[string]string{
		"GET " + api + "/issues/1":        "403",
		"POST " + api + "/merge_requests": "403",
	})
	_, err := p.Issue(context.Background(), project, 1)
	var hostErr *hostapi.HostError
	if !errors.As(err, &hostErr) || hostErr.Scope != "read_api" || !strings.Contains(err.Error(), "read_api") {
		t.Fatalf("Issue 403: %v", err)
	}
	_, err = p.CreateChange(context.Background(), project, hostapi.NewChange{SourceBranch: "a", TargetBranch: "b", Title: "c"})
	if !errors.As(err, &hostErr) || hostErr.Scope != "api" {
		t.Fatalf("CreateChange 403: %v", err)
	}
}

// GITLAB_URL=https://gitlab.example.com/api/v4 GITLAB_TOKEN=… GITLAB_PROJECT=group/repo go test -run RealGitLab -v ./core/hostapi/gitlab
func TestAgainstRealGitLab(t *testing.T) {
	apiURL, token, realProject := os.Getenv("GITLAB_URL"), os.Getenv("GITLAB_TOKEN"), os.Getenv("GITLAB_PROJECT")
	if apiURL == "" || realProject == "" {
		t.Skip("set GITLAB_URL, GITLAB_TOKEN and GITLAB_PROJECT to run against a GitLab instance")
	}
	host := hostapi.Host{Name: "real", API: apiURL}
	client := hostapi.New(hostapi.Options{Enabled: hostapitest.AllowAll, Tokens: staticToken(token)})
	p := New(client, host)
	ctx := context.Background()
	if user, err := p.CurrentUser(ctx); err != nil || user.Username == "" {
		t.Fatalf("CurrentUser = %+v, %v", user, err)
	}
	changes, truncated, err := p.OpenChanges(ctx, realProject)
	if err != nil {
		t.Fatalf("OpenChanges: %v", err)
	}
	t.Logf("%d open merge requests (truncated=%v)", len(changes), truncated)
	for _, change := range changes {
		threads, err := p.Threads(ctx, realProject, change.Number, change.HeadSHA)
		if err != nil {
			t.Fatalf("Threads(%d): %v", change.Number, err)
		}
		reviews, err := p.Reviews(ctx, realProject, change)
		if err != nil {
			t.Fatalf("Reviews(%d): %v", change.Number, err)
		}
		t.Logf("!%d %s by %s, %d threads, reviews %v", change.Number, change.Title, change.Author.Username, len(threads), reviews)
	}
}

type staticToken string

func (s staticToken) Token(context.Context, string) (string, error) { return string(s), nil }

func TestBranchRulesFromRecordedFixtureAndSilentOnForbidden(t *testing.T) {
	p, _ := newProvider(t, map[string]string{"GET " + api + "/protected_branches?per_page=100": "protected_branches.json"})
	rules, err := p.BranchRules(context.Background(), project, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []hostapi.BranchRule{
		{Pattern: "main", NoPush: true, NoForcePush: true, NoDelete: true},
		{Pattern: "develop", RestrictedPush: true, NoDelete: true},
		{Pattern: "release/*", RestrictedPush: true, NoForcePush: true, NoDelete: true},
	}
	if len(rules) != len(want) {
		t.Fatalf("rules = %+v", rules)
	}
	for i := range want {
		if rules[i] != want[i] {
			t.Errorf("rule %d = %+v, want %+v", i, rules[i], want[i])
		}
	}

	p, _ = newProvider(t, map[string]string{"GET " + api + "/protected_branches?per_page=100": "403"})
	rules, err = p.BranchRules(context.Background(), project, nil)
	if err != nil || rules != nil {
		t.Fatalf("without the Maintainer role: %v, %v; want no rules and no error", rules, err)
	}
}

func TestClosedChangesFilesAndCommits(t *testing.T) {
	p, _ := newProvider(t, map[string]string{
		"GET " + api + "/merge_requests?order_by=updated_at&per_page=25&sort=desc&state=merged": `[{"iid": 1, "state": "merged", "title": "older", "updated_at": "2026-10-01T00:00:00Z", "source_branch": "a", "target_branch": "main", "author": {"username": "ada"}}]`,
		"GET " + api + "/merge_requests?order_by=updated_at&per_page=25&sort=desc&state=closed": `[{"iid": 2, "state": "closed", "title": "newer", "updated_at": "2026-10-05T00:00:00Z", "source_branch": "b", "target_branch": "main", "author": {"username": "bob"}}]`,
		"GET " + api + "/merge_requests/2/diffs?per_page=100":                                   `[{"old_path": "a.go", "new_path": "b.go", "renamed_file": true, "diff": "@@ -1,2 +1,2 @@\n-x\n+y\n"}, {"old_path": "n.go", "new_path": "n.go", "new_file": true, "diff": "--- /dev/null\n+++ b/n.go\n@@ -0,0 +1,2 @@\n+---\n+b\n"}]`,
		"GET " + api + "/merge_requests/2/commits?per_page=100":                                 `[{"id": "abc", "title": "fix: it", "author_name": "Bob", "created_at": "2026-10-02T03:04:05Z"}]`,
	})
	changes, truncated, err := p.ClosedChanges(context.Background(), project)
	if err != nil || truncated || len(changes) != 2 || changes[0].Number != 2 || changes[0].State != hostapi.ChangeClosed || changes[1].State != hostapi.ChangeMerged {
		t.Fatalf("ClosedChanges = %+v, truncated %v, %v", changes, truncated, err)
	}
	files, err := p.ChangeFiles(context.Background(), project, 2)
	if err != nil || len(files) != 2 || files[0] != (hostapi.ChangeFile{Path: "b.go", OldPath: "a.go", Status: "renamed", Additions: 1, Deletions: 1}) ||
		files[1] != (hostapi.ChangeFile{Path: "n.go", Status: "added", Additions: 2}) {
		t.Fatalf("ChangeFiles = %+v, %v", files, err)
	}
	commits, err := p.ChangeCommits(context.Background(), project, 2)
	if err != nil || len(commits) != 1 || commits[0] != (hostapi.ChangeCommit{SHA: "abc", Subject: "fix: it", Author: "Bob", Date: commits[0].Date}) || commits[0].Date.IsZero() {
		t.Fatalf("ChangeCommits = %+v, %v", commits, err)
	}
}
