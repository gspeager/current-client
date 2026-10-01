package git

import (
	"context"
	"path/filepath"
	"strings"
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
	result, err := runResult(ctx, repoPath, "rev-parse", "--git-dir")
	if err != nil {
		return "", err
	}
	dir := strings.TrimSpace(result.Stdout)
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(repoPath, dir)
	}
	return dir, nil
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
