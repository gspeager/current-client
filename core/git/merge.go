package git

import "context"

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
