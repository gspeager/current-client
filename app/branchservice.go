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
	WorktreePath   string `json:"worktreePath"`
	Tip            string `json:"tip"`
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

func (s *BranchService) SetUpstream(repoPath, branch, remoteBranch string) error {
	return git.SetUpstream(context.Background(), repoPath, branch, remoteBranch)
}

func (s *BranchService) UnsetUpstream(repoPath, branch string) error {
	return git.UnsetUpstream(context.Background(), repoPath, branch)
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

// MergedBranchInfo is a local branch whose work is already in Base, found
// with Git alone, or merged on the host as change Number when the cache
// knows that. Newer means the branch has commits past what was merged.
type MergedBranchInfo struct {
	Name     string `json:"name"`
	Current  bool   `json:"current"`
	Upstream string `json:"upstream"`
	Tip      string `json:"tip"`
	Newer    bool   `json:"newer"`
	Number   int    `json:"number"`
	Prefix   string `json:"prefix"`
}

type MergedBranches struct {
	Default  string             `json:"default"`
	Base     string             `json:"base"`
	Branches []MergedBranchInfo `json:"branches"`
}

// MergedBranches makes no request: the host's part comes from the cache.
func (s *BranchService) MergedBranches(repoPath string) (MergedBranches, error) {
	ctx := context.Background()
	report, err := git.MergedBranches(ctx, repoPath)
	if err != nil {
		return MergedBranches{}, err
	}
	def, err := git.DefaultBranch(ctx, repoPath)
	if err != nil {
		return MergedBranches{}, err
	}
	result := MergedBranches{Default: def, Base: report.Base}
	listed := map[string]int{}
	for _, b := range report.Branches {
		listed[b.Name] = len(result.Branches)
		result.Branches = append(result.Branches, MergedBranchInfo{
			Name: b.Name, Current: b.Current, Upstream: b.Upstream, Tip: b.Tip, Newer: b.Newer,
		})
	}
	onHost := mergedOnHost(repoPath)
	if len(onHost) == 0 {
		return result, nil
	}
	branches, err := git.ListBranches(ctx, repoPath)
	if err != nil {
		return MergedBranches{}, err
	}
	for _, b := range branches {
		change, ok := onHost[b.Upstream]
		if !ok || b.Name == def {
			continue
		}
		if i, ok := listed[b.Name]; ok {
			result.Branches[i].Number, result.Branches[i].Prefix = change.Number, change.Prefix
			continue
		}
		result.Branches = append(result.Branches, MergedBranchInfo{
			Name: b.Name, Current: b.Current, Upstream: b.Upstream, Tip: b.Tip,
			Newer: b.Tip != change.HeadSHA, Number: change.Number, Prefix: change.Prefix,
		})
	}
	return result, nil
}

// hostMergedChange is what the cache knows of a merged change, by its
// source branch's remote-tracking name.
type hostMergedChange struct {
	Number  int
	Prefix  string
	HeadSHA string
}
