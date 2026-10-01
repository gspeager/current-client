package app

import (
	"context"

	"github.com/gspeager/current-client/core/git"
)

type ContributorsService struct{}

type ContributorStatInfo struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Commits int    `json:"commits"`
}

// GetContributors takes since/until in any format git's --since accepts;
// empty means unbounded.
func (s *ContributorsService) GetContributors(repoPath, since, until string) ([]ContributorStatInfo, error) {
	stats, err := git.Contributors(context.Background(), repoPath, since, until)
	if err != nil {
		return nil, err
	}
	return mapSlice(stats, func(c git.ContributorStat) ContributorStatInfo { return ContributorStatInfo(c) }), nil
}
