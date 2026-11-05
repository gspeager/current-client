package git

import (
	"context"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

// repoWithChangeHead returns a repository whose remote publishes
// refs/merge-requests/3/head at a commit main doesn't have.
func repoWithChangeHead(t *testing.T, remote string) (dir, headSHA string) {
	t.Helper()
	remoteDir := t.TempDir() + "/" + remote + ".git"
	gittest.Run(t, "", "init", "--bare", "-b", "main", remoteDir)
	dir = gittest.InitRepo(t)
	gittest.Run(t, dir, "remote", "add", remote, remoteDir)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	gittest.Run(t, dir, "push", "-u", remote, "main")
	gittest.CommitFile(t, dir, "file.txt", "v2", "on the merge request")
	headSHA = gittest.Run(t, dir, "rev-parse", "HEAD")
	gittest.Run(t, dir, "push", remote, "HEAD:refs/merge-requests/3/head")
	gittest.Run(t, dir, "reset", "--hard", "HEAD~1")
	return dir, headSHA
}

func TestFetchHostChangeHeadLandsOutsideListRefs(t *testing.T) {
	dir, want := repoWithChangeHead(t, "origin")
	ctx := context.Background()

	if err := FetchHostChangeHead(ctx, dir, "origin", "refs/merge-requests/3/head", 3); err != nil {
		t.Fatalf("FetchHostChangeHead: %v", err)
	}
	if got := gittest.Run(t, dir, "rev-parse", HostChangeRef("origin", 3)); got != want {
		t.Fatalf("fetched head = %s, want %s", got, want)
	}

	refs, err := ListRefs(ctx, dir)
	if err != nil {
		t.Fatalf("ListRefs: %v", err)
	}
	for _, ref := range refs {
		if ref.SHA == want || strings.Contains(ref.Name, "mr/3") {
			t.Fatalf("ListRefs shows the fetched head as %+v", ref)
		}
	}
}

func TestAllRefReadsSkipTheAppsOwnRefs(t *testing.T) {
	dir := gittest.InitRepo(t)
	ctx := context.Background()
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	gittest.CommitFile(t, dir, "hidden.txt", "v1", "only on the app's refs")
	gittest.Run(t, dir, "update-ref", HostChangeRef("origin", 3), "HEAD")
	gittest.Run(t, dir, "update-ref", appRefNamespace+"snapshots/1", "HEAD")
	hidden := gittest.Run(t, dir, "rev-parse", "HEAD")
	gittest.Run(t, dir, "reset", "--hard", "HEAD~1")

	history, err := History(ctx, dir, 100, 0, HistoryFilter{Ref: "--all"})
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range history {
		if entry.SHA == hidden {
			t.Fatal("History's All branches lists a commit only the app's refs reach")
		}
	}

	activity, err := CommitActivity(ctx, dir, 7)
	if err != nil {
		t.Fatal(err)
	}
	commits := 0
	for _, day := range activity {
		commits += day.Commits
	}
	if commits != 1 {
		t.Fatalf("Activity counts %d commits, want 1", commits)
	}

	stats, err := ComputeRepoStats(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalCommits != 1 {
		t.Fatalf("stats count %d commits, want 1", stats.TotalCommits)
	}

	churn, err := ComputeFileChurn(ctx, dir, 7, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range churn {
		if file.Path == "hidden.txt" {
			t.Fatal("churn counts a file only the app's refs reach")
		}
	}
}

func TestDeleteHostRefsLeavesLocalRefsAndFindsOldLayout(t *testing.T) {
	dir := gittest.InitRepo(t)
	ctx := context.Background()
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	snapshot := appRefNamespace + "snapshots/1"
	gittest.Run(t, dir, "update-ref", snapshot, "HEAD")
	gittest.Run(t, dir, "update-ref", HostChangeRef("snapshots", 1), "HEAD")
	gittest.Run(t, dir, "update-ref", HostChangeRef("group/origin", 2), "HEAD")
	gittest.Run(t, dir, "update-ref", appRefNamespace+"origin/mr/9", "HEAD")

	if err := DeleteHostRefs(ctx, dir, "snapshots"); err != nil {
		t.Fatal(err)
	}
	if err := DeleteHostRefs(ctx, dir, "origin"); err != nil {
		t.Fatal(err)
	}
	left := gittest.Run(t, dir, "for-each-ref", "--format=%(refname)", appRefNamespace)
	if want := HostChangeRef("group/origin", 2) + "\n" + snapshot; left != want {
		t.Fatalf("refs left = %q, want %q", left, want)
	}

	if err := DeleteHostRefs(ctx, dir, ""); err != nil {
		t.Fatal(err)
	}
	if got := gittest.Run(t, dir, "for-each-ref", "--format=%(refname)", appRefNamespace); got != snapshot {
		t.Fatalf("refs left after deleting all hosts' = %q, want %q", got, snapshot)
	}
}

func TestDeleteHostRefsKeepsOtherRemotesAndBranches(t *testing.T) {
	dir, _ := repoWithChangeHead(t, "origin")
	ctx := context.Background()
	if err := FetchHostChangeHead(ctx, dir, "origin", "refs/merge-requests/3/head", 3); err != nil {
		t.Fatalf("FetchHostChangeHead: %v", err)
	}
	other := HostChangeRef("upstream", 7)
	gittest.Run(t, dir, "update-ref", other, "HEAD")

	if err := DeleteHostRefs(ctx, dir, "origin"); err != nil {
		t.Fatalf("DeleteHostRefs: %v", err)
	}
	if got := gittest.Run(t, dir, "for-each-ref", "--format=%(refname)", hostRefPrefix); got != other {
		t.Fatalf("refs left after deleting origin's = %q, want only %q", got, other)
	}
	gittest.Run(t, dir, "rev-parse", "--verify", "refs/heads/main")

	if err := DeleteHostRefs(ctx, dir, ""); err != nil {
		t.Fatalf("DeleteHostRefs(all): %v", err)
	}
	if got := gittest.Run(t, dir, "for-each-ref", hostRefPrefix); got != "" {
		t.Fatalf("refs left after deleting all: %q", got)
	}
}
