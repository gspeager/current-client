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

type StashOptions struct {
	Message          string   `json:"message"`
	IncludeUntracked bool     `json:"includeUntracked"`
	KeepIndex        bool     `json:"keepIndex"`
	Paths            []string `json:"paths"`
}

func (s *StashService) StashSave(repoPath string, opts StashOptions) error {
	return git.StashPush(context.Background(), repoPath, git.StashOptions(opts))
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

func (s *StashService) GetChangedFiles(repoPath string, index int) ([]ChangedFile, error) {
	ctx := context.Background()
	files, err := git.StashChangedFiles(ctx, repoPath, index)
	if err != nil {
		return nil, err
	}
	stats, err := git.StashNumstat(ctx, repoPath, index)
	if err != nil {
		return nil, err
	}
	return withLineCounts(files, stats), nil
}
