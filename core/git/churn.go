package git

import (
	"context"
	"sort"
	"strings"
	"time"
)

type FileChurn struct {
	Path    string
	Changes int
}

// ComputeFileChurn returns the most-changed files, most-changed first. git
// log omits merge commits' file lists by default, which keeps a merged
// branch's files from being double-counted.
func ComputeFileChurn(ctx context.Context, repoPath string, days, limit int) ([]FileChurn, error) {
	since := sinceMidnight(time.Now().AddDate(0, 0, -days))
	result, err := runResult(ctx, repoPath, "log", "--all", "--since="+since, "--format=", "--name-only")
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	var order []string
	for _, line := range strings.Split(result.Stdout, "\n") {
		path := strings.TrimSpace(line)
		if path == "" {
			continue
		}
		if counts[path] == 0 {
			order = append(order, path)
		}
		counts[path]++
	}

	churn := make([]FileChurn, len(order))
	for i, path := range order {
		churn[i] = FileChurn{Path: path, Changes: counts[path]}
	}
	sort.SliceStable(churn, func(i, j int) bool { return churn[i].Changes > churn[j].Changes })

	if len(churn) > limit {
		churn = churn[:limit]
	}
	return churn, nil
}
