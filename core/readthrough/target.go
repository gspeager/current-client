package readthrough

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
)

// WorktreeKey starts the key of a worktree that has no branch checked out.
const WorktreeKey = "worktree:"

type Target struct {
	// Key names the marks and notes of the read through.
	Key  string
	Name string
	// Path is the working tree its commands run in.
	Path string
	Head string
	// CheckedOut is whether the ref is what's checked out at Path.
	CheckedOut bool
}

// Resolve finds a ref by its full name or, when path is set, the worktree at path.
func Resolve(ctx context.Context, repoPath, ref, path string) (Target, error) {
	if path != "" {
		t := Target{Path: path, CheckedOut: true, Key: WorktreeKey + path, Name: filepath.Base(path)}
		head, err := git.ResolveCommit(ctx, path, "HEAD")
		if err != nil {
			return t, err
		}
		t.Head = head
		worktrees, err := git.ListWorktrees(ctx, path)
		if err != nil {
			return t, err
		}
		for _, worktree := range worktrees {
			if !worktree.Current {
				continue
			}
			t.Key = WorktreeKey + worktree.Path
			if worktree.Branch != "" {
				t.Key, t.Name = "refs/heads/"+worktree.Branch, worktree.Branch
			}
		}
		return t, nil
	}
	t := Target{Key: ref, Name: strings.TrimPrefix(strings.TrimPrefix(ref, "refs/heads/"), "refs/remotes/")}
	head, err := git.ResolveCommit(ctx, repoPath, ref)
	if err != nil {
		return t, err
	}
	t.Head = head
	branches, err := git.ListBranches(ctx, repoPath)
	if err != nil {
		return t, err
	}
	for _, branch := range branches {
		if "refs/heads/"+branch.Name != ref {
			continue
		}
		t.CheckedOut = branch.Current
		if _, err := os.Stat(branch.WorktreePath); !branch.Current && branch.WorktreePath != "" && err == nil {
			t.Path, t.CheckedOut = branch.WorktreePath, true
		}
	}
	return t, nil
}

// SettleBase picks the target, when none was asked for, from a stacked branch's
// parent or the compare base, and the base head is compared with. settled is ""
// when there's nothing to compare with.
func SettleBase(ctx context.Context, repoPath, key, name, target, head string, links []git.StackLink) (settled, base string, err error) {
	branch := strings.TrimPrefix(key, "refs/heads/")
	builtOn := ""
	for _, link := range links {
		if link.Branch != branch || (target != "" && target != link.Parent) {
			continue
		}
		if _, err := git.ResolveCommit(ctx, repoPath, "refs/heads/"+link.Parent); err == nil {
			target, builtOn = link.Parent, link.ParentTip
		}
	}
	if target == "" {
		if target, err = git.CompareBase(ctx, repoPath); err != nil || target == "" {
			return "", "", err
		}
	}
	tip, err := git.ResolveCommit(ctx, repoPath, target)
	if err != nil {
		return target, "", &gitexec.AppError{Message: target + " doesn't exist.", Detail: err.Error()}
	}
	base, err = Base(ctx, repoPath, tip, head, builtOn)
	if errors.Is(err, git.ErrUnrelatedHistories) {
		return target, "", &gitexec.AppError{Message: name + " and " + target + " share no history."}
	}
	return target, base, err
}

func ForgetGone(ctx context.Context, repoPath string) error {
	gone, err := Gone(ctx, repoPath)
	if err != nil {
		return err
	}
	return Forget(ctx, repoPath, gone)
}

// Gone returns whether the branch or worktree a key names has been deleted.
func Gone(ctx context.Context, repoPath string) (func(key string) bool, error) {
	worktrees, err := git.ListWorktrees(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	refs, err := gitexec.RunIn(ctx, repoPath, "for-each-ref", "--format=%(refname)")
	if err != nil {
		return nil, err
	}
	present := map[string]bool{}
	for _, ref := range strings.Split(refs.Stdout, "\n") {
		present[ref] = true
	}
	for _, worktree := range worktrees {
		present[WorktreeKey+worktree.Path] = true
	}
	return func(key string) bool { return !present[key] }, nil
}
