package app

import (
	"context"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
)

type BranchService struct{}

type BranchInfo struct {
	Name           string `json:"name"`
	Current        bool   `json:"current"`
	Upstream       string `json:"upstream"`
	Ahead          int    `json:"ahead"`
	Behind         int    `json:"behind"`
	LastCommitDate string `json:"lastCommitDate"`
}

type BranchStatusInfo struct {
	Current    string `json:"current"`
	Upstream   string `json:"upstream"`
	Ahead      int    `json:"ahead"`
	Behind     int    `json:"behind"`
	DetachedAt string `json:"detachedAt"`
}

func (s *BranchService) ListLocal(repoPath string) ([]BranchInfo, error) {
	branches, err := git.ListBranches(context.Background(), repoPath)
	if err != nil {
		return nil, err
	}
	return mapSlice(branches, func(b git.Branch) BranchInfo { return BranchInfo(b) }), nil
}

func (s *BranchService) ListRemote(repoPath string) ([]string, error) {
	return git.ListRemote(context.Background(), repoPath)
}

func (s *BranchService) CreateBranch(repoPath, name string) error {
	return git.CreateBranch(context.Background(), repoPath, name)
}

func (s *BranchService) CheckoutCommit(repoPath, sha string) error {
	return git.CheckoutCommit(context.Background(), repoPath, sha)
}

func (s *BranchService) CheckoutBranch(repoPath, name string) error {
	return git.CheckoutBranch(context.Background(), repoPath, name)
}

func (s *BranchService) CreateBranchAt(repoPath, name, startPoint string) error {
	return git.CreateBranchAt(context.Background(), repoPath, name, startPoint)
}

func (s *BranchService) RenameBranch(repoPath, oldName, newName string) error {
	return git.RenameBranch(context.Background(), repoPath, oldName, newName)
}

// DeleteBranch returns the deleted branch's tip, so it can be recreated.
func (s *BranchService) DeleteBranch(repoPath, name string, force bool) (string, error) {
	ctx := context.Background()
	tip, err := git.ResolveCommit(ctx, repoPath, "refs/heads/"+name)
	if err != nil {
		return "", err
	}
	return tip, git.DeleteBranch(ctx, repoPath, name, force)
}

// MergeBranch takes mode "" (fast-forward when possible), "no-ff" or "squash".
func (s *BranchService) MergeBranch(repoPath, branch, mode string) error {
	m := git.MergeMode(mode)
	if m != git.MergeFastForward && m != git.MergeCommit && m != git.MergeSquash {
		return &gitexec.AppError{Message: "Unknown merge option.", Detail: mode}
	}
	return git.MergeBranchMode(context.Background(), repoPath, branch, m)
}

func (s *BranchService) DefaultBranch(repoPath string) (string, error) {
	return git.DefaultBranch(context.Background(), repoPath)
}

func (s *BranchService) RebaseOnto(repoPath, onto string) error {
	return git.RebaseOnto(context.Background(), repoPath, onto)
}

func (s *BranchService) CurrentBranchStatus(repoPath string) (BranchStatusInfo, error) {
	status, err := git.CurrentBranchStatus(context.Background(), repoPath)
	return BranchStatusInfo(status), err
}
