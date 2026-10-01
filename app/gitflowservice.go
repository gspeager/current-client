package app

import (
	"context"

	"github.com/gspeager/current-client/core/git"
)

type GitFlowService struct{}

type GitFlowStartResult struct {
	BranchName string `json:"branchName"`
	Base       string `json:"base"`
}

// StartBranch's kind is "feature", "release", or "hotfix".
func (s *GitFlowService) StartBranch(repoPath, kind, name string) (GitFlowStartResult, error) {
	branchName, base, err := git.StartGitFlowBranch(context.Background(), repoPath, git.GitFlowKind(kind), name)
	if err != nil {
		return GitFlowStartResult{}, err
	}
	return GitFlowStartResult{BranchName: branchName, Base: base}, nil
}
