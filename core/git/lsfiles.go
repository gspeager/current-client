package git

import (
	"context"
	"strings"
)

// WatchedPaths returns tracked and untracked-but-not-ignored files, so a
// watcher never descends into ignored directories like node_modules.
func WatchedPaths(ctx context.Context, repoPath string) ([]string, error) {
	tracked, err := runResult(ctx, repoPath, "ls-files", "-z")
	if err != nil {
		return nil, err
	}
	untracked, err := runResult(ctx, repoPath, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, output := range []string{tracked.Stdout, untracked.Stdout} {
		for _, p := range strings.Split(output, "\x00") {
			if p != "" {
				paths = append(paths, p)
			}
		}
	}
	return paths, nil
}
