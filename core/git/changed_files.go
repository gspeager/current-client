package git

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
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
	parents, err := parentCount(ctx, repoPath, sha)
	if err != nil {
		return nil, err
	}
	if parents > 1 {
		return []string{sha + "^1", sha}, nil
	}
	return []string{sha}, nil
}

func parentCount(ctx context.Context, repoPath, rev string) (int, error) {
	result, err := runResult(ctx, repoPath, "rev-list", "--parents", "-n", "1", rev)
	if err != nil {
		return 0, err
	}
	return len(strings.Fields(result.Stdout)) - 1, nil
}

// FileVersion is a file in a tree; both are "" when it isn't there.
type FileVersion struct {
	Mode string
	Blob string
}

// TreeChange is a file that differs between two trees, with its content in each.
type TreeChange struct {
	ChangedFile
	Added   int
	Removed int
	Binary  bool
	Old     FileVersion
	New     FileVersion
}

// TreeChanges lists what differs from one tree or commit to another. env can
// name where their objects are.
func TreeChanges(ctx context.Context, repoPath string, env []string, from, to string) ([]TreeChange, error) {
	return treeChanges(ctx, repoPath, env, "-M", from, to)
}

// TreeChangesWithoutRenames is TreeChanges with each path compared with itself.
func TreeChangesWithoutRenames(ctx context.Context, repoPath string, env []string, from, to string) ([]TreeChange, error) {
	return treeChanges(ctx, repoPath, env, "--no-renames", from, to)
}

func treeChanges(ctx context.Context, repoPath string, env []string, renames, from, to string) ([]TreeChange, error) {
	result, err := gitexec.NewExecutor("").RunChecked(ctx, gitexec.Command{
		Dir:  repoPath,
		Env:  env,
		Args: []string{"diff", "--raw", "--numstat", "--no-abbrev", "--no-ext-diff", "--no-relative", renames, "-z", from, to},
	})
	if err != nil {
		return nil, err
	}
	return parseTreeChanges(result.Stdout)
}

// Every file's ":<mode> <mode> <blob> <blob> <status>\0<path>\0" comes first, with a
// second path for a rename, then every file's numstat.
func parseTreeChanges(output string) ([]TreeChange, error) {
	var changes []TreeChange
	for strings.HasPrefix(output, ":") {
		header, rest, _ := strings.Cut(output[1:], "\x00")
		fields := strings.Fields(header)
		if len(fields) != 5 {
			return nil, fmt.Errorf("malformed raw diff entry: %q", header)
		}
		change := TreeChange{Old: fileVersion(fields[0], fields[2]), New: fileVersion(fields[1], fields[3])}
		change.Status = fields[4]
		change.Path, rest, _ = strings.Cut(rest, "\x00")
		if change.Status[0] == 'R' || change.Status[0] == 'C' {
			change.OrigPath = change.Path
			change.Path, rest, _ = strings.Cut(rest, "\x00")
		}
		changes = append(changes, change)
		output = rest
	}
	stats, err := parseNumstat(output)
	if err != nil {
		return nil, err
	}
	byPath := make(map[string]NumstatEntry, len(stats))
	for _, entry := range stats {
		byPath[entry.Path] = entry
	}
	for i := range changes {
		entry := byPath[changes[i].Path]
		changes[i].Added, changes[i].Removed, changes[i].Binary = entry.Added, entry.Removed, entry.Binary
	}
	return changes, nil
}

func fileVersion(mode, blob string) FileVersion {
	if strings.Trim(mode, "0") == "" {
		return FileVersion{}
	}
	return FileVersion{Mode: mode, Blob: blob}
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
				return nil, errors.New("malformed diff-tree rename entry: missing new path")
			}
			cf.OrigPath = path
			cf.Path = fields[i]
		}
		files = append(files, cf)
	}
	return files, nil
}
