package git

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type DayActivity struct {
	Date    string // YYYY-MM-DD
	Commits int
}

// activityWindowDays covers Activity's calendar. Shorter windows, such as the
// velocity chart and the tab sparklines, are cut from the same read.
const activityWindowDays = 371

// CommitActivity returns one entry per day, oldest first, including days
// with zero commits.
func CommitActivity(ctx context.Context, repoPath string, days int) ([]DayActivity, error) {
	window := max(days, activityWindowDays)
	today := time.Now()
	counts, err := cachedByRefs(ctx, repoPath, fmt.Sprintf("activity\x00%s\x00%d", today.Format("2006-01-02"), window), func() (map[string]int, error) {
		since := sinceMidnight(today.AddDate(0, 0, -window+1))
		result, err := runResult(ctx, repoPath, "log", "--exclude=refs/stash", "--all", "--since="+since, "--format=%ad", "--date=short")
		if err != nil {
			return nil, err
		}
		counts := make(map[string]int)
		for _, line := range strings.Split(result.Stdout, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				counts[line]++
			}
		}
		return counts, nil
	})
	if err != nil {
		return nil, err
	}

	start := today.AddDate(0, 0, -days+1)
	activity := make([]DayActivity, days)
	for i := 0; i < days; i++ {
		date := start.AddDate(0, 0, i).Format("2006-01-02")
		activity[i] = DayActivity{Date: date, Commits: counts[date]}
	}
	return activity, nil
}
