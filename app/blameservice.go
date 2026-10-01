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
}

func (s *BlameService) GetBlame(repoPath, path string) ([]BlameLineInfo, error) {
	lines, err := git.Blame(context.Background(), repoPath, path)
	if err != nil {
		return nil, err
	}
	return mapSlice(lines, func(l git.BlameLine) BlameLineInfo {
		return BlameLineInfo{
			SHA:         l.SHA,
			AuthorName:  l.AuthorName,
			AuthorEmail: l.AuthorEmail,
			Date:        l.Date.Format(time.RFC3339),
			Summary:     l.Summary,
			LineNo:      l.LineNo,
			Content:     l.Content,
		}
	}), nil
}
