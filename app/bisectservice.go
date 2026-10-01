package app

import (
	"context"

	"github.com/gspeager/current-client/core/git"
)

type BisectService struct{}

type BisectStatusInfo struct {
	Active         bool   `json:"active"`
	Done           bool   `json:"done"`
	FoundSHA       string `json:"foundSha"`
	CurrentSHA     string `json:"currentSha"`
	RevisionsLeft  int    `json:"revisionsLeft"`
	StepsRemaining int    `json:"stepsRemaining"`
}

func (s *BisectService) Start(repoPath, badRev, goodRev string) (BisectStatusInfo, error) {
	status, err := git.BisectStart(context.Background(), repoPath, badRev, goodRev)
	return BisectStatusInfo(status), err
}

// Mark's verdict is "good", "bad", or "skip".
func (s *BisectService) Mark(repoPath, verdict string) (BisectStatusInfo, error) {
	status, err := git.BisectMark(context.Background(), repoPath, verdict)
	return BisectStatusInfo(status), err
}

func (s *BisectService) Reset(repoPath string) error {
	return git.BisectReset(context.Background(), repoPath)
}

func (s *BisectService) IsActive(repoPath string) (bool, error) {
	return git.BisectActive(context.Background(), repoPath)
}
