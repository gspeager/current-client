// Package readthrough lists what a branch or a worktree changes against a base,
// and keeps which of those files have been read.
package readthrough

import (
	"context"
	"os"

	"github.com/gspeager/current-client/core/diff"
	"github.com/gspeager/current-client/core/filekind"
	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
	"github.com/gspeager/current-client/core/snapshot"
)

// Range is from the commit Base to the commit Head, or to the working tree at
// repoPath when Uncommitted.
type Range struct {
	Base        string
	Head        string
	Uncommitted bool
}

type File struct {
	git.TreeChange
	Kind filekind.Kind
}

// Files lists what the range changes. skipped counts untracked files too large
// to include.
func Files(ctx context.Context, repoPath string, r Range) (files []File, skipped int, err error) {
	env, discard := git.ScratchObjects(ctx, repoPath)
	defer discard()
	to := r.Head
	if r.Uncommitted {
		if to, skipped, err = snapshot.WorkingTree(ctx, repoPath, env); err != nil {
			return nil, 0, err
		}
	}
	changes, err := git.TreeChanges(ctx, repoPath, env, r.Base, to)
	if err != nil {
		return nil, 0, err
	}
	for _, change := range changes {
		files = append(files, File{change, filekind.Of(change.Path)})
	}
	return files, skipped, nil
}

// Base is where head left target. A branch built on another is compared with the
// commit it was built on, which an amended parent no longer holds.
func Base(ctx context.Context, repoPath, target, head, builtOn string) (string, error) {
	base, err := git.MergeBase(ctx, repoPath, target, head)
	if err != nil {
		return "", err
	}
	if builtOn != "" {
		if shared, err := git.MergeBase(ctx, repoPath, builtOn, head); err == nil && shared == builtOn {
			if older, err := git.MergeBase(ctx, repoPath, base, builtOn); base == "" || (err == nil && older == base) {
				return builtOn, nil
			}
		}
	}
	if base == "" {
		return "", git.ErrUnrelatedHistories
	}
	return base, nil
}

// FileDiff is one file's diff over the range.
func FileDiff(ctx context.Context, repoPath string, r Range, origPath, path string, ignoreWhitespace bool) (diff.FileDiff, error) {
	if !r.Uncommitted {
		return diff.GetRangeDiff(ctx, repoPath, nil, r.Base, r.Head, origPath, path, ignoreWhitespace)
	}
	dir, err := os.MkdirTemp("", "current-client-index")
	if err != nil {
		return diff.FileDiff{}, err
	}
	defer os.RemoveAll(dir)
	env, err := snapshot.CopyIndex(ctx, repoPath, dir)
	if err != nil {
		return diff.FileDiff{}, err
	}
	// So an untracked file is compared as a tracked one is. A deleted one has
	// nothing to add, and an ignored one was never listed.
	_, _ = gitexec.NewExecutor("").Run(ctx, gitexec.Command{
		Dir:  repoPath,
		Env:  env,
		Args: []string{"add", "--intent-to-add", "--", ":(literal)" + path},
	})
	return diff.GetRangeDiff(ctx, repoPath, env, r.Base, "", origPath, path, ignoreWhitespace)
}
