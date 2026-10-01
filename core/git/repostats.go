package git

import (
	"context"
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

	files, err := runResult(ctx, repoPath, "ls-files")
	if err != nil {
		return stats, err
	}
	stats.FileCount = countNonEmptyLines(files.Stdout)

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

	stats.CurrentBranch, err = currentBranchName(ctx, repoPath)
	if err != nil {
		return stats, err
	}

	return stats, nil
}

func countNonEmptyLines(output string) int {
	count := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}
