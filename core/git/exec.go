package git

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gspeager/current-client/core/gitexec"
)

// GitDir resolves the real .git directory, which isn't "<repoPath>/.git" for
// a worktree.
func GitDir(ctx context.Context, repoPath string) (string, error) {
	dirs, err := gitDirs(ctx, repoPath)
	return dirs.gitDir, err
}

// CommonDir resolves the .git directory a repository's refs live in, which
// all its worktrees share; outside a worktree it's the same as GitDir.
func CommonDir(ctx context.Context, repoPath string) (string, error) {
	dirs, err := gitDirs(ctx, repoPath)
	return dirs.commonDir, err
}

type repoDirs struct {
	gitDir    string
	commonDir string
}

var dirCache sync.Map

// gitDirs asks git once per repository path; the answer only changes if the
// repository is removed, which the existence check catches.
func gitDirs(ctx context.Context, repoPath string) (repoDirs, error) {
	if cached, ok := dirCache.Load(repoPath); ok {
		dirs := cached.(repoDirs)
		if _, err := os.Stat(dirs.gitDir); err == nil {
			return dirs, nil
		}
		dirCache.Delete(repoPath)
	}
	dirs, err, _ := flights.Do("dirs\x00"+repoPath, func() (any, error) {
		result, err := gitexec.RunIn(ctx, repoPath, "rev-parse", "--git-dir", "--git-common-dir")
		if err != nil {
			return repoDirs{}, err
		}
		gitDir, commonDir, _ := strings.Cut(strings.TrimSpace(result.Stdout), "\n")
		dirs := repoDirs{gitDir: absoluteIn(repoPath, gitDir), commonDir: absoluteIn(repoPath, commonDir)}
		dirCache.Store(repoPath, dirs)
		return dirs, nil
	})
	return dirs.(repoDirs), err
}

func absoluteIn(repoPath, dir string) string {
	dir = strings.TrimSpace(dir)
	if filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(repoPath, dir)
}

// sinceMidnight formats a --since value with an explicit time: a bare
// "YYYY-MM-DD" was observed (git 2.53) to sometimes exclude that whole day.
func sinceMidnight(day time.Time) string {
	return day.Format("2006-01-02") + " 00:00:00"
}

// hasNoCommits tells a HEAD-based command that failed because the repository
// is empty apart from one that failed for a real reason.
func hasNoCommits(ctx context.Context, repoPath string) bool {
	_, err := gitexec.RunIn(ctx, repoPath, "rev-parse", "--verify", "--quiet", "HEAD")
	return err != nil
}

// runStoppable runs a command that can stop on conflicts. A stop that leaves
// the operation in progress isn't a failure: the conflict banner finishes it.
func runStoppable(ctx context.Context, cmd gitexec.Command) (gitexec.Result, error) {
	result, err := gitexec.Run(ctx, cmd)
	if err != nil {
		return gitexec.Result{}, gitexec.WrapRunError(err)
	}
	if result.ExitCode == 0 {
		return result, nil
	}
	if strings.Contains(result.Stdout, "CONFLICT (") {
		if state, err := DetectConflictState(ctx, cmd.Dir); err == nil && state.Operation != ConflictNone {
			return result, nil
		}
	}
	return gitexec.Result{}, gitexec.WrapResult(result)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// HeadBranch is the checked-out branch's short name; it fails on a detached HEAD.
func HeadBranch(ctx context.Context, repoPath string) (string, error) {
	return gitexec.OutputIn(ctx, repoPath, nil, "symbolic-ref", "--quiet", "--short", "HEAD")
}

// lines are output's lines, trimmed, without empty ones.
func lines(output string) []string {
	var kept []string
	for line := range strings.SplitSeq(output, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			kept = append(kept, line)
		}
	}
	return kept
}

// nulFields are -z output's fields, without empty ones.
func nulFields(output string) []string {
	var kept []string
	for field := range strings.SplitSeq(output, "\x00") {
		if field != "" {
			kept = append(kept, field)
		}
	}
	return kept
}

type ranked struct {
	key   string
	count int
}

// rank counts each key, most first and ties in the order first seen.
func rank(keys []string) []ranked {
	counts := map[string]int{}
	var order []string
	for _, key := range keys {
		if counts[key] == 0 {
			order = append(order, key)
		}
		counts[key]++
	}
	ranks := make([]ranked, len(order))
	for i, key := range order {
		ranks[i] = ranked{key, counts[key]}
	}
	slices.SortStableFunc(ranks, func(a, b ranked) int { return b.count - a.count })
	return ranks
}
