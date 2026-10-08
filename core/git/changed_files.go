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

// ChangedFiles lists a merge commit's changes against its first parent: what
// the merge brought in, as Revert and Cherry-pick apply it.
func ChangedFiles(ctx context.Context, repoPath, sha string) ([]ChangedFile, error) {
	trees, err := commitTrees(ctx, repoPath, sha)
	if err != nil {
		return nil, err
	}
	result, err := runResult(ctx, repoPath, append([]string{"diff-tree", "--no-commit-id", "--name-status", "-r", "-z", "--root"}, trees...)...)
	if err != nil {
		return nil, err
	}
	return parseChangedFiles(result.Stdout)
}

// commitTrees is what diff-tree compares for a commit: the commit alone, which
// diffs it against its parent, or a merge's first parent and the merge.
func commitTrees(ctx context.Context, repoPath, sha string) ([]string, error) {
	parents, err := runResult(ctx, repoPath, "rev-list", "--parents", "-n", "1", sha)
	if err != nil {
		return nil, err
	}
	if len(strings.Fields(parents.Stdout)) > 2 {
		return []string{sha + "^1", sha}, nil
	}
	return []string{sha}, nil
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
