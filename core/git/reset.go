package git

import "context"

type ResetMode string

const (
	ResetSoft  ResetMode = "soft"
	ResetMixed ResetMode = "mixed"
	ResetHard  ResetMode = "hard"
	// ResetKeep moves HEAD but keeps local changes, refusing when a changed
	// file also differs between the two commits.
	ResetKeep ResetMode = "keep"
)

func Reset(ctx context.Context, repoPath, ref string, mode ResetMode) error {
	_, err := runResult(ctx, repoPath, "reset", "--"+string(mode), ref)
	return err
}
