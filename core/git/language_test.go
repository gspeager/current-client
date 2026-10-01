package git

import (
	"context"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestLanguageBreakdown(t *testing.T) {
	dir := gittest.InitRepo(t)

	for _, name := range []string{"a.go", "b.go", "c.go", "main.ts", "README", "LICENSE"} {
		gittest.WriteFile(t, dir, name, "v1")
	}
	gittest.Run(t, dir, "add", ".")
	gittest.Run(t, dir, "commit", "-m", "initial")

	stats, err := LanguageBreakdown(context.Background(), dir, 10)
	if err != nil {
		t.Fatalf("LanguageBreakdown: %v", err)
	}
	if len(stats) != 3 {
		t.Fatalf("got %d groups, want 3 (go, ts, other), stats=%+v", len(stats), stats)
	}
	if stats[0].Extension != "go" || stats[0].Count != 3 {
		t.Errorf("top group = %+v, want {go 3}", stats[0])
	}

	var other *LanguageStat
	for i := range stats {
		if stats[i].Extension == "other" {
			other = &stats[i]
		}
	}
	if other == nil || other.Count != 2 {
		t.Errorf("other group = %+v, want count 2 (README, LICENSE)", other)
	}
}

func TestLanguageBreakdownRollsUpBeyondLimit(t *testing.T) {
	dir := gittest.InitRepo(t)

	for _, name := range []string{"a.aa", "b.bb", "c.cc", "d.dd"} {
		gittest.WriteFile(t, dir, name, "v1")
	}
	gittest.Run(t, dir, "add", ".")
	gittest.Run(t, dir, "commit", "-m", "initial")

	stats, err := LanguageBreakdown(context.Background(), dir, 2)
	if err != nil {
		t.Fatalf("LanguageBreakdown: %v", err)
	}
	if len(stats) != 3 {
		t.Fatalf("got %d groups, want 3 (2 kept + other), stats=%+v", len(stats), stats)
	}
	total := 0
	for _, s := range stats {
		total += s.Count
	}
	if total != 4 {
		t.Errorf("total count = %d, want 4", total)
	}
}
