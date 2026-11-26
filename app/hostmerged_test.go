//go:build !nohost

package app

import (
	"context"
	"testing"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/internal/gittest"
)

func TestMergedBranchesNamesTheHostChangeAndFlagsNewerCommits(t *testing.T) {
	repo := enabledGitLab(t)
	gittest.Run(t, repo, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "first")
	gittest.Run(t, repo, "update-ref", "refs/remotes/origin/main", "HEAD")
	gittest.Run(t, repo, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	for _, name := range []string{"feature", "kept"} {
		gittest.Run(t, repo, "switch", "-q", "-c", name, "main")
		writeAndCommit(t, repo, name+".txt", name, name)
		gittest.Run(t, repo, "update-ref", "refs/remotes/origin/"+name, name)
		gittest.Run(t, repo, "branch", "-q", "-u", "origin/"+name, name)
	}
	gittest.Run(t, repo, "switch", "-q", "main")
	featureTip := gittest.Output(t, repo, "rev-parse", "feature")
	api := "/api/v4/projects/group%2Fapp"
	transport := &routeTransport{routes: map[string]string{
		api + "/merge_requests/7": `{"iid": 7, "state": "merged", "source_branch": "feature", "target_branch": "main", "sha": "` + featureTip + `", "web_url": "https://gitlab.example.com/group/app/-/merge_requests/7"}`,
		api + "/merge_requests/8": `{"iid": 8, "state": "opened", "source_branch": "kept", "target_branch": "main", "sha": "x"}`,
	}, hits: map[string]int{}}
	swapHostClient(t, transport)
	_, info, _, err := primaryHost(repo)
	if err != nil {
		t.Fatal(err)
	}
	cacheRemoteRefs.put(info.Hostname, repo, "origin", git.RemoteChanges{Kind: git.MergeRequestRefs, ByBranch: map[string]int{"kept": 8}})
	lastOpen := []hostapi.Change{{Number: 7, State: hostapi.ChangeOpen, SourceBranch: "feature"}}

	refreshMergedChanges(context.Background(), providerFor(info), info, repo, "origin", lastOpen, nil)
	refreshMergedChanges(context.Background(), providerFor(info), info, repo, "origin", lastOpen, nil)
	if transport.hits[api+"/merge_requests/7"] != 1 || transport.hits[api+"/merge_requests/8"] != 2 {
		t.Fatalf("a merged change is asked for once, an open one each time: %v", transport.hits)
	}

	var s BranchService
	report, err := s.MergedBranches(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Branches) != 1 || report.Branches[0] != (MergedBranchInfo{Name: "feature", Upstream: "origin/feature", Tip: featureTip, Number: 7, Prefix: "!"}) || report.Default != "main" {
		t.Fatalf("report = %+v", report)
	}
	if transport.hits[api+"/merge_requests/7"] != 1 {
		t.Fatal("listing made a request")
	}

	gittest.Run(t, repo, "switch", "-q", "feature")
	gittest.Run(t, repo, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", "after the merge")
	gittest.Run(t, repo, "switch", "-q", "main")
	report, err = s.MergedBranches(repo)
	if err != nil || len(report.Branches) != 1 || !report.Branches[0].Newer {
		t.Fatalf("after a commit past the merged head, report = %+v, %v", report, err)
	}
}
