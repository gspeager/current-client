package git

import (
	"context"
	"strings"
)

// WatchedPaths returns tracked and untracked-but-not-ignored files, so a
// watcher never descends into ignored directories like node_modules.
func WatchedPaths(ctx context.Context, repoPath string) ([]string, error) {
	// The exclude patterns only apply to --others: a tracked file is listed even if ignored.
	result, err := runResult(ctx, repoPath, "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, p := range strings.Split(result.Stdout, "\x00") {
		if p != "" {
			paths = append(paths, p)
		}
	}
	return paths, nil
}
