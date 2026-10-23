package git

import (
	"context"
	"slices"
)

type GitFlowKind string

const (
	GitFlowFeature GitFlowKind = "feature"
	GitFlowRelease GitFlowKind = "release"
	GitFlowHotfix  GitFlowKind = "hotfix"
)

// StartGitFlowBranch branches hotfixes from main/master and everything else
// from develop when it exists.
func StartGitFlowBranch(ctx context.Context, repoPath string, kind GitFlowKind, name string) (branchName, base string, err error) {
	branches, err := ListBranches(ctx, repoPath)
	if err != nil {
		return "", "", err
	}
	base = gitFlowBase(kind, branches)
	branchName = string(kind) + "/" + name
	if err := CreateBranchAt(ctx, repoPath, branchName, base); err != nil {
		return "", "", err
	}
	if err := CheckoutBranch(ctx, repoPath, branchName); err != nil {
		return "", "", err
	}
	return branchName, base, nil
}

func gitFlowBase(kind GitFlowKind, branches []Branch) string {
	mainLike := defaultBranch(branches, "")
	if mainLike == "" {
		mainLike = "main"
	}
	if kind == GitFlowHotfix {
		return mainLike
	}
	if hasBranch(branches, "develop") {
		return "develop"
	}
	return mainLike
}

func hasBranch(branches []Branch, name string) bool {
	return slices.ContainsFunc(branches, func(b Branch) bool { return b.Name == name })
}
