package app

import (
	"context"

	"github.com/gspeager/current-client/core/git"
)

type WorktreeService struct{}

type WorktreeInfo struct {
	Path    string `json:"path"`
	Branch  string `json:"branch"`
	Head    string `json:"head"`
	Main    bool   `json:"main"`
	Current bool   `json:"current"`
	Locked  bool   `json:"locked"`
	Missing bool   `json:"missing"`
}

func (s *WorktreeService) List(repoPath string) ([]WorktreeInfo, error) {
	worktrees, err := git.ListWorktrees(context.Background(), repoPath)
	if err != nil {
		return nil, err
	}
	return mapSlice(worktrees, func(w git.Worktree) WorktreeInfo { return WorktreeInfo(w) }), nil
}

// Add checks out branch in a new worktree at path, creating the branch from
// HEAD first when create is set.
func (s *WorktreeService) Add(repoPath, path, branch string, create bool) error {
	return git.AddWorktree(context.Background(), repoPath, path, branch, create, "")
}

func (s *WorktreeService) Remove(repoPath, path string, force bool) error {
	return git.RemoveWorktree(context.Background(), repoPath, path, force)
}
