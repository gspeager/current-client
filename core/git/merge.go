package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type MergeMode string

const (
	// MergeFastForward fast-forwards when it can, as plain `git merge` does.
	MergeFastForward MergeMode = ""
	// MergeCommit always records a merge commit (--no-ff).
	MergeCommit MergeMode = "no-ff"
	// MergeSquash stages the branch's changes as one change and leaves the
	// commit to the caller (--squash).
	MergeSquash MergeMode = "squash"
)

func MergeBranch(ctx context.Context, repoPath, branch string) error {
	return MergeBranchMode(ctx, repoPath, branch, MergeFastForward)
}

func MergeBranchMode(ctx context.Context, repoPath, branch string, mode MergeMode) error {
	args := []string{"merge", "--no-edit"}
	switch mode {
	case MergeCommit:
		args = append(args, "--no-ff")
	case MergeSquash:
		args = append(args, "--squash")
	}
	_, err := runResult(ctx, repoPath, append(args, branch)...)
	return err
}

// SquashedSubjects lists the subjects of the commits a pending squash merge
// staged, newest first, read from Git's SQUASH_MSG. It's nil when no squash is
// pending; committing or resetting removes the file.
func SquashedSubjects(ctx context.Context, repoPath string) ([]string, error) {
	dir, err := GitDir(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "SQUASH_MSG"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// Each commit is "commit <sha>", header lines, a blank line, then its
	// message indented by four spaces.
	var subjects []string
	awaitingSubject := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case strings.HasPrefix(line, "commit "):
			awaitingSubject = true
		case awaitingSubject && strings.HasPrefix(line, "    "):
			subjects = append(subjects, strings.TrimSpace(line))
			awaitingSubject = false
		}
	}
	return subjects, nil
}
