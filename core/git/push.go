package git

import (
	"context"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

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

// ForcePushUpstream force-pushes the current branch to exactly its configured
// upstream, which can have a different name from the local branch and sit on a
// remote whose name contains slashes. Like ForcePush it uses --force-with-lease.
func ForcePushUpstream(ctx context.Context, repoPath string) error {
	head, err := runResult(ctx, repoPath, "symbolic-ref", "-q", "HEAD")
	if err != nil {
		return &gitexec.AppError{Message: "Check out a branch to force-push it."}
	}
	upstream, err := runResult(ctx, repoPath, "for-each-ref",
		"--format=%(upstream:remotename)%09%(upstream:remoteref)", strings.TrimSpace(head.Stdout))
	if err != nil {
		return err
	}
	remote, remoteRef, _ := strings.Cut(strings.TrimSpace(upstream.Stdout), "\t")
	// "." is a local branch tracking another local branch, not a remote.
	if remote == "" || remote == "." || remoteRef == "" {
		return &gitexec.AppError{Message: "No upstream configured to force-push to."}
	}
	_, err = runResult(ctx, repoPath, "push", "--force-with-lease", remote, "HEAD:"+remoteRef)
	return err
}
