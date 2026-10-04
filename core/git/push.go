package git

import "context"

func Push(ctx context.Context, repoPath string) error {
	_, err := runResult(ctx, repoPath, "push")
	return err
}

func PushSetUpstream(ctx context.Context, repoPath, remote, branch string) error {
	_, err := runResult(ctx, repoPath, "push", "--set-upstream", remote, branch)
	return err
}

// ForcePush uses --force-with-lease so it can't discard commits pushed by
// someone else since the last fetch.
func ForcePush(ctx context.Context, repoPath, remote, branch string) error {
	_, err := runResult(ctx, repoPath, "push", "--force-with-lease", remote, branch)
	return err
}

// DeleteRemoteBranch deletes branch on remote. The full ref keeps a tag of
// the same name from being deleted instead; git drops the matching
// remote-tracking branch itself.
func DeleteRemoteBranch(ctx context.Context, repoPath, remote, branch string) error {
	_, err := runResult(ctx, repoPath, "push", remote, "--delete", "refs/heads/"+branch)
	return err
}
