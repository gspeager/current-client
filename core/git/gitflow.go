package git

import "context"

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
	has := func(name string) bool {
		for _, b := range branches {
			if b.Name == name {
				return true
			}
		}
		return false
	}
	mainLike := defaultBranch(branches, "")
	if mainLike == "" {
		mainLike = "main"
	}
	if kind == GitFlowHotfix {
		return mainLike
	}
	if has("develop") {
		return "develop"
	}
	return mainLike
}
