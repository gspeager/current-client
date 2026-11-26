package git

import (
	"context"
	"time"

	"github.com/gspeager/current-client/core/gitexec"
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
	// -z, so a name Git would quote comes as it is.
	result, err := gitexec.RunIn(ctx, repoPath, append([]string{"log", "--since=" + since, "--format=", "--name-only", "-z"}, allRefs...)...)
	if err != nil {
		return nil, err
	}
	churn := []FileChurn{}
	for _, r := range rank(nulFields(result.Stdout)) {
		churn = append(churn, FileChurn{Path: r.key, Changes: r.count})
	}

	if len(churn) > limit {
		churn = churn[:limit]
	}
	return churn, nil
}
