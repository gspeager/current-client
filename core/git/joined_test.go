package git

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestJoinedIntoFollowsTheFirstParentLine(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "a.txt", "a\n", "base")
	base := gittest.Run(t, dir, "rev-parse", "HEAD")
	gittest.Run(t, dir, "switch", "-q", "-c", "feature")
	gittest.CommitFile(t, dir, "f.txt", "f\n", "feature")
	feature := gittest.Run(t, dir, "rev-parse", "HEAD")
	gittest.Run(t, dir, "switch", "-q", "-c", "nested")
	gittest.CommitFile(t, dir, "n.txt", "n\n", "nested")
	gittest.Run(t, dir, "switch", "-q", "feature")
	gittest.Run(t, dir, "merge", "-q", "--no-ff", "-m", "merge nested", "nested")
	gittest.Run(t, dir, "switch", "-q", "main")
	gittest.CommitFile(t, dir, "b.txt", "b\n", "main moves on")
	gittest.Run(t, dir, "merge", "-q", "--no-ff", "-m", "merge feature", "feature")
	merge := gittest.Run(t, dir, "rev-parse", "HEAD")
	gittest.Run(t, dir, "switch", "-q", "-c", "ff")
	gittest.CommitFile(t, dir, "c.txt", "c\n", "fast-forwarded")
	ff := gittest.Run(t, dir, "rev-parse", "HEAD")
	gittest.Run(t, dir, "switch", "-q", "main")
	gittest.Run(t, dir, "merge", "-q", "--ff-only", "ff")
	gittest.Run(t, dir, "switch", "-q", "-c", "open", base)
	gittest.CommitFile(t, dir, "o.txt", "o\n", "still open")
	gittest.Run(t, dir, "switch", "-q", "--orphan", "gh-pages")
	gittest.CommitFile(t, dir, "index.html", "site\n", "site")
	gittest.Run(t, dir, "switch", "-q", "main")
	mergeDate, _ := strconv.ParseInt(gittest.Run(t, dir, "log", "-1", "--format=%ct", merge), 10, 64)

	joined, err := JoinedInto(context.Background(), dir, "main", []string{feature, "nested", "feature", "ff", base, "open", "gh-pages"})
	if err != nil {
		t.Fatal(err)
	}
	want := []Joined{
		{At: merge, Date: time.Unix(mergeDate, 0)},
		{At: merge, Date: time.Unix(mergeDate, 0)},
		{At: merge, Date: time.Unix(mergeDate, 0)},
		{At: ff, OnLine: true, Date: joined[3].Date},
		{At: base, OnLine: true, Date: joined[4].Date},
		{},
		{},
	}
	for i, got := range joined {
		if got != want[i] {
			t.Errorf("%d: got %+v, want %+v", i, got, want[i])
		}
	}
	if joined[3].Date.IsZero() || joined[4].Date.IsZero() {
		t.Errorf("line commits have dates: %+v %+v", joined[3], joined[4])
	}
}

func TestJoinedIntoWithNothingAsked(t *testing.T) {
	joined, err := JoinedInto(context.Background(), t.TempDir(), "main", nil)
	if err != nil || len(joined) != 0 {
		t.Fatalf("joined = %v, %v", joined, err)
	}
}
