package hostapi

import (
	"sort"
)

// ForecastCap bounds the open changes a forecast checks, newest first.
const ForecastCap = 20

// ForecastCandidates are the open changes a branch's forecast checks: those
// targeting base, other than the branch's own, newest first up to the cap.
// total counts them before the cap.
func ForecastCandidates(changes []Change, base, ownBranch string) (candidates []Change, total int) {
	for _, change := range changes {
		if change.TargetBranch != base || change.HeadSHA == "" {
			continue
		}
		if change.SourceProject == "" && change.SourceBranch == ownBranch {
			continue
		}
		candidates = append(candidates, change)
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].UpdatedAt.After(candidates[j].UpdatedAt) })
	return candidates[:min(len(candidates), ForecastCap)], len(candidates)
}

// ForecastBase is the branch its own open change targets, "" when it has none.
func ForecastBase(changes []Change, ownBranch string) string {
	base := ""
	for _, change := range changes {
		if change.SourceProject == "" && change.SourceBranch == ownBranch && ownBranch != "" {
			base = change.TargetBranch
		}
	}
	return base
}
