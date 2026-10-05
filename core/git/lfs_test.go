package git

import (
	"context"
	"reflect"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestLFSPaths(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, ".gitattributes", "*.psd filter=lfs diff=lfs merge=lfs -text\n", "track psd in lfs")

	got, err := LFSPaths(context.Background(), dir, []string{"art/cover.psd", "README.md", "with space.psd"})
	if err != nil {
		t.Fatalf("LFSPaths: %v", err)
	}
	if want := map[string]bool{"art/cover.psd": true, "with space.psd": true}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if got, _ := LFSPaths(context.Background(), dir, nil); len(got) != 0 {
		t.Fatalf("no paths = %v", got)
	}
}
