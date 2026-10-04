package git

import (
	"context"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestForcePushRewritesRemoteHistory(t *testing.T) {
	remoteDir := t.TempDir() + "/remote.git"
	gittest.Run(t, "", "init", "--bare", "-b", "main", remoteDir)

	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "remote", "add", "origin", remoteDir)
	gittest.CommitFile(t, dir, "file.txt", "v1", "A")
	gittest.Run(t, dir, "push", "-u", "origin", "main")
	firstSHA := gittest.Run(t, dir, "rev-parse", "HEAD")

	gittest.CommitFile(t, dir, "file.txt", "v2", "B")
	gittest.Run(t, dir, "push", "origin", "main")

	gittest.Run(t, dir, "reset", "--hard", firstSHA)
	gittest.CommitFile(t, dir, "file.txt", "v3", "C (rewrites B)")
	rewrittenSHA := gittest.Run(t, dir, "rev-parse", "HEAD")

	if err := Push(context.Background(), dir); err == nil {
		t.Fatal("expected a plain push to be rejected as non-fast-forward")
	}

	if err := ForcePush(context.Background(), dir, "origin", "main"); err != nil {
		t.Fatalf("ForcePush: %v", err)
	}

	remoteSHA := gittest.Run(t, "", "ls-remote", remoteDir, "refs/heads/main")
	if remoteSHA == "" || remoteSHA[:len(rewrittenSHA)] != rewrittenSHA {
		t.Fatalf("remote main = %q, want it to start with %q", remoteSHA, rewrittenSHA)
	}
}

func TestForcePushRejectsWhenRemoteMovedUnexpectedly(t *testing.T) {
	remoteDir := t.TempDir() + "/remote.git"
	gittest.Run(t, "", "init", "--bare", "-b", "main", remoteDir)

	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "remote", "add", "origin", remoteDir)
	gittest.CommitFile(t, dir, "file.txt", "v1", "A")
	gittest.Run(t, dir, "push", "-u", "origin", "main")

	// Someone else pushes to the remote without dir knowing (no fetch).
	otherClone := t.TempDir()
	gittest.Run(t, "", "clone", "-q", remoteDir, otherClone)
	gittest.Run(t, otherClone, "config", "user.email", "test@example.com")
	gittest.Run(t, otherClone, "config", "user.name", "Test")
	gittest.WriteFile(t, otherClone, "file.txt", "from someone else")
	gittest.Run(t, otherClone, "add", "file.txt")
	gittest.Run(t, otherClone, "commit", "-m", "someone else's commit")
	gittest.Run(t, otherClone, "push", "-q", "origin", "main")

	// dir rewrites its own history, still believing origin/main is where it
	// left it - a real force-with-lease should refuse this.
	gittest.Run(t, dir, "commit", "--amend", "-m", "A (amended)")

	if err := ForcePush(context.Background(), dir, "origin", "main"); err == nil {
		t.Fatal("expected --force-with-lease to reject a remote that moved since the last known state")
	}
}

// Local "feature" tracks "feat-x" on a remote named "team/origin", and the
// remote also has an unrelated "feature" that must not be touched.
func TestForcePushUpstreamUpdatesTheConfiguredUpstream(t *testing.T) {
	remoteDir := t.TempDir() + "/remote.git"
	gittest.Run(t, "", "init", "--bare", "-b", "main", remoteDir)
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "remote", "add", "team/origin", remoteDir)
	gittest.CommitFile(t, dir, "file.txt", "v1", "A")
	base := gittest.Run(t, dir, "rev-parse", "HEAD")
	gittest.Run(t, dir, "push", "team/origin", "main", "main:feat-x", "main:feature")
	gittest.Run(t, dir, "fetch", "-q", "team/origin")
	gittest.Run(t, dir, "checkout", "-q", "-b", "feature", "--track", "team/origin/feat-x")
	gittest.CommitFile(t, dir, "file.txt", "v2", "B")
	gittest.Run(t, dir, "push", "-q", "team/origin", "HEAD:feat-x")
	gittest.Run(t, dir, "reset", "-q", "--hard", base)
	gittest.CommitFile(t, dir, "file.txt", "v3", "C (rewrites B)")
	rewritten := gittest.Run(t, dir, "rev-parse", "HEAD")

	if err := ForcePushUpstream(context.Background(), dir); err != nil {
		t.Fatalf("ForcePushUpstream: %v", err)
	}

	if got := gittest.Run(t, "", "--git-dir", remoteDir, "rev-parse", "feat-x"); got != rewritten {
		t.Fatalf("remote feat-x = %q, want the rewritten %q", got, rewritten)
	}
	if got := gittest.Run(t, "", "--git-dir", remoteDir, "rev-parse", "feature"); got != base {
		t.Fatalf("remote feature = %q, want it untouched at %q", got, base)
	}
}

func TestForcePushUpstreamRefusesWithoutUpstream(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "A")

	err := ForcePushUpstream(context.Background(), dir)
	if err == nil || err.Error() != "No upstream configured to force-push to." {
		t.Fatalf("err = %v, want the no-upstream message", err)
	}
}

func TestForcePushUpstreamRejectsWhenUpstreamMovedSinceFetch(t *testing.T) {
	remoteDir := t.TempDir() + "/remote.git"
	gittest.Run(t, "", "init", "--bare", "-b", "main", remoteDir)
	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "remote", "add", "origin", remoteDir)
	gittest.CommitFile(t, dir, "file.txt", "v1", "A")
	gittest.Run(t, dir, "push", "-u", "origin", "main:feat-x")
	gittest.Run(t, dir, "branch", "-q", "--set-upstream-to=origin/feat-x")

	clone := t.TempDir()
	gittest.Run(t, "", "clone", "-q", "-b", "feat-x", remoteDir, clone)
	gittest.Run(t, clone, "config", "user.email", "test@example.com")
	gittest.Run(t, clone, "config", "user.name", "Test")
	gittest.CommitFile(t, clone, "theirs.txt", "theirs", "someone else's work")
	gittest.Run(t, clone, "push", "-q", "origin", "feat-x")
	theirs := gittest.Run(t, clone, "rev-parse", "HEAD")

	gittest.CommitFile(t, dir, "file.txt", "v2", "mine")
	if err := ForcePushUpstream(context.Background(), dir); err == nil {
		t.Fatal("force push overwrote work it hadn't fetched")
	}
	if got := gittest.Run(t, "", "--git-dir", remoteDir, "rev-parse", "feat-x"); got != theirs {
		t.Fatalf("remote feat-x = %q, want their commit %q kept", got, theirs)
	}
}
