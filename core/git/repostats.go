package git

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type RepoStats struct {
	TotalCommits     int
	ContributorCount int
	FileCount        int
	RepoAgeDays      int
	BranchCount      int
	TagCount         int
	StashCount       int
	CurrentBranch    string
}

func ComputeRepoStats(ctx context.Context, repoPath string) (RepoStats, error) {
	var stats RepoStats

	// Stash entries are commits too; --exclude must come before --all to apply.
	commits, err := runResult(ctx, repoPath, "rev-list", "--count", "--exclude=refs/stash", "--all")
	if err != nil {
		return stats, err
	}
	stats.TotalCommits, _ = strconv.Atoi(strings.TrimSpace(commits.Stdout))

	contributors, err := Contributors(ctx, repoPath, "", "")
	if err != nil {
		return stats, err
	}
	stats.ContributorCount = len(contributors)

	files, err := trackedFiles(ctx, repoPath)
	if err != nil {
		return stats, err
	}
	stats.FileCount = len(files)

	firstCommit, err := runResult(ctx, repoPath, "log", "--all", "--format=%at", "--reverse", "-1")
	if err != nil {
		return stats, err
	}
	if unixTime := strings.TrimSpace(firstCommit.Stdout); unixTime != "" {
		if epoch, convErr := strconv.ParseInt(unixTime, 10, 64); convErr == nil {
			stats.RepoAgeDays = int(time.Since(time.Unix(epoch, 0)).Hours() / 24)
		}
	}

	branches, err := ListBranches(ctx, repoPath)
	if err != nil {
		return stats, err
	}
	stats.BranchCount = len(branches)

	tags, err := ListTags(ctx, repoPath)
	if err != nil {
		return stats, err
	}
	stats.TagCount = len(tags)

	stashes, err := ListStashes(ctx, repoPath)
	if err != nil {
		return stats, err
	}
	stats.StashCount = len(stashes)

	status, err := CurrentBranchStatus(ctx, repoPath)
	if err != nil {
		return stats, err
	}
	stats.CurrentBranch = status.Current

	return stats, nil
}

// trackedFiles is cached until the index changes, as Activity's stats and
// language breakdown both list them.
func trackedFiles(ctx context.Context, repoPath string) ([]string, error) {
	gitDir, err := GitDir(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	return cachedByRefs(ctx, repoPath, "tracked-files", func() ([]string, error) {
		result, err := runResult(ctx, repoPath, "ls-files")
		if err != nil {
			return nil, err
		}
		var files []string
		for _, line := range strings.Split(result.Stdout, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				files = append(files, line)
			}
		}
		return files, nil
	}, filepath.Join(gitDir, "index"))
}
