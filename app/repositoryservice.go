package app

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
	"github.com/gspeager/current-client/core/repository"
	"github.com/gspeager/current-client/internal/config"
)

type RepositoryService struct{}

// opened adds a newly opened, initialized or cloned repository to the recent list.
func opened(repo *repository.Repository, err error) (string, error) {
	if err != nil {
		return "", err
	}
	_ = updateConfig(func(cfg *config.Config) {
		cfg.RecentRepositories = repository.AddRecent(cfg.RecentRepositories, repo.Path)
	})
	return repo.Path, nil
}

func (s *RepositoryService) PickRepositoryFolder() (string, error) {
	return pickFolder("Open Repository")
}

func (s *RepositoryService) PickDestinationFolder() (string, error) {
	return pickFolder("Choose Destination Folder")
}

func (s *RepositoryService) OpenRepository(path string) (string, error) {
	return opened(repository.Open(context.Background(), path))
}

func (s *RepositoryService) InitRepository(path string) (string, error) {
	return opened(repository.Init(context.Background(), path))
}

// CloneRepository clones into a new folder named folderName inside parentDir.
func (s *RepositoryService) CloneRepository(ctx context.Context, url, parentDir, folderName string) (string, error) {
	if folderName == "" || folderName == "." || folderName == ".." || strings.ContainsAny(folderName, `/\`) {
		return "", &gitexec.AppError{Message: "Folder name can't be empty or contain slashes.", Detail: folderName}
	}
	return opened(repository.Clone(ctx, url, filepath.Join(parentDir, folderName)))
}

func (s *RepositoryService) GetRecentRepositories() ([]string, error) {
	cfg, _, err := loadCurrentConfig()
	if err != nil {
		return nil, err
	}
	return cfg.RecentRepositories, nil
}

type OpenTabsInfo struct {
	Tabs    []string `json:"tabs"`
	Focused string   `json:"focused"`
}

// RemoveRecentRepository only forgets the path; nothing on disk is touched.
func (s *RepositoryService) RemoveRecentRepository(path string) error {
	return updateConfig(func(cfg *config.Config) {
		cfg.RecentRepositories = repository.RemoveRecent(cfg.RecentRepositories, path)
	})
}

func (s *RepositoryService) GetOpenTabs() (OpenTabsInfo, error) {
	cfg, _, err := loadCurrentConfig()
	if err != nil {
		return OpenTabsInfo{}, err
	}
	return OpenTabsInfo{Tabs: cfg.OpenTabs, Focused: cfg.FocusedTab}, nil
}

func (s *RepositoryService) SetOpenTabs(tabs []string, focused string) error {
	return updateConfig(func(cfg *config.Config) {
		cfg.OpenTabs = tabs
		cfg.FocusedTab = focused
	})
}

const sparklineDays = 14

type RepoSummaryInfo struct {
	Path          string `json:"path"`
	Name          string `json:"name"`
	Available     bool   `json:"available"`
	Dirty         int    `json:"dirty"`
	CurrentBranch string `json:"currentBranch"`
	Ahead         int    `json:"ahead"`
	Behind        int    `json:"behind"`
	Activity      []int  `json:"activity"`
}

// GetRepoSummaries marks a moved or deleted repo as unavailable instead of
// failing the whole batch.
func (s *RepositoryService) GetRepoSummaries(paths []string) []RepoSummaryInfo {
	return mapSlice(paths, repoSummary)
}

func repoSummary(repoPath string) RepoSummaryInfo {
	ctx := context.Background()
	summary := RepoSummaryInfo{Path: repoPath, Name: filepath.Base(repoPath)}

	statuses, err := git.GetStatus(ctx, repoPath)
	if err != nil {
		return summary
	}
	summary.Dirty = len(statuses)

	branchStatus, err := git.CurrentBranchStatus(ctx, repoPath)
	if err != nil {
		return summary
	}
	summary.CurrentBranch = branchStatus.Current
	summary.Ahead = branchStatus.Ahead
	summary.Behind = branchStatus.Behind

	activity, err := git.CommitActivity(ctx, repoPath, sparklineDays)
	if err != nil {
		return summary
	}
	summary.Activity = mapSlice(activity, func(d git.DayActivity) int { return d.Commits })

	summary.Available = true
	return summary
}
