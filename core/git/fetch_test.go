package git

import (
	"context"
	"testing"
	"time"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestFetchUpdatesRemoteTrackingRef(t *testing.T) {
	remoteDir := t.TempDir() + "/remote.git"
	gittest.Run(t, "", "init", "--bare", "-b", "main", remoteDir)

	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "remote", "add", "origin", remoteDir)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	gittest.Run(t, dir, "push", "-u", "origin", "main")

	clone := t.TempDir()
	gittest.Run(t, "", "clone", "-q", remoteDir, clone)
	gittest.Run(t, clone, "config", "user.email", "test@example.com")
	gittest.Run(t, clone, "config", "user.name", "Test")
	gittest.WriteFile(t, clone, "file.txt", "v2")
	gittest.Run(t, clone, "add", "file.txt")
	gittest.Run(t, clone, "commit", "-m", "from clone")
	gittest.Run(t, clone, "push", "-q", "origin", "main")
	newSHA := gittest.Run(t, clone, "rev-parse", "HEAD")

	before := gittest.Run(t, dir, "rev-parse", "origin/main")
	if before == newSHA {
		t.Fatal("origin/main already matches the new commit before fetching")
	}

	if err := Fetch(context.Background(), dir, "origin"); err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	after := gittest.Run(t, dir, "rev-parse", "origin/main")
	if after != newSHA {
		t.Fatalf("origin/main = %q after fetch, want %q", after, newSHA)
	}
}

func TestFetchRejectsUnknownRemote(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init", "-b", "main")

	if err := Fetch(context.Background(), dir, "does-not-exist"); err == nil {
		t.Fatal("expected an error fetching an unconfigured remote")
	}
}

func TestFetchAllUpdatesEveryRemote(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")

	newSHAs := make(map[string]string)
	for _, name := range []string{"origin", "upstream"} {
		remoteDir := t.TempDir() + "/" + name + ".git"
		gittest.Run(t, "", "init", "--bare", "-b", "main", remoteDir)
		gittest.Run(t, dir, "remote", "add", name, remoteDir)
		gittest.Run(t, dir, "push", "-u", name, "main")

		clone := t.TempDir()
		gittest.Run(t, "", "clone", "-q", remoteDir, clone)
		gittest.Run(t, clone, "config", "user.email", "test@example.com")
		gittest.Run(t, clone, "config", "user.name", "Test")
		gittest.WriteFile(t, clone, "file.txt", "from "+name)
		gittest.Run(t, clone, "add", "file.txt")
		gittest.Run(t, clone, "commit", "-m", "update from "+name+" clone")
		gittest.Run(t, clone, "push", "-q", "origin", "main")
		newSHAs[name] = gittest.Run(t, clone, "rev-parse", "HEAD")
	}

	if err := FetchAll(context.Background(), dir); err != nil {
		t.Fatalf("FetchAll: %v", err)
	}

	for name, want := range newSHAs {
		got := gittest.Run(t, dir, "rev-parse", name+"/main")
		if got != want {
			t.Fatalf("%s/main = %q after FetchAll, want %q", name, got, want)
		}
	}
}

func TestFetchAllPruneDeletesGoneRemoteBranches(t *testing.T) {
	remoteDir := t.TempDir() + "/remote.git"
	gittest.Run(t, "", "init", "--bare", "-b", "main", remoteDir)

	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "remote", "add", "origin", remoteDir)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	gittest.Run(t, dir, "push", "-u", "origin", "main")
	gittest.Run(t, dir, "push", "origin", "main:feature")
	gittest.Run(t, dir, "fetch", "origin")
	gittest.Run(t, "", "--git-dir", remoteDir, "branch", "-D", "feature")

	if err := FetchAll(context.Background(), dir); err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if out := gittest.Run(t, dir, "branch", "-r", "--list", "origin/feature"); out == "" {
		t.Fatal("FetchAll removed origin/feature; only FetchAllPrune should")
	}

	if err := FetchAllPrune(context.Background(), dir); err != nil {
		t.Fatalf("FetchAllPrune: %v", err)
	}
	if out := gittest.Run(t, dir, "branch", "-r", "--list", "origin/feature"); out != "" {
		t.Fatalf("origin/feature still listed after FetchAllPrune: %q", out)
	}
	if out := gittest.Run(t, dir, "branch", "-r", "--list", "origin/main"); out == "" {
		t.Fatal("FetchAllPrune removed origin/main, which still exists on the remote")
	}
}

func TestLastFetchTimeNilBeforeAnyFetch(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init", "-b", "main")

	got, err := LastFetchTime(context.Background(), dir)
	if err != nil {
		t.Fatalf("LastFetchTime: %v", err)
	}
	if got != nil {
		t.Fatalf("got %v, want nil before any fetch has happened", got)
	}
}

func TestLastFetchTimeReflectsMostRecentFetch(t *testing.T) {
	remoteDir := t.TempDir() + "/remote.git"
	gittest.Run(t, "", "init", "--bare", "-b", "main", remoteDir)

	dir := gittest.InitRepo(t)
	gittest.Run(t, dir, "remote", "add", "origin", remoteDir)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	gittest.Run(t, dir, "push", "-u", "origin", "main")

	before := time.Now().Add(-time.Second)
	if err := Fetch(context.Background(), dir, "origin"); err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	got, err := LastFetchTime(context.Background(), dir)
	if err != nil {
		t.Fatalf("LastFetchTime: %v", err)
	}
	if got == nil {
		t.Fatal("got nil after a real fetch")
	}
	if got.Before(before) {
		t.Fatalf("got %v, want a time at or after %v", got, before)
	}
}
