package git

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestWatchedPathsIncludesTrackedAndUntrackedNotIgnored(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "tracked.txt", "v1", "initial")
	gittest.WriteFile(t, dir, "untracked.txt", "v1")
	gittest.WriteFile(t, dir, "ignored.txt", "v1")
	gittest.CommitFile(t, dir, ".gitignore", "ignored.txt\n", "add gitignore")

	got, err := WatchedPaths(context.Background(), dir)
	if err != nil {
		t.Fatalf("WatchedPaths: %v", err)
	}
	sort.Strings(got)
	want := []string{".gitignore", "tracked.txt", "untracked.txt"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v (ignored.txt must not appear)", got, want)
	}
}
