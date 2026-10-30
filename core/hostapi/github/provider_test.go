package github

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

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostapi/hostapitest"
)

const project = "cli/cli"

const api = "/repos/cli/cli"

type request struct {
	Method, Path, Query, Body string
}

// Fixtures were recorded from api.github.com for cli/cli; the check-run
// list is cut to five runs and one reply was added to the review comments.
type fixtures struct {
	mu       sync.Mutex
	requests []request
	// routes map "METHOD /<path>?<query>" to a testdata file, a literal body
	// when it starts with "[" or "{", a status such as "403" for a refusal, or
	// "redirect:<path>" for an answer that points elsewhere on the server.
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
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	case route == "403":
		http.Error(w, `{"message":"Resource not accessible by integration"}`, http.StatusForbidden)
	case isStatus(route):
		status, _ := strconv.Atoi(route)
		http.Error(w, `{"message":"`+http.StatusText(status)+`"}`, status)
	case strings.HasPrefix(route, "redirect:"):
		http.Redirect(w, r, strings.TrimPrefix(route, "redirect:"), http.StatusFound)
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

// isStatus is a route that is a status alone, "404" or "405".
func isStatus(route string) bool {
	return len(route) == 3 && route >= "400" && route <= "599"
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

func newProvider(t *testing.T, routes map[string]string) (*Provider, *fixtures) {
	t.Helper()
	hostapitest.IsolateGitConfig(t)
	f := &fixtures{routes: routes}
	server := httptest.NewServer(f)
	t.Cleanup(server.Close)
	host := hostapi.Host{Name: "github.example.com", API: server.URL + "/api/v3"}
	client := hostapi.New(hostapi.Options{Enabled: hostapitest.AllowAll, PageCap: 3})
	return New(client, host), f
}

const v3 = "/api/v3" + api

func TestOpenChangesFromRecordedFixtureNamesAFork(t *testing.T) {
	p, _ := newProvider(t, map[string]string{"GET " + v3 + "/pulls?per_page=100&state=open": "pulls.json"})
	changes, truncated, err := p.OpenChanges(context.Background(), project)
	if err != nil || truncated || len(changes) != 2 {
		t.Fatalf("OpenChanges = %d changes, truncated %v, %v", len(changes), truncated, err)
	}
	fork, own := changes[0], changes[1]
	if fork.Number != 14783 || fork.State != hostapi.ChangeOpen || !fork.Draft || fork.SourceBranch != "release-download-verify" ||
		fork.SourceProject != "mrecachinas/cli" || fork.TargetBranch != "trunk" || fork.Author != (hostapi.User{Name: "mrecachinas", Username: "mrecachinas"}) ||
		fork.HeadSHA != "654d851d8145b2912fe474ceb0a47681854b37e2" || strings.Join(fork.Labels, ",") != "needs-triage,external" ||
		fork.CI != hostapi.CIUnknown || fork.WebURL != "https://github.com/cli/cli/pull/14783" {
		t.Fatalf("fork = %+v", fork)
	}
	if own.Number != 14781 || own.SourceProject != "" || own.Author.Username != "williammartin" {
		t.Fatalf("own = %+v", own)
	}
}

func TestChangeReadsStateAndConflicts(t *testing.T) {
	p, _ := newProvider(t, map[string]string{"GET " + v3 + "/pulls/11000": "pull.json"})
	change, err := p.Change(context.Background(), project, 11000)
	if err != nil || change.Number != 11000 || change.State != hostapi.ChangeClosed || change.HasConflicts || !change.Draft ||
		change.SourceProject != "ejahnGithub/cli" || change.SourceBranch != "eugene/release-verify" {
		t.Fatalf("Change = %+v, %v", change, err)
	}

	p, _ = newProvider(t, map[string]string{"GET " + v3 + "/pulls/1": `{"number": 1, "state": "closed", "merged_at": "2026-10-01T00:00:00Z", "mergeable": false, "head": {"ref": "a", "sha": "b"}, "base": {"ref": "main"}, "user": {"login": "ada"}}`})
	change, err = p.Change(context.Background(), project, 1)
	if err != nil || change.State != hostapi.ChangeMerged || !change.HasConflicts || change.Author != (hostapi.User{Name: "ada", Username: "ada"}) {
		t.Fatalf("merged change = %+v, %v", change, err)
	}
}

func TestThreadsGroupRepliesAndMarkOutdatedOnes(t *testing.T) {
	p, _ := newProvider(t, map[string]string{"GET " + v3 + "/pulls/14783/comments?per_page=100": "review_comments.json"})
	head := "654d851d8145b2912fe474ceb0a47681854b37e2"
	threads, err := p.Threads(context.Background(), project, 14783, head)
	if err != nil || len(threads) != 3 {
		t.Fatalf("Threads = %d, %v", len(threads), err)
	}
	first := threads[0]
	if first.ID != "4339733038" || first.Path != "pkg/cmd/release/download/download.go" || first.Line != 600 || first.OldLine != 0 ||
		first.Outdated || first.Resolved || len(first.Notes) != 2 || first.Notes[1].Author.Username != "mrecachinas" || first.Notes[1].Body != "Agreed, will change." {
		t.Fatalf("first thread = %+v", first)
	}
	if threads[1].Line != 303 || len(threads[1].Notes) != 1 {
		t.Fatalf("second thread = %+v", threads[1])
	}

	threads, err = p.Threads(context.Background(), project, 14783, "0000000")
	if err != nil || !threads[0].Outdated {
		t.Fatalf("against another head: %+v, %v", threads[0], err)
	}
}

func TestReviewsKeepEachReviewersVerdict(t *testing.T) {
	p, _ := newProvider(t, map[string]string{"GET " + v3 + "/pulls/14716/reviews?per_page=100": "reviews.json"})
	requested := hostapi.Review{Reviewer: hostapi.User{Name: "niik", Username: "niik"}, State: hostapi.ReviewRequested}
	reviews, err := p.Reviews(context.Background(), project, hostapi.Change{Number: 14716, Reviews: []hostapi.Review{requested}})
	want := []hostapi.Review{{Reviewer: hostapi.User{Name: "babakks", Username: "babakks"}, State: hostapi.ReviewApproved}, requested}
	if err != nil || !reflect.DeepEqual(reviews, want) {
		t.Fatalf("Reviews = %+v, %v", reviews, err)
	}

	p, _ = newProvider(t, map[string]string{"GET " + v3 + "/pulls/1/reviews?per_page=100": `[
		{"user": {"login": "ada"}, "state": "COMMENTED"},
		{"user": {"login": "ada"}, "state": "APPROVED"},
		{"user": {"login": "ada"}, "state": "COMMENTED"},
		{"user": {"login": "bo"}, "state": "CHANGES_REQUESTED"},
		{"user": {"login": "cy"}, "state": "DISMISSED"},
		{"user": {"login": "di"}, "state": "PENDING"}]`})
	again := hostapi.Review{Reviewer: hostapi.User{Name: "bo", Username: "bo"}, State: hostapi.ReviewRequested}
	reviews, err = p.Reviews(context.Background(), project, hostapi.Change{Number: 1, Reviews: []hostapi.Review{again}})
	want = []hostapi.Review{
		{Reviewer: hostapi.User{Name: "ada", Username: "ada"}, State: hostapi.ReviewApproved},
		again,
		{Reviewer: hostapi.User{Name: "cy", Username: "cy"}, State: hostapi.ReviewCommented},
	}
	if err != nil || !reflect.DeepEqual(reviews, want) {
		t.Fatalf("Reviews = %+v, %v", reviews, err)
	}
}

func TestOpenChangesCarryRequestedReviewers(t *testing.T) {
	p, _ := newProvider(t, map[string]string{"GET " + v3 + "/pulls?per_page=100&state=open": `[{"number": 2, "state": "open", "head": {"ref": "a"}, "base": {"ref": "main"}, "user": {"login": "ada"}, "requested_reviewers": [{"login": "bo", "avatar_url": "https://avatars.example/bo"}]}]`})
	changes, _, err := p.OpenChanges(context.Background(), project)
	if err != nil || len(changes) != 1 || !reflect.DeepEqual(changes[0].Reviews, []hostapi.Review{{Reviewer: hostapi.User{Name: "bo", Username: "bo"}, State: hostapi.ReviewRequested}}) {
		t.Fatalf("OpenChanges = %+v, %v", changes, err)
	}
}

func TestLatestPipelineCombinesCheckRunsAndStatuses(t *testing.T) {
	sha := "654d851d8145b2912fe474ceb0a47681854b37e2"
	p, f := newProvider(t, map[string]string{
		"GET " + v3 + "/commits/" + sha + "/check-runs?per_page=100": "check_runs.json",
		"GET " + v3 + "/commits/" + sha + "/status":                  "status.json",
	})
	pipeline, ok, err := p.LatestPipeline(context.Background(), project, sha)
	if err != nil || !ok || pipeline.Status != hostapi.CISkipped || pipeline.ID != 0x654d851d8145b ||
		pipeline.WebURL != "https://github.example.com/cli/cli/commit/"+sha+"/checks" {
		t.Fatalf("LatestPipeline = %+v, %v, %v", pipeline, ok, err)
	}

	// The id is 13 hex digits, exact as a JavaScript number, and the host
	// takes the abbreviated SHA back.
	if float64(pipeline.ID) > 1<<53 {
		t.Fatalf("pipeline id %d is not exact as a JavaScript number", pipeline.ID)
	}
	short := "654d851d8145b"
	f.routes["GET "+v3+"/commits/"+short+"/check-runs?per_page=100"] = "check_runs.json"
	f.routes["GET "+v3+"/commits/"+short+"/status"] = `{"state": "failure", "statuses": [{"id": 7, "context": "ci/circleci", "state": "failure", "target_url": "https://circleci.example.com/1"}]}`
	jobs, err := p.Jobs(context.Background(), project, pipeline.ID)
	if err != nil || len(jobs) != 6 {
		t.Fatalf("Jobs = %d, %v", len(jobs), err)
	}
	if first := jobs[0]; first.ID != 119400197497 || first.Name != "label-external" || first.Stage != "github-actions" || first.Status != hostapi.CISkipped ||
		!strings.HasSuffix(first.WebURL, "/job/119400197497") {
		t.Fatalf("first job = %+v", jobs[0])
	}
	if jobs[5] != (hostapi.Job{ID: 7, Name: "ci/circleci", Stage: "status", Status: hostapi.CIFailed, WebURL: "https://circleci.example.com/1"}) {
		t.Fatalf("status job = %+v", jobs[5])
	}

	empty := "0000000000000000000000000000000000000000"
	f.routes["GET "+v3+"/commits/"+empty+"/check-runs?per_page=100"] = `{"total_count": 0, "check_runs": []}`
	f.routes["GET "+v3+"/commits/"+empty+"/status"] = `{"state": "pending", "statuses": []}`
	if _, ok, err := p.LatestPipeline(context.Background(), project, empty); ok || err != nil {
		t.Fatalf("a commit without CI: %v, %v", ok, err)
	}
}

func TestCombineReadsSeveralResultsAsOne(t *testing.T) {
	cases := []struct {
		in   []hostapi.CIStatus
		want hostapi.CIStatus
	}{
		{[]hostapi.CIStatus{hostapi.CISuccess, hostapi.CIFailed, hostapi.CIRunning}, hostapi.CIFailed},
		{[]hostapi.CIStatus{hostapi.CISuccess, hostapi.CIRunning}, hostapi.CIRunning},
		{[]hostapi.CIStatus{hostapi.CISuccess, hostapi.CIPending}, hostapi.CIPending},
		{[]hostapi.CIStatus{hostapi.CISuccess, hostapi.CISkipped}, hostapi.CISuccess},
		{[]hostapi.CIStatus{hostapi.CISkipped}, hostapi.CISkipped},
		{nil, hostapi.CIUnknown},
	}
	for _, c := range cases {
		if got := combine(c.in); got != c.want {
			t.Errorf("combine(%v) = %q, want %q", c.in, got, c.want)
		}
	}
	run := checkRun{Status: "completed", Conclusion: "failure"}
	if checkStatus(run) != hostapi.CIFailed || checkStatus(checkRun{Status: "in_progress"}) != hostapi.CIRunning || checkStatus(checkRun{Status: "queued"}) != hostapi.CIPending {
		t.Error("check run statuses")
	}
}

func TestJobLogFollowsTheRedirectAndKeepsTheTail(t *testing.T) {
	p, _ := newProvider(t, map[string]string{
		"GET " + v3 + "/actions/jobs/5/logs": "redirect:/blob/5",
		"GET /blob/5":                        "{\"not\": \"json\"}\nline two\nline three\n",
	})
	log, truncated, err := p.JobLog(context.Background(), project, 5, 15)
	if err != nil || !truncated || log != "line three\n" {
		t.Fatalf("JobLog = %q, %v, %v", log, truncated, err)
	}
	log, truncated, err = p.JobLog(context.Background(), project, 5, 0)
	if err != nil || truncated || !strings.HasSuffix(log, "line three\n") {
		t.Fatalf("whole log = %q, %v, %v", log, truncated, err)
	}
}

func TestIssueFromRecordedFixture(t *testing.T) {
	p, _ := newProvider(t, map[string]string{"GET " + v3 + "/issues/10000": "issue.json"})
	issue, err := p.Issue(context.Background(), project, 10000)
	if err != nil || issue.Number != 10000 || issue.State != "closed" || !strings.Contains(issue.Title, "allow-forking") || issue.WebURL != "https://github.com/cli/cli/issues/10000" {
		t.Fatalf("Issue = %+v, %v", issue, err)
	}
}

func TestAssignedIssuesLeaveOutPullRequests(t *testing.T) {
	p, _ := newProvider(t, map[string]string{"GET " + v3 + "/issues?assignee=BagToad&per_page=50&sort=updated&state=open": "assigned_issues.json"})
	issues, err := p.AssignedIssues(context.Background(), project, "BagToad")
	if err != nil || len(issues) != 4 || issues[0].Number != 13968 || issues[0].State != "open" || issues[0].WebURL != "https://github.com/cli/cli/issues/13968" {
		t.Fatalf("AssignedIssues = %+v, %v", issues, err)
	}

	p, _ = newProvider(t, map[string]string{"GET " + v3 + "/issues?assignee=ada&per_page=50&sort=updated&state=open": `[{"number": 1, "title": "A pull", "pull_request": {"url": "x"}}, {"number": 2, "title": "An issue"}]`})
	issues, err = p.AssignedIssues(context.Background(), project, "ada")
	if err != nil || len(issues) != 1 || issues[0].Number != 2 {
		t.Fatalf("AssignedIssues = %+v, %v", issues, err)
	}
}

func TestProjectIssuesByStateLeaveOutPullRequests(t *testing.T) {
	p, _ := newProvider(t, map[string]string{
		"GET " + v3 + "/issues?per_page=50&sort=updated&state=closed": `[{"number": 1, "title": "A pull", "state": "closed", "pull_request": {"url": "x"}}, {"number": 2, "title": "Done", "state": "closed"}]`,
	})
	issues, err := p.ProjectIssues(context.Background(), project, false)
	if err != nil || len(issues) != 1 || issues[0].Number != 2 || issues[0].State != "closed" {
		t.Fatalf("ProjectIssues = %+v, %v", issues, err)
	}
}

func TestCreateChangeSendsHeadBaseAndDraft(t *testing.T) {
	p, f := newProvider(t, map[string]string{"POST " + v3 + "/pulls": "pull.json"})
	change, err := p.CreateChange(context.Background(), project, hostapi.NewChange{
		SourceBranch: "feat/thing", TargetBranch: "trunk", Title: "Add the thing", Description: "Body", Draft: true,
	})
	if err != nil || change.Number != 11000 {
		t.Fatalf("CreateChange = %+v, %v", change, err)
	}
	var sent map[string]any
	json.Unmarshal([]byte(f.requests[0].Body), &sent)
	want := map[string]any{"head": "feat/thing", "base": "trunk", "title": "Add the thing", "body": "Body", "draft": true}
	for k, v := range want {
		if sent[k] != v {
			t.Errorf("sent %s = %v, want %v", k, sent[k], v)
		}
	}
}

func TestBranchRulesPerBranchAndSilentWhereUnavailable(t *testing.T) {
	p, f := newProvider(t, map[string]string{
		"GET " + v3 + "/rules/branches/trunk":         "rules.json",
		"GET " + v3 + "/rules/branches/main":          `[{"type": "deletion"}, {"type": "non_fast_forward"}, {"type": "pull_request", "parameters": {}}]`,
		"GET " + v3 + "/rules/branches/release%2F1.0": `[{"type": "update"}]`,
	})
	rules, err := p.BranchRules(context.Background(), project, []string{"trunk", "main", "release/1.0"})
	if err != nil {
		t.Fatal(err)
	}
	want := []hostapi.BranchRule{
		{Pattern: "main", NoPush: true, NoForcePush: true, NoDelete: true},
		{Pattern: "release/1.0", RestrictedPush: true},
	}
	if len(rules) != 2 || rules[0] != want[0] || rules[1] != want[1] {
		t.Fatalf("rules = %+v", rules)
	}
	if f.count("GET", v3+"/rules/branches/release%2F1.0") != 1 {
		t.Fatalf("requests: %+v", f.requests)
	}

	p, _ = newProvider(t, map[string]string{"GET " + v3 + "/rules/branches/main": "404"})
	rules, err = p.BranchRules(context.Background(), project, []string{"main", "other"})
	if err != nil || rules != nil {
		t.Fatalf("without rulesets: %v, %v", rules, err)
	}
}

func TestCurrentUserAndRefusalsNameTheScope(t *testing.T) {
	p, _ := newProvider(t, map[string]string{"GET /api/v3/user": `{"login": "ada", "name": "Ada Lovelace", "avatar_url": "https://avatars.example.com/1"}`})
	user, err := p.CurrentUser(context.Background())
	if err != nil || user != (hostapi.User{Name: "Ada Lovelace", Username: "ada"}) {
		t.Fatalf("CurrentUser = %+v, %v", user, err)
	}

	p, _ = newProvider(t, map[string]string{"POST " + v3 + "/pulls": "403"})
	_, err = p.CreateChange(context.Background(), project, hostapi.NewChange{SourceBranch: "a", TargetBranch: "b", Title: "c"})
	var hostErr *hostapi.HostError
	if !errors.As(err, &hostErr) || hostErr.Scope != "repo" {
		t.Fatalf("CreateChange 403: %v", err)
	}
}

func TestWebRootForGitHubComAndEnterprise(t *testing.T) {
	client := hostapi.New(hostapi.Options{Enabled: hostapitest.AllowAll})
	if root := New(client, hostapi.Host{Name: "github.com", API: "https://api.github.com"}).webRoot(); root != "https://github.com/" {
		t.Errorf("github.com root = %q", root)
	}
	if root := New(client, hostapi.Host{Name: "ghe.example.com", API: "https://ghe.example.com/api/v3"}).webRoot(); root != "https://ghe.example.com/" {
		t.Errorf("enterprise root = %q", root)
	}
}

// GITHUB_URL=https://api.github.com GITHUB_TOKEN=… GITHUB_PROJECT=owner/repo go test -run RealGitHub -v ./core/hostapi/github
func TestAgainstRealGitHub(t *testing.T) {
	apiURL, token, realProject := os.Getenv("GITHUB_URL"), os.Getenv("GITHUB_TOKEN"), os.Getenv("GITHUB_PROJECT")
	if apiURL == "" || realProject == "" {
		t.Skip("set GITHUB_URL, GITHUB_TOKEN and GITHUB_PROJECT to run against GitHub")
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
	t.Logf("%d open pull requests (truncated=%v)", len(changes), truncated)
	for _, change := range changes {
		threads, err := p.Threads(ctx, realProject, change.Number, change.HeadSHA)
		if err != nil {
			t.Fatalf("Threads(%d): %v", change.Number, err)
		}
		reviews, err := p.Reviews(ctx, realProject, change)
		if err != nil {
			t.Fatalf("Reviews(%d): %v", change.Number, err)
		}
		pipeline, ok, err := p.LatestPipeline(ctx, realProject, change.HeadSHA)
		t.Logf("#%d %s by %s, %d threads, reviews %v, CI %v %q %v", change.Number, change.Title, change.Author.Username, len(threads), reviews, ok, pipeline.Status, err)
	}
}

type staticToken string

func (s staticToken) Token(context.Context, string) (string, error) { return string(s), nil }

func TestClosedChangesFilesAndCommits(t *testing.T) {
	p, f := newProvider(t, map[string]string{
		"GET " + v3 + "/pulls?direction=desc&per_page=50&sort=updated&state=closed": `[{"number": 1, "state": "closed", "merged_at": "2026-10-01T00:00:00Z", "head": {"ref": "a", "sha": "b"}, "base": {"ref": "main"}, "user": {"login": "ada"}}]`,
		"GET " + v3 + "/pulls/1/files?per_page=100":                                 `[{"filename": "cmd/new.go", "status": "added", "additions": 10, "deletions": 0}, {"filename": "cmd/b.go", "previous_filename": "cmd/a.go", "status": "renamed", "additions": 1, "deletions": 1}, {"filename": "old.go", "status": "removed", "additions": 0, "deletions": 4}]`,
		"GET " + v3 + "/pulls/1/commits?per_page=100":                               `[{"sha": "abc", "commit": {"message": "feat: add\n\nbody", "author": {"name": "Ada", "date": "2026-10-02T03:04:05Z"}}}]`,
	})
	changes, truncated, err := p.ClosedChanges(context.Background(), project)
	if err != nil || truncated || len(changes) != 1 || changes[0].State != hostapi.ChangeMerged {
		t.Fatalf("ClosedChanges = %+v, truncated %v, %v", changes, truncated, err)
	}
	files, err := p.ChangeFiles(context.Background(), project, 1)
	if err != nil || len(files) != 3 || files[0] != (hostapi.ChangeFile{Path: "cmd/new.go", Status: "added", Additions: 10}) ||
		files[1] != (hostapi.ChangeFile{Path: "cmd/b.go", OldPath: "cmd/a.go", Status: "renamed", Additions: 1, Deletions: 1}) ||
		files[2].Status != "deleted" {
		t.Fatalf("ChangeFiles = %+v, %v", files, err)
	}
	commits, err := p.ChangeCommits(context.Background(), project, 1)
	if err != nil || len(commits) != 1 || commits[0].SHA != "abc" || commits[0].Subject != "feat: add" || commits[0].Author != "Ada" || commits[0].Date.IsZero() {
		t.Fatalf("ChangeCommits = %+v, %v", commits, err)
	}
	if n := f.count("GET", v3+"/pulls"); n != 1 {
		t.Fatalf("pulls fetched %d times", n)
	}
}
