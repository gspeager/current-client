package git

import (
	"context"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
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

// TreeFile is a file at a commit or tree. Symbolic links and submodules aren't.
type TreeFile struct {
	Path string
	Blob string
}

func TreeFiles(ctx context.Context, repoPath string, env []string, rev string) ([]TreeFile, error) {
	result, err := gitexec.NewExecutor("").RunChecked(ctx, gitexec.Command{
		Dir:  repoPath,
		Env:  env,
		Args: []string{"ls-tree", "-r", "-z", "--full-tree", rev},
	})
	if err != nil {
		return nil, err
	}
	var files []TreeFile
	for entry := range strings.SplitSeq(result.Stdout, "\x00") {
		header, path, ok := strings.Cut(entry, "\t")
		if fields := strings.Fields(header); ok && len(fields) == 3 && strings.HasPrefix(fields[0], "100") {
			files = append(files, TreeFile{Path: path, Blob: fields[2]})
		}
	}
	return files, nil
}
