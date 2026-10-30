package git

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestReachedAnswersFromTheTipsHistory(t *testing.T) {
	repo := gittest.InitRepo(t)
	commit := func(message string) string {
		gittest.Run(t, repo, "commit", "-q", "--allow-empty", "-m", message)
		return strings.TrimSpace(gittest.Run(t, repo, "rev-parse", "HEAD"))
	}
	first := commit("first")
	deployed := commit("deployed")
	after := commit("after the deploy")
	gittest.Run(t, repo, "switch", "-q", "-c", "side", first)
	side := commit("on a side branch")
	ctx := context.Background()

	got, err := Reached(ctx, repo, deployed, []string{first, deployed, after, side, "0000000000000000000000000000000000000001"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{first: true, deployed: true, after: false, side: false}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Reached = %v, want %v (the unknown commit left out)", got, want)
	}
	if got, err := Reached(ctx, repo, "0000000000000000000000000000000000000002", []string{first}); err != nil || len(got) != 0 {
		t.Fatalf("a tip that isn't fetched: %v, %v", got, err)
	}
	if got, _ := Reached(ctx, repo, deployed, []string{first[:7]}); !got[first[:7]] {
		t.Fatalf("an abbreviated commit: %v", got)
	}
}
