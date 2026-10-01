package git

import (
	"context"
	"fmt"
	"strings"
)

type ChangedFile struct {
	Status   string
	Path     string
	OrigPath string
}

// ChangedFiles returns nothing for a merge commit, matching `git show`.
func ChangedFiles(ctx context.Context, repoPath, sha string) ([]ChangedFile, error) {
	result, err := runResult(ctx, repoPath, "diff-tree", "--no-commit-id", "--name-status", "-r", "-z", "--root", sha)
	if err != nil {
		return nil, err
	}
	return parseChangedFiles(result.Stdout)
}

// RefRangeChangedFiles compares the two tips directly (not against their
// merge base), matching diff.GetRefDiff so the list and its diffs agree.
func RefRangeChangedFiles(ctx context.Context, repoPath, fromRef, toRef string) ([]ChangedFile, error) {
	result, err := runResult(ctx, repoPath, "diff", "--name-status", "-z", fromRef, toRef)
	if err != nil {
		return nil, err
	}
	return parseChangedFiles(result.Stdout)
}

func parseChangedFiles(output string) ([]ChangedFile, error) {
	fields := strings.Split(output, "\x00")
	var files []ChangedFile
	for i := 0; i < len(fields); i++ {
		status := fields[i]
		if status == "" {
			continue
		}
		i++
		if i >= len(fields) {
			return nil, fmt.Errorf("malformed diff-tree entry: missing path for status %q", status)
		}
		path := fields[i]
		cf := ChangedFile{Status: status, Path: path}
		if status[0] == 'R' || status[0] == 'C' {
			i++
			if i >= len(fields) {
				return nil, fmt.Errorf("malformed diff-tree rename entry: missing new path")
			}
			cf.OrigPath = path
			cf.Path = fields[i]
		}
		files = append(files, cf)
	}
	return files, nil
}
