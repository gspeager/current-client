package app

import (
	"context"

	"github.com/gspeager/current-client/core/git"
)

type DashboardService struct{}

type RepoStatsInfo struct {
	TotalCommits     int    `json:"totalCommits"`
	ContributorCount int    `json:"contributorCount"`
	FileCount        int    `json:"fileCount"`
	RepoAgeDays      int    `json:"repoAgeDays"`
	BranchCount      int    `json:"branchCount"`
	TagCount         int    `json:"tagCount"`
	StashCount       int    `json:"stashCount"`
	CurrentBranch    string `json:"currentBranch"`
}

func (s *DashboardService) GetRepoStats(repoPath string) (RepoStatsInfo, error) {
	stats, err := git.ComputeRepoStats(context.Background(), repoPath)
	return RepoStatsInfo(stats), err
}

const (
	fileChurnWindowDays = 90
	fileChurnLimit      = 10
	// The frontend narrows this to the languages it has colors for.
	languageBreakdownLimit = 25
)

type FileChurnInfo struct {
	Path    string `json:"path"`
	Changes int    `json:"changes"`
}

func (s *DashboardService) GetFileChurn(repoPath string) ([]FileChurnInfo, error) {
	churn, err := git.ComputeFileChurn(context.Background(), repoPath, fileChurnWindowDays, fileChurnLimit)
	if err != nil {
		return nil, err
	}
	return mapSlice(churn, func(c git.FileChurn) FileChurnInfo { return FileChurnInfo(c) }), nil
}

type LanguageStatInfo struct {
	Extension string `json:"extension"`
	Count     int    `json:"count"`
}

func (s *DashboardService) GetLanguageBreakdown(repoPath string) ([]LanguageStatInfo, error) {
	stats, err := git.LanguageBreakdown(context.Background(), repoPath, languageBreakdownLimit)
	if err != nil {
		return nil, err
	}
	return mapSlice(stats, func(l git.LanguageStat) LanguageStatInfo { return LanguageStatInfo(l) }), nil
}

type RepoDiskUsageInfo struct {
	LooseObjectCount int `json:"looseObjectCount"`
	LooseSizeKB      int `json:"looseSizeKb"`
	PackCount        int `json:"packCount"`
	PackedSizeKB     int `json:"packedSizeKb"`
	TotalSizeKB      int `json:"totalSizeKb"`
}

func (s *DashboardService) GetRepoDiskUsage(repoPath string) (RepoDiskUsageInfo, error) {
	usage, err := git.ComputeRepoDiskUsage(context.Background(), repoPath)
	return RepoDiskUsageInfo(usage), err
}

// RunFsckCheck returns the problems found; empty means healthy.
func (s *DashboardService) RunFsckCheck(repoPath string) ([]string, error) {
	return git.FsckCheck(context.Background(), repoPath)
}

func (s *DashboardService) RunGC(repoPath string) error {
	return git.RunGC(context.Background(), repoPath)
}
