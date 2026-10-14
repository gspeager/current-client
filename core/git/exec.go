package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gspeager/current-client/core/gitexec"
)

func runResult(ctx context.Context, repoPath string, args ...string) (gitexec.Result, error) {
	return gitexec.NewExecutor("").RunChecked(ctx, gitexec.Command{
		Dir:  repoPath,
		Args: args,
	})
}

// GitDir resolves the real .git directory, which isn't "<repoPath>/.git" for
// a worktree.
func GitDir(ctx context.Context, repoPath string) (string, error) {
	dirs, err := gitDirs(ctx, repoPath)
	return dirs[0], err
}

// CommonDir resolves the .git directory a repository's refs live in, which
// all its worktrees share; outside a worktree it's the same as GitDir.
func CommonDir(ctx context.Context, repoPath string) (string, error) {
	dirs, err := gitDirs(ctx, repoPath)
	return dirs[1], err
}

var dirCache sync.Map

// gitDirs asks git once per repository path; the answer only changes if the
// repository is removed, which the existence check catches.
func gitDirs(ctx context.Context, repoPath string) ([2]string, error) {
	if dirs, ok := dirCache.Load(repoPath); ok {
		if _, err := os.Stat(dirs.([2]string)[0]); err == nil {
			return dirs.([2]string), nil
		}
		dirCache.Delete(repoPath)
	}
	dirs, err, _ := flights.Do("dirs\x00"+repoPath, func() (any, error) {
		result, err := runResult(ctx, repoPath, "rev-parse", "--git-dir", "--git-common-dir")
		if err != nil {
			return [2]string{}, err
		}
		var dirs [2]string
		for i, dir := range strings.SplitN(strings.TrimSpace(result.Stdout), "\n", 2) {
			dir = strings.TrimSpace(dir)
			if !filepath.IsAbs(dir) {
				dir = filepath.Join(repoPath, dir)
			}
			dirs[i] = dir
		}
		dirCache.Store(repoPath, dirs)
		return dirs, nil
	})
	return dirs.([2]string), err
}

// sinceMidnight formats a --since value with an explicit time: a bare
// "YYYY-MM-DD" was observed (git 2.53) to sometimes exclude that whole day.
func sinceMidnight(day time.Time) string {
	return day.Format("2006-01-02") + " 00:00:00"
}

// hasNoCommits tells a HEAD-based command that failed because the repository
// is empty apart from one that failed for a real reason.
func hasNoCommits(ctx context.Context, repoPath string) bool {
	_, err := runResult(ctx, repoPath, "rev-parse", "--verify", "--quiet", "HEAD")
	return err != nil
}

// currentBranchName also works before the first commit, when HEAD names a
// branch that doesn't exist yet. A detached HEAD reads as "HEAD".
func currentBranchName(ctx context.Context, repoPath string) (string, error) {
	result, err := runResult(ctx, repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err == nil {
		return strings.TrimSpace(result.Stdout), nil
	}
	unborn, symErr := runResult(ctx, repoPath, "symbolic-ref", "--quiet", "--short", "HEAD")
	if symErr != nil {
		return "", err
	}
	return strings.TrimSpace(unborn.Stdout), nil
}
