package git

import (
	"context"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestGitFlowBaseFeatureUsesDevelopWhenPresent(t *testing.T) {
	branches := []Branch{{Name: "main"}, {Name: "develop"}}
	if got := gitFlowBase(GitFlowFeature, branches); got != "develop" {
		t.Fatalf("gitFlowBase(feature) = %q, want %q", got, "develop")
	}
}

func TestGitFlowBaseFeatureFallsBackToMainWithoutDevelop(t *testing.T) {
	branches := []Branch{{Name: "main"}}
	if got := gitFlowBase(GitFlowFeature, branches); got != "main" {
		t.Fatalf("gitFlowBase(feature) = %q, want %q", got, "main")
	}
}

func TestGitFlowBaseFeatureFallsBackToMasterWhenNoMain(t *testing.T) {
	branches := []Branch{{Name: "master"}}
	if got := gitFlowBase(GitFlowFeature, branches); got != "master" {
		t.Fatalf("gitFlowBase(feature) = %q, want %q", got, "master")
	}
}

func TestGitFlowBaseHotfixIgnoresDevelop(t *testing.T) {
	branches := []Branch{{Name: "main"}, {Name: "develop"}}
	if got := gitFlowBase(GitFlowHotfix, branches); got != "main" {
		t.Fatalf("gitFlowBase(hotfix) = %q, want %q", got, "main")
	}
}

func TestGitFlowBaseReleaseUsesDevelopWhenPresent(t *testing.T) {
	branches := []Branch{{Name: "main"}, {Name: "develop"}}
	if got := gitFlowBase(GitFlowRelease, branches); got != "develop" {
		t.Fatalf("gitFlowBase(release) = %q, want %q", got, "develop")
	}
}

func TestStartGitFlowBranchCreatesAndChecksOutFromDevelop(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	gittest.Run(t, dir, "checkout", "-q", "-b", "develop")
	gittest.CommitFile(t, dir, "develop.txt", "v1", "on develop")
	gittest.Run(t, dir, "checkout", "-q", "main")

	branchName, base, err := StartGitFlowBranch(context.Background(), dir, GitFlowFeature, "my-feature")
	if err != nil {
		t.Fatalf("StartGitFlowBranch: %v", err)
	}
	if branchName != "feature/my-feature" {
		t.Fatalf("branchName = %q, want %q", branchName, "feature/my-feature")
	}
	if base != "develop" {
		t.Fatalf("base = %q, want %q", base, "develop")
	}

	branches, err := ListBranches(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	for _, b := range branches {
		if b.Name == "feature/my-feature" && !b.Current {
			t.Fatalf("branches = %+v, want feature/my-feature checked out", branches)
		}
	}

	// develop.txt only exists if the branch started from develop.
	gittest.Run(t, dir, "cat-file", "-e", "feature/my-feature:develop.txt")
}

func TestStartGitFlowBranchHotfixFromMain(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	gittest.Run(t, dir, "checkout", "-q", "-b", "develop")
	gittest.Run(t, dir, "checkout", "-q", "main")

	branchName, base, err := StartGitFlowBranch(context.Background(), dir, GitFlowHotfix, "urgent-fix")
	if err != nil {
		t.Fatalf("StartGitFlowBranch: %v", err)
	}
	if branchName != "hotfix/urgent-fix" {
		t.Fatalf("branchName = %q, want %q", branchName, "hotfix/urgent-fix")
	}
	if base != "main" {
		t.Fatalf("base = %q, want %q", base, "main")
	}
}
