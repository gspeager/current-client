package git

import (
	"context"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

// Git quotes a non-ASCII name, as "caf\303\251.txt", unless asked for -z.
func TestNonASCIINamesComeAsTheyAre(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "café.txt", "one", "add")
	gittest.CommitFile(t, dir, "café.txt", "two", "change")

	churn, err := ComputeFileChurn(context.Background(), dir, 30, 10)
	if err != nil || len(churn) != 1 || churn[0].Path != "café.txt" || churn[0].Changes != 2 {
		t.Fatalf("churn: %+v, %v", churn, err)
	}
	files, err := trackedFiles(context.Background(), dir)
	if err != nil || len(files) != 1 || files[0] != "café.txt" {
		t.Fatalf("tracked files: %q, %v", files, err)
	}
}

func TestRankKeepsFirstSeenOrderForTies(t *testing.T) {
	got := rank([]string{"b", "a", "b", "c", "a", "d"})
	want := []ranked{{"b", 2}, {"a", 2}, {"c", 1}, {"d", 1}}
	if len(got) != len(want) {
		t.Fatalf("rank = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("rank = %+v, want %+v", got, want)
		}
	}
}
