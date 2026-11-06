package app

import (
	"context"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/risk"
)

type CommitService struct{}

type RiskFindingInfo struct {
	Rule    string `json:"rule"`
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Message string `json:"message"`
}

// GetRiskReport warns about what's staged. Nothing it finds blocks a commit.
func (s *CommitService) GetRiskReport(repoPath string) ([]RiskFindingInfo, error) {
	findings, err := risk.Check(context.Background(), repoPath, configOrDefault().RiskRulesOff)
	if err != nil {
		return nil, err
	}
	report := []RiskFindingInfo{}
	for _, finding := range findings {
		report = append(report, RiskFindingInfo(finding))
	}
	return report, nil
}

func (s *CommitService) Commit(repoPath, message string, amend, allowEmpty, sign bool) error {
	return git.Commit(context.Background(), repoPath, message, git.CommitOptions{Amend: amend, AllowEmpty: allowEmpty, Sign: sign})
}

func (s *CommitService) GetLastCommitMessage(repoPath string) (string, error) {
	return git.LastCommitMessage(context.Background(), repoPath)
}

// GetSquashedSubjects lists the commits a pending squash merge staged, newest
// first, or nothing when no squash is pending.
func (s *CommitService) GetSquashedSubjects(repoPath string) ([]string, error) {
	return git.SquashedSubjects(context.Background(), repoPath)
}

func (s *CommitService) GetGPGSignDefault(repoPath string) bool {
	return git.GPGSignDefault(context.Background(), repoPath)
}

func (s *CommitService) UndoLastCommit(repoPath string) error {
	return git.UndoLastCommit(context.Background(), repoPath)
}

func (s *CommitService) IsLastCommitPushed(repoPath string) (bool, error) {
	return git.IsLastCommitPushed(context.Background(), repoPath)
}

func (s *CommitService) CherryPick(repoPath, sha string) error {
	return git.CherryPick(context.Background(), repoPath, sha)
}

func (s *CommitService) Revert(repoPath, sha string) error {
	return git.Revert(context.Background(), repoPath, sha)
}

// Reset's mode is "soft", "mixed", or "hard".
func (s *CommitService) Reset(repoPath, ref, mode string) error {
	if mode == "hard" {
		snapshotBefore(repoPath, "reset")
	}
	return git.Reset(context.Background(), repoPath, ref, git.ResetMode(mode))
}

type AuthorInfo struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (s *CommitService) ListRecentAuthors(repoPath string) ([]AuthorInfo, error) {
	authors, err := git.RecentAuthors(context.Background(), repoPath)
	if err != nil {
		return nil, err
	}
	return mapSlice(authors, func(a git.Author) AuthorInfo { return AuthorInfo(a) }), nil
}
