package git

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

// WatchedPaths returns tracked and untracked-but-not-ignored files, so a
// watcher never descends into ignored directories like node_modules.
func WatchedPaths(ctx context.Context, repoPath string) ([]string, error) {
	// The exclude patterns only apply to --others: a tracked file is listed even if ignored.
	result, err := gitexec.RunIn(ctx, repoPath, "ls-files", "-z", "--cached", "--others", "--exclude-standard")
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

// FindTracked lists up to limit tracked files whose path holds query, ignoring case.
func FindTracked(ctx context.Context, repoPath, query string, limit int) ([]string, error) {
	result, err := gitexec.RunIn(ctx, repoPath, "ls-files", "-z", "--cached")
	if err != nil {
		return nil, err
	}
	query = strings.ToLower(query)
	var found []string
	for p := range strings.SplitSeq(result.Stdout, "\x00") {
		if p != "" && strings.Contains(strings.ToLower(p), query) {
			if found = append(found, p); len(found) == limit {
				break
			}
		}
	}
	return found, nil
}

type TreeFile struct {
	Path string
	Blob string
}

// TreeFiles leaves out symbolic links and submodules.
func TreeFiles(ctx context.Context, repoPath string, env []string, rev string) ([]TreeFile, error) {
	result, err := gitexec.RunChecked(ctx, gitexec.Command{
		Dir:  repoPath,
		Env:  env,
		Args: []string{"ls-tree", "-r", "-z", "--full-tree", rev},
	})
	if err != nil {
		return nil, err
	}
	var files []TreeFile
	for _, entry := range ParseTree(result.Stdout) {
		if strings.HasPrefix(entry.Mode, "100") {
			files = append(files, TreeFile{Path: entry.Path, Blob: entry.Blob})
		}
	}
	return files, nil
}

// TreeEntry is one line of git ls-tree -z: a file, link or submodule and its object.
type TreeEntry struct {
	Mode string
	Blob string
	Path string
}

func ParseTree(output string) []TreeEntry {
	var entries []TreeEntry
	for line := range strings.SplitSeq(output, "\x00") {
		header, path, ok := strings.Cut(line, "\t")
		if fields := strings.Fields(header); ok && len(fields) == 3 {
			entries = append(entries, TreeEntry{Mode: fields[0], Blob: fields[2], Path: path})
		}
	}
	return entries
}

// trackedFiles is cached until the index changes, as Activity's stats and
// language breakdown both list them.
func trackedFiles(ctx context.Context, repoPath string) ([]string, error) {
	gitDir, err := GitDir(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	return cachedByRefs(ctx, repoPath, "tracked-files", func() ([]string, error) {
		// -z, so a name Git would quote comes as it is.
		result, err := gitexec.RunIn(ctx, repoPath, "ls-files", "-z")
		if err != nil {
			return nil, err
		}
		return nulFields(result.Stdout), nil
	}, filepath.Join(gitDir, "index"))
}
