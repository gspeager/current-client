package git

import (
	"context"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

// LFSPaths returns which of paths .gitattributes stores in Git LFS
// (filter=lfs). Paths go in on stdin, so any number fit.
func LFSPaths(ctx context.Context, repoPath string, paths []string) (map[string]bool, error) {
	lfs := map[string]bool{}
	if len(paths) == 0 {
		return lfs, nil
	}
	result, err := gitexec.NewExecutor("").RunChecked(ctx, gitexec.Command{
		Dir:   repoPath,
		Args:  []string{"check-attr", "--stdin", "-z", "filter"},
		Stdin: strings.NewReader(strings.Join(paths, "\x00") + "\x00"),
	})
	if err != nil {
		return nil, err
	}
	// Triples of path, attribute and value.
	fields := strings.Split(result.Stdout, "\x00")
	for i := 0; i+2 < len(fields); i += 3 {
		if fields[i+2] == "lfs" {
			lfs[fields[i]] = true
		}
	}
	return lfs, nil
}

// LFSInstalled reports whether the git-lfs extension is available. Without it,
// files stored in LFS are left as small pointer files in the working tree.
func LFSInstalled(ctx context.Context) bool {
	result, err := gitexec.NewExecutor("").Run(ctx, gitexec.Command{Args: []string{"lfs", "version"}})
	return err == nil && result.ExitCode == 0
}
