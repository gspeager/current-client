package git

import (
	"context"
	"reflect"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestMergeBasesAnswersEachPair(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "a.txt", "a\n", "base")
	base := gittest.Run(t, dir, "rev-parse", "HEAD")
	gittest.Run(t, dir, "switch", "-q", "-c", "feature")
	gittest.CommitFile(t, dir, "f.txt", "f\n", "feature")
	gittest.Run(t, dir, "switch", "-q", "--orphan", "gh-pages")
	gittest.CommitFile(t, dir, "index.html", "site\n", "site")
	gittest.Run(t, dir, "switch", "-q", "main")
	gittest.CommitFile(t, dir, "b.txt", "b\n", "main moves on")
	tip := gittest.Run(t, dir, "rev-parse", "HEAD")

	bases, err := MergeBases(context.Background(), dir, [][2]string{{"main", "feature"}, {"feature", "main"}, {"main", "gh-pages"}, {"main", "main"}, {"feature", "feature~1"}})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{base, base, "", tip, base}; !reflect.DeepEqual(bases, want) {
		t.Errorf("MergeBases = %v, want %v", bases, want)
	}
	if _, err := MergeBases(context.Background(), dir, [][2]string{{"main", "nope"}}); err == nil {
		t.Error("a missing revision should fail")
	}
}
