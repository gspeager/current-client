package git

import (
	"context"
	"strings"
	"time"
)

type DayActivity struct {
	Date    string // YYYY-MM-DD
	Commits int
}

// CommitActivity returns one entry per day, oldest first, including days
// with zero commits.
func CommitActivity(ctx context.Context, repoPath string, days int) ([]DayActivity, error) {
	start := time.Now().AddDate(0, 0, -days+1)
	result, err := runResult(ctx, repoPath, "log", "--exclude=refs/stash", "--all", "--since="+sinceMidnight(start), "--format=%ad", "--date=short")
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int)
	for _, line := range strings.Split(result.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		counts[line]++
	}

	activity := make([]DayActivity, days)
	for i := 0; i < days; i++ {
		date := start.AddDate(0, 0, i).Format("2006-01-02")
		activity[i] = DayActivity{Date: date, Commits: counts[date]}
	}
	return activity, nil
}
