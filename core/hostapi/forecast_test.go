package hostapi

import (
	"testing"
	"time"
)

func TestForecastCandidates(t *testing.T) {
	now := time.Now()
	changes := []Change{
		{Number: 1, SourceBranch: "mine", TargetBranch: "main", HeadSHA: "a", UpdatedAt: now},
		{Number: 2, SourceBranch: "old", TargetBranch: "main", HeadSHA: "b", UpdatedAt: now.Add(-time.Hour)},
		{Number: 3, SourceBranch: "new", TargetBranch: "main", HeadSHA: "c", UpdatedAt: now},
		{Number: 4, SourceBranch: "other", TargetBranch: "develop", HeadSHA: "d", UpdatedAt: now},
		{Number: 5, SourceBranch: "unfetched", TargetBranch: "main", UpdatedAt: now},
	}
	if base := ForecastBase(changes, "mine"); base != "main" {
		t.Fatalf("base = %q", base)
	}
	candidates, total := ForecastCandidates(changes, "main", "mine")
	if total != 2 || len(candidates) != 2 || candidates[0].Number != 3 || candidates[1].Number != 2 {
		t.Fatalf("candidates = %+v, total %d", candidates, total)
	}
}
