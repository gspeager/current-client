package git

import (
	"context"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestPushSetUpstreamPublishesBranchAndSetsTracking(t *testing.T) {
	remoteDir := t.TempDir() + "/remote.git"
	gittest.Run(t, "", "init", "--bare", "-b", "main", remoteDir)

	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "remote", "add", "origin", remoteDir)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")

	if err := PushSetUpstream(context.Background(), dir, "origin", "main"); err != nil {
		t.Fatalf("PushSetUpstream: %v", err)
	}

	remoteBranches := gittest.Run(t, "", "ls-remote", "--heads", remoteDir, "main")
	if remoteBranches == "" {
		t.Fatal("expected main to exist on the remote after push")
	}

	upstream := gittest.Run(t, dir, "rev-parse", "--abbrev-ref", "main@{upstream}")
	if upstream != "origin/main" {
		t.Fatalf("upstream = %q, want %q", upstream, "origin/main")
	}
}

func TestPushRejectsWhenNoUpstreamConfigured(t *testing.T) {
	remoteDir := t.TempDir() + "/remote.git"
	gittest.Run(t, "", "init", "--bare", "-b", "main", remoteDir)

	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "remote", "add", "origin", remoteDir)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")

	if err := Push(context.Background(), dir); err == nil {
		t.Fatal("expected an error pushing with no upstream configured")
	}
}
