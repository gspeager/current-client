package app

import (
	"context"

	"github.com/gspeager/current-client/core/git"
)

type CommitService struct{}

func (s *CommitService) Commit(repoPath, message string, amend, allowEmpty, sign bool) error {
	return git.Commit(context.Background(), repoPath, message, git.CommitOptions{Amend: amend, AllowEmpty: allowEmpty, Sign: sign})
}

func (s *CommitService) GetLastCommitMessage(repoPath string) (string, error) {
	return git.LastCommitMessage(context.Background(), repoPath)
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
