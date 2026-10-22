package overlap

import (
	"context"
	"reflect"
	"testing"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestPredictTipsChecksFetchedHeadsAndRemembersThem(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "shared.txt", "base\n", "base")
	gittest.Run(t, dir, "switch", "-q", "-c", "conflicting")
	gittest.CommitFile(t, dir, "shared.txt", "theirs\n", "theirs")
	conflicting := gittest.Run(t, dir, "rev-parse", "HEAD")
	gittest.Run(t, dir, "switch", "-q", "-c", "clean", "main")
	gittest.CommitFile(t, dir, "other.txt", "new\n", "other")
	clean := gittest.Run(t, dir, "rev-parse", "HEAD")
	gittest.Run(t, dir, "switch", "-q", "main")
	gittest.CommitFile(t, dir, "shared.txt", "ours\n", "ours")
	// The heads live only under refs/current-client/, as a host's change heads do.
	gittest.Run(t, dir, "update-ref", git.HostChangeRef("origin", 7), conflicting)
	gittest.Run(t, dir, "update-ref", git.HostChangeRef("origin", 8), clean)
	gittest.Run(t, dir, "branch", "-q", "-D", "conflicting", "clean")

	merges := 0
	previous := mergeMany
	mergeMany = func(ctx context.Context, repoPath, ours string, theirs []string) ([][]string, error) {
		merges++
		return previous(ctx, repoPath, ours, theirs)
	}
	t.Cleanup(func() { mergeMany = previous })

	got, err := PredictTips(context.Background(), dir, []string{conflicting, clean})
	if err != nil {
		t.Fatalf("PredictTips: %v", err)
	}
	want := [][]string{{"shared.txt"}, nil}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PredictTips = %+v, want %+v", got, want)
	}
	if _, err := PredictTips(context.Background(), dir, []string{conflicting, clean}); err != nil {
		t.Fatal(err)
	}
	if merges != 1 {
		t.Fatalf("unchanged heads ran %d merges, want 1", merges)
	}
	if refs, _ := git.ListRefs(context.Background(), dir); len(refs) != 1 {
		t.Fatalf("the fetched heads should stay out of ListRefs: %+v", refs)
	}
}
