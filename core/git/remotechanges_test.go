package git

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

// hostWithChanges returns a repository whose remote publishes branches
// feature and old plus the given refs, each pointing at the branch named
// in refs' values or at a commit no branch has.
func hostWithChanges(t *testing.T, refs map[string]string) string {
	t.Helper()
	remoteDir := t.TempDir() + "/remote.git"
	gittest.Run(t, "", "init", "--bare", "-b", "main", remoteDir)
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "remote", "add", "origin", remoteDir)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	gittest.Run(t, dir, "push", "-q", "origin", "main")
	for _, branch := range []string{"feature", "old", "gone"} {
		gittest.Run(t, dir, "switch", "-q", "-c", branch, "main")
		gittest.CommitFile(t, dir, branch+".txt", branch, "on "+branch)
		gittest.Run(t, dir, "push", "-q", "origin", branch)
	}
	gittest.Run(t, dir, "push", "-q", "origin", "--delete", "gone")
	for ref, branch := range refs {
		gittest.Run(t, dir, "push", "-q", "origin", branch+":"+ref)
	}
	return dir
}

func TestListRemoteChangesOnGitHubStyleHost(t *testing.T) {
	dir := hostWithChanges(t, map[string]string{
		"refs/pull/1/head":  "feature",
		"refs/pull/1/merge": "feature",
		"refs/pull/2/head":  "old",  // merged, branch kept: no merge ref
		"refs/pull/3/head":  "gone", // branch deleted
		"refs/pull/3/merge": "gone",
	})
	got, err := ListRemoteChanges(context.Background(), dir, "origin")
	if err != nil {
		t.Fatalf("ListRemoteChanges: %v", err)
	}
	want := RemoteChanges{Kind: PullRefs, ByBranch: map[string]int{"feature": 1}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestListRemoteChangesWithoutMergeRefsMatchesTips(t *testing.T) {
	dir := hostWithChanges(t, map[string]string{
		"refs/pull/4/head": "feature",
		"refs/pull/5/head": "feature", // reopened as a newer pull request
		"refs/pull/6/head": "gone",
	})
	got, err := ListRemoteChanges(context.Background(), dir, "origin")
	if err != nil {
		t.Fatalf("ListRemoteChanges: %v", err)
	}
	want := RemoteChanges{Kind: PullRefs, ByBranch: map[string]int{"feature": 5}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestListRemoteChangesOnGitLabIgnoresMergeRefs(t *testing.T) {
	dir := hostWithChanges(t, map[string]string{
		"refs/merge-requests/7/head":  "feature",
		"refs/merge-requests/8/head":  "old",
		"refs/merge-requests/8/merge": "old",
		"refs/merge-requests/9/head":  "gone",
		"refs/merge-requests/9/merge": "gone",
	})
	got, err := ListRemoteChanges(context.Background(), dir, "origin")
	if err != nil {
		t.Fatalf("ListRemoteChanges: %v", err)
	}
	want := RemoteChanges{Kind: MergeRequestRefs, ByBranch: map[string]int{"feature": 7, "old": 8}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestListRemoteChangesOnHostWithoutChangeRefs(t *testing.T) {
	dir := hostWithChanges(t, nil)
	got, err := ListRemoteChanges(context.Background(), dir, "origin")
	if err != nil {
		t.Fatalf("ListRemoteChanges: %v", err)
	}
	if got.Kind != "" || len(got.ByBranch) != 0 {
		t.Fatalf("got %+v, want nothing", got)
	}
	if _, err := ListRemoteChanges(context.Background(), dir, "nowhere"); err == nil {
		t.Fatal("an unknown remote should fail")
	}
}

// REMOTE_CHANGES_URL=https://gitlab.com/group/project.git go test -run RealHost -v ./core/git
func TestListRemoteChangesAgainstRealHost(t *testing.T) {
	url := os.Getenv("REMOTE_CHANGES_URL")
	if url == "" {
		t.Skip("set REMOTE_CHANGES_URL to a hosted repository with open changes")
	}
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "remote", "add", "origin", url)
	got, err := ListRemoteChanges(context.Background(), dir, "origin")
	if err != nil {
		t.Fatalf("ListRemoteChanges: %v", err)
	}
	t.Logf("%s: %d branches with an open change", got.Kind, len(got.ByBranch))
	for branch, number := range got.ByBranch {
		t.Logf("%s → %d", branch, number)
	}
}
