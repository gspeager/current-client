package app

import (
	"context"

	"github.com/gspeager/current-client/core/git"
)

type ActivityService struct{}

type DayActivityInfo struct {
	Date    string `json:"date"`
	Commits int    `json:"commits"`
}

func (s *ActivityService) GetCommitActivity(repoPath string, days int) ([]DayActivityInfo, error) {
	activity, err := git.CommitActivity(context.Background(), repoPath, days)
	if err != nil {
		return nil, err
	}
	return mapSlice(activity, func(d git.DayActivity) DayActivityInfo { return DayActivityInfo(d) }), nil
}
