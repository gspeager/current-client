package git

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/gspeager/current-client/core/gitexec"
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
	var err error
	if stats.TotalCommits, err = CountCommits(ctx, repoPath, allRefs...); err != nil {
		return stats, err
	}

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

	// The root commits; a repository that merged in another history has several.
	roots, err := gitexec.RunIn(ctx, repoPath, append([]string{"log", "--max-parents=0", "--format=%at"}, allRefs...)...)
	if err != nil {
		return stats, err
	}
	if oldest, ok := oldestTime(roots.Stdout); ok {
		stats.RepoAgeDays = int(time.Since(oldest).Hours() / 24)
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

func oldestTime(unixTimes string) (time.Time, bool) {
	var oldest int64
	found := false
	for _, field := range strings.Fields(unixTimes) {
		if t, err := strconv.ParseInt(field, 10, 64); err == nil && (!found || t < oldest) {
			oldest, found = t, true
		}
	}
	return time.Unix(oldest, 0), found
}
