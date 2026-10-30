//go:build !nohost

package app

import (
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostapi/hostapitest"
)

func TestReviewWritesRefreshTheThreadsAndReviews(t *testing.T) {
	repo := enabledGitLab(t)
	_, info, _, err := primaryHost(repo)
	if err != nil {
		t.Fatal(err)
	}
	api := "/api/v4/projects/group%2Fapp"
	mr := api + "/merge_requests/7"
	transport := &routeTransport{routes: map[string]string{
		"/api/v4/user":               `{"username": "ada", "name": "Ada"}`,
		mr:                           `{"iid": 7, "diff_refs": {"base_sha": "b", "head_sha": "h", "start_sha": "s"}}`,
		mr + "/discussions/d1/notes": `{"id": 1}`,
		mr + "/discussions/d1":       `{"id": "d1"}`,
		mr + "/discussions":          `{"id": "d2"}`,
		mr + "/notes":                `{"id": 2}`,
		mr + "/approve":              `{"approved": true}`,
		mr + "/discussions?per_page=100": `[{"id": "d1", "notes": [{"body": "Rename this.", "author": {"username": "bob", "name": "Bob"}, "resolved": true, "position": {"head_sha": "h", "new_path": "app/x.go", "new_line": 2}}]},
			{"id": "d2", "notes": [{"body": "Why?", "author": {"username": "ada", "name": "Ada"}, "resolved": false, "position": {"head_sha": "h", "new_path": "app/y.go", "old_line": 3}}]}]`,
		mr + "/reviewers": `[{"user": {"username": "ada", "name": "Ada"}, "state": "approved"}]`,
		mr + "/approvals": `{"approved_by": [{"user": {"username": "ada", "name": "Ada"}}]}`,
	}, hits: map[string]int{}}
	swapHostClient(t, transport)
	change := hostapi.Change{Number: 7, SourceBranch: "feature", TargetBranch: "main", HeadSHA: "h", State: hostapi.ChangeOpen, UnresolvedThreads: 1}
	hostClient.Cache.Put(info.Hostname, repo, cacheChanges, info.Project, cachedChanges{[]hostapi.Change{change}, false})
	var s HostService

	if caps, err := s.ReviewCapabilities(repo); err != nil || !caps.Reply || !caps.Resolve || caps.RequestChanges {
		t.Fatalf("capabilities = %+v, %v", caps, err)
	}
	threads, err := s.ReplyToThread(repo, 7, "d1", "Agreed.")
	if err != nil {
		t.Fatal(err)
	}
	if transport.hits[mr+"/discussions/d1/notes"] != 1 || transport.hits[mr+"/discussions?per_page=100"] != 1 {
		t.Fatalf("a reply posts once and reads the threads once: %v", transport.hits)
	}
	if len(threads.Threads) != 2 || threads.Total != 1 || threads.Unresolved["app/y.go"] != 1 {
		t.Errorf("threads after the reply = %+v", threads)
	}
	if list, _ := s.Changes(repo); list.Changes[0].UnresolvedThreads != 1 {
		t.Errorf("the open list follows: %+v", list.Changes[0])
	}
	if _, err := s.ResolveThread(repo, 7, "d1", false); err != nil || transport.hits[mr+"/discussions/d1"] != 1 {
		t.Fatalf("resolve = %v, hits %v", err, transport.hits)
	}
	if _, err := s.CommentOnLine(repo, 7, hostapi.NewComment{Path: "app/x.go", Line: 2, Body: "Here."}); err != nil || transport.hits[mr+"/discussions"] != 1 {
		t.Fatalf("comment = %v, hits %v", err, transport.hits)
	}
	if _, err := s.SubmitReview(repo, 7, hostapi.ReviewSubmission{Verdict: hostapi.ReviewApprove, Summary: "Good."}); err != nil {
		t.Fatal(err)
	}
	if transport.hits[mr+"/approve"] != 1 || transport.hits[mr+"/notes"] != 1 || transport.hits[mr+"/reviewers"] != 1 {
		t.Fatalf("an approval approves, notes the summary and reads the reviews: %v", transport.hits)
	}
	if list, _ := s.Changes(repo); len(list.Changes[0].Reviews) != 1 || list.Changes[0].Reviews[0].State != hostapi.ReviewApproved {
		t.Errorf("the open list carries the new review: %+v", list.Changes[0].Reviews)
	}
}

func TestReviewWritesNeedTheHostOn(t *testing.T) {
	useTempSettings(t)
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q")
	gitIn(t, repo, "remote", "add", "origin", "git@gitlab.example.com:group/app.git")
	swapHostClient(t, hostapitest.Forbid(t))
	var s HostService
	if caps, err := s.ReviewCapabilities(repo); err != nil || caps != (hostapi.ReviewCapabilities{}) {
		t.Fatalf("capabilities with the host off = %+v, %v", caps, err)
	}
	if _, err := s.ReplyToThread(repo, 7, "d1", "x"); err == nil {
		t.Fatal("a write with the host off fails before any request")
	}
}
