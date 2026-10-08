package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type NumstatEntry struct {
	Path     string
	OrigPath string
	Added    int
	Removed  int
	Binary   bool
}

func WorkingTreeNumstat(ctx context.Context, repoPath string) ([]NumstatEntry, error) {
	result, err := runResult(ctx, repoPath, "diff", "-M", "--numstat", "-z")
	if err != nil {
		return nil, err
	}
	return parseNumstat(result.Stdout)
}

func IndexNumstat(ctx context.Context, repoPath string) ([]NumstatEntry, error) {
	result, err := runResult(ctx, repoPath, "diff", "--cached", "-M", "--numstat", "-z")
	if err != nil {
		return nil, err
	}
	return parseNumstat(result.Stdout)
}

// CommitNumstat passes -M so a renamed-and-edited file counts as a small edit
// rather than a full delete plus a full add.
func CommitNumstat(ctx context.Context, repoPath, sha string) ([]NumstatEntry, error) {
	trees, err := commitTrees(ctx, repoPath, sha)
	if err != nil {
		return nil, err
	}
	result, err := runResult(ctx, repoPath, append([]string{"diff-tree", "--no-commit-id", "-M", "--numstat", "-r", "-z", "--root"}, trees...)...)
	if err != nil {
		return nil, err
	}
	return parseNumstat(result.Stdout)
}

func RefRangeNumstat(ctx context.Context, repoPath, fromRef, toRef string) ([]NumstatEntry, error) {
	result, err := runResult(ctx, repoPath, "diff", "-M", "--numstat", "-z", fromRef, toRef)
	if err != nil {
		return nil, err
	}
	return parseNumstat(result.Stdout)
}

// parseNumstat handles -z renames, which leave the path field empty and put
// the old and new paths in the next two NUL-separated fields.
func parseNumstat(output string) ([]NumstatEntry, error) {
	fields := strings.Split(output, "\x00")
	var entries []NumstatEntry
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		if field == "" {
			continue
		}
		parts := strings.SplitN(field, "\t", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("malformed numstat entry: %q", field)
		}
		entry := NumstatEntry{Binary: parts[0] == "-" || parts[1] == "-"}
		if !entry.Binary {
			added, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, fmt.Errorf("malformed numstat added count %q: %w", parts[0], err)
			}
			removed, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("malformed numstat removed count %q: %w", parts[1], err)
			}
			entry.Added, entry.Removed = added, removed
		}
		if parts[2] != "" {
			entry.Path = parts[2]
			entries = append(entries, entry)
			continue
		}
		i++
		if i >= len(fields) {
			return nil, fmt.Errorf("malformed numstat rename entry: missing old path")
		}
		entry.OrigPath = fields[i]
		i++
		if i >= len(fields) {
			return nil, fmt.Errorf("malformed numstat rename entry: missing new path")
		}
		entry.Path = fields[i]
		entries = append(entries, entry)
	}
	return entries, nil
}
