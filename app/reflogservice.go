package app

import (
	"context"
	"time"

	"github.com/gspeager/current-client/core/git"
)

type ReflogService struct{}

type ReflogEntryInfo struct {
	SHA      string `json:"sha"`
	Selector string `json:"selector"`
	Action   string `json:"action"`
	Date     string `json:"date"`
	Subject  string `json:"subject"`
}

func (s *ReflogService) GetReflog(repoPath string, limit int) ([]ReflogEntryInfo, error) {
	entries, err := git.Reflog(context.Background(), repoPath, limit)
	if err != nil {
		return nil, err
	}
	return mapSlice(entries, func(e git.ReflogEntry) ReflogEntryInfo {
		return ReflogEntryInfo{
			SHA:      e.SHA,
			Selector: e.Selector,
			Action:   e.Action,
			Date:     e.Date.Format(time.RFC3339),
			Subject:  e.Subject,
		}
	}), nil
}
