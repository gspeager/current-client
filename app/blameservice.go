package app

import (
	"context"
	"time"

	"github.com/gspeager/current-client/core/git"
)

type BlameService struct{}

type BlameLineInfo struct {
	SHA         string `json:"sha"`
	AuthorName  string `json:"authorName"`
	AuthorEmail string `json:"authorEmail"`
	Date        string `json:"date"`
	Summary     string `json:"summary"`
	LineNo      int    `json:"lineNo"`
	Content     string `json:"content"`
	// Previous and PreviousPath are what "Blame before this commit" opens; empty
	// for the commit that added the file.
	Previous     string `json:"previous"`
	PreviousPath string `json:"previousPath"`
}

func (s *BlameService) GetBlame(repoPath, path string) ([]BlameLineInfo, error) {
	return s.GetBlameAt(repoPath, path, "")
}

// GetBlameAt blames path as of rev, or the working tree when rev is empty.
func (s *BlameService) GetBlameAt(repoPath, path, rev string) ([]BlameLineInfo, error) {
	lines, err := git.BlameAt(context.Background(), repoPath, path, rev)
	if err != nil {
		return nil, err
	}
	return mapSlice(lines, func(l git.BlameLine) BlameLineInfo {
		return BlameLineInfo{
			SHA:          l.SHA,
			AuthorName:   l.AuthorName,
			AuthorEmail:  l.AuthorEmail,
			Date:         l.Date.Format(time.RFC3339),
			Summary:      l.Summary,
			LineNo:       l.LineNo,
			Content:      l.Content,
			Previous:     l.Previous,
			PreviousPath: l.PreviousPath,
		}
	}), nil
}
