package github

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
)

func TestIssueTrackerKeysAreNumbers(t *testing.T) {
	var p Provider
	if got := p.KeyPattern().FindAllStringSubmatch("Fix #12, see group/x#7 and v1.2#3x", -1); len(got) != 2 || got[0][1] != "12" || got[1][1] != "7" {
		t.Fatalf("KeyPattern matches = %v", got)
	}
	_, err := p.IssueByKey(context.Background(), "g/r", "PROJ-7")
	var hostErr *hostapi.HostError
	if !errors.As(err, &hostErr) || hostErr.Status != 404 {
		t.Fatalf("a non-numeric key should be a 404, got %v", err)
	}
}

func TestIssueByKeyAndCommentsCarryTheDetails(t *testing.T) {
	p, _ := newProvider(t, map[string]string{"GET " + v3 + "/issues/4": `{"number": 4, "title": "Login fails", "state": "open", "body": "Steps", "labels": [{"name": "bug"}], "milestone": {"title": "0.5.0"}, "user": {"login": "ada"}, "html_url": "https://github.example.com/cli/cli/issues/4"}`, "GET " + v3 + "/issues/4/comments?per_page=100": `[{"user": {"login": "grace"}, "body": "Same here", "created_at": "2026-10-20T10:00:00Z"}]`})
	issue, err := p.IssueDetails(context.Background(), project, "4")
	if err != nil || issue.Key != "4" || issue.Title != "Login fails" || issue.State != "open" || issue.Description != "Steps" ||
		!reflect.DeepEqual(issue.Labels, []string{"bug"}) || issue.Milestone != "0.5.0" || issue.Author.Username != "ada" {
		t.Fatalf("IssueDetails = %+v, %v", issue, err)
	}
	notes, err := p.IssueComments(context.Background(), project, "4")
	if err != nil || len(notes) != 1 || notes[0].Author.Username != "grace" || notes[0].Body != "Same here" {
		t.Fatalf("IssueComments = %+v, %v", notes, err)
	}
	if p.KeyInBranch("12-fix-login") != "12" || p.KeyInBranch("fix-login") != "" || p.Reference("12") != "#12" {
		t.Fatal("branch keys and references")
	}
}
