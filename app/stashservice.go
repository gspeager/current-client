package app

import (
	"context"
	"time"

	"github.com/gspeager/current-client/core/git"
)

type StashService struct{}

type StashInfo struct {
	Index   int    `json:"index"`
	Message string `json:"message"`
	Date    string `json:"date"`
}

func (s *StashService) ListStash(repoPath string) ([]StashInfo, error) {
	stashes, err := git.ListStashes(context.Background(), repoPath)
	if err != nil {
		return nil, err
	}
	return mapSlice(stashes, func(st git.Stash) StashInfo {
		return StashInfo{Index: st.Index, Message: st.Message, Date: st.Date.Format(time.RFC3339)}
	}), nil
}

func (s *StashService) StashSave(repoPath, message string, includeUntracked bool) error {
	return git.StashSave(context.Background(), repoPath, message, includeUntracked)
}

func (s *StashService) StashApply(repoPath string, index int) error {
	return git.StashApply(context.Background(), repoPath, index)
}

func (s *StashService) StashPop(repoPath string, index int) error {
	return git.StashPop(context.Background(), repoPath, index)
}

func (s *StashService) StashDrop(repoPath string, index int) error {
	return git.StashDrop(context.Background(), repoPath, index)
}
