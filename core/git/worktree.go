package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

type Worktree struct {
	Path string
	// Branch is "" when the worktree's HEAD is detached.
	Branch string
	Head   string
	// Main is the repository's own working tree, which can't be removed.
	Main bool
	// Current is the worktree repoPath is in.
	Current bool
	Locked  bool
	// Missing means its folder is gone; removing it just forgets it.
	Missing bool
}

// ListWorktrees takes repoPath as the top of a working tree, as Open returns
// it. Without linked worktrees, the main one comes from the cached branch list.
func ListWorktrees(ctx context.Context, repoPath string) ([]Worktree, error) {
	if main, ok := onlyWorktree(ctx, repoPath); ok {
		return []Worktree{main}, nil
	}
	result, err := runResult(ctx, repoPath, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	worktrees := parseWorktrees(result.Stdout)
	for i := range worktrees {
		worktrees[i].Current = samePath(worktrees[i].Path, repoPath)
	}
	return worktrees, nil
}

// onlyWorktree is false when there may be linked worktrees, or HEAD isn't on
// a branch, which leaves git to say.
func onlyWorktree(ctx context.Context, repoPath string) (Worktree, bool) {
	commonDir, err := CommonDir(ctx, repoPath)
	if err != nil {
		return Worktree{}, false
	}
	if linked, err := os.ReadDir(filepath.Join(commonDir, "worktrees")); err == nil && len(linked) > 0 {
		return Worktree{}, false
	}
	list, err := listBranches(ctx, repoPath)
	if err != nil || list.head == "" {
		return Worktree{}, false
	}
	for _, b := range list.branches {
		if b.Current {
			return Worktree{Path: repoPath, Branch: b.Name, Head: list.head, Main: true, Current: true}, true
		}
	}
	return Worktree{}, false
}

// Blocks of "key value" lines separated by blank lines; the first is the main worktree.
func parseWorktrees(output string) []Worktree {
	var worktrees []Worktree
	for _, block := range strings.Split(strings.TrimSpace(output), "\n\n") {
		var w Worktree
		for _, line := range strings.Split(block, "\n") {
			key, value, _ := strings.Cut(line, " ")
			switch key {
			case "worktree":
				w.Path = filepath.FromSlash(value)
			case "HEAD":
				w.Head = value
			case "branch":
				w.Branch = strings.TrimPrefix(value, "refs/heads/")
			case "locked":
				w.Locked = true
			case "prunable":
				w.Missing = true
			}
		}
		if w.Path != "" {
			w.Main = len(worktrees) == 0
			worktrees = append(worktrees, w)
		}
	}
	return worktrees
}

// samePath compares two absolute paths as the file system would, resolving
// symlinks (macOS's /tmp is /private/tmp) and ignoring case on Windows.
func samePath(a, b string) bool {
	resolve := func(p string) string {
		p = filepath.Clean(filepath.FromSlash(p))
		if r, err := filepath.EvalSymlinks(p); err == nil {
			return r
		}
		return p
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(resolve(a), resolve(b))
	}
	return resolve(a) == resolve(b)
}

// AddWorktree checks out branch in a new worktree at path. With create, branch
// is made first, at startPoint ("" for HEAD).
func AddWorktree(ctx context.Context, repoPath, path, branch string, create bool, startPoint string) error {
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		return &gitexec.AppError{Message: "That folder already exists.", Detail: path}
	}
	args := []string{"worktree", "add"}
	if create {
		args = append(args, "-b", branch, path)
		if startPoint != "" {
			args = append(args, startPoint)
		}
	} else {
		args = append(args, path, branch)
	}
	_, err := runResult(ctx, repoPath, args...)
	return err
}

// RemoveWorktree deletes the worktree's folder and forgets it. Without force,
// Git refuses when it has uncommitted changes.
func RemoveWorktree(ctx context.Context, repoPath, path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	_, err := runResult(ctx, repoPath, append(args, path)...)
	return err
}
