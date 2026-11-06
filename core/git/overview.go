package git

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"
)

// OverviewDay is a day's commits among those History lists. FirstRow is the
// newest one's row, which is where a jump to this day lands. Rows follow graph
// order, which dates needn't, so days are sorted by date but their rows aren't.
type OverviewDay struct {
	Day      time.Time
	Commits  int
	FirstRow int
}

// Marker kinds beside RefKind's: HEAD, and a stash, which History doesn't list.
const (
	OverviewHead   = "head"
	OverviewBranch = "branch"
	OverviewTag    = "tag"
	OverviewStash  = "stash"
)

type OverviewMarker struct {
	Kind string
	Name string
	At   time.Time
}

type HistoryOverview struct {
	Days    []OverviewDay
	Markers []OverviewMarker
}

// Overview describes the same commits History lists under filter, all of them
// rather than a page. Only refs on one of those commits get a marker.
func Overview(ctx context.Context, repoPath string, filter HistoryFilter) (HistoryOverview, error) {
	key := "overview\x00" + strings.Join([]string{filter.Ref, filter.Since, filter.Until, filter.Author, filter.Type, strconv.FormatBool(filter.FirstParent)}, "\x00")
	return cachedByRefs(ctx, repoPath, key, func() (HistoryOverview, error) {
		commits, err := overviewCommits(ctx, repoPath, filter)
		if err != nil {
			if hasNoCommits(ctx, repoPath) {
				return HistoryOverview{}, nil
			}
			return HistoryOverview{}, err
		}
		overview := HistoryOverview{Days: overviewDays(commits)}
		overview.Markers, err = overviewMarkers(ctx, repoPath, commits)
		return overview, err
	})
}

type overviewCommit struct {
	sha string
	at  time.Time
}

func overviewCommits(ctx context.Context, repoPath string, filter HistoryFilter) ([]overviewCommit, error) {
	if filter.Type != "" {
		entries, err := History(ctx, repoPath, 0, 0, filter)
		if err != nil {
			return nil, err
		}
		commits := make([]overviewCommit, len(entries))
		for i, entry := range entries {
			commits[i] = overviewCommit{entry.SHA, entry.Date}
		}
		return commits, nil
	}
	result, err := runResult(ctx, repoPath, append([]string{"log", "--format=%H %at"}, filterArgs(filter)...)...)
	if err != nil {
		return nil, err
	}
	var commits []overviewCommit
	for _, line := range strings.Split(result.Stdout, "\n") {
		sha, at, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		seconds, err := strconv.ParseInt(at, 10, 64)
		if err != nil {
			continue
		}
		commits = append(commits, overviewCommit{sha, time.Unix(seconds, 0)})
	}
	return commits, nil
}

func overviewDays(commits []overviewCommit) []OverviewDay {
	byDay := map[string]*OverviewDay{}
	for row, commit := range commits {
		local := commit.at.Local()
		key := local.Format("2006-01-02")
		day, seen := byDay[key]
		if !seen {
			day = &OverviewDay{
				Day:      time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location()),
				FirstRow: row,
			}
			byDay[key] = day
		}
		day.Commits++
	}
	days := make([]OverviewDay, 0, len(byDay))
	for _, day := range byDay {
		days = append(days, *day)
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Day.Before(days[j].Day) })
	return days
}

func overviewMarkers(ctx context.Context, repoPath string, commits []overviewCommit) ([]OverviewMarker, error) {
	listed := make(map[string]time.Time, len(commits))
	for _, commit := range commits {
		listed[commit.sha] = commit.at
	}
	var markers []OverviewMarker
	if head, err := ResolveCommit(ctx, repoPath, "HEAD"); err == nil {
		if at, ok := listed[head]; ok {
			markers = append(markers, OverviewMarker{OverviewHead, "HEAD", at})
		}
	}
	result, err := runResult(ctx, repoPath, "for-each-ref", "refs/heads", "refs/tags", "refs/stash",
		"--format=%(refname)%00%(objectname)%00%(*objectname)%00%(authordate:unix)")
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(result.Stdout, "\n") {
		fields := strings.Split(line, "\x00")
		if len(fields) != 4 {
			continue
		}
		ref, sha := fields[0], fields[1]
		if fields[2] != "" {
			sha = fields[2]
		}
		switch {
		case ref == "refs/stash":
			if seconds, err := strconv.ParseInt(fields[3], 10, 64); err == nil {
				markers = append(markers, OverviewMarker{OverviewStash, "stash", time.Unix(seconds, 0)})
			}
		case strings.HasPrefix(ref, "refs/heads/"):
			if at, ok := listed[sha]; ok {
				markers = append(markers, OverviewMarker{OverviewBranch, strings.TrimPrefix(ref, "refs/heads/"), at})
			}
		case strings.HasPrefix(ref, "refs/tags/"):
			if at, ok := listed[sha]; ok {
				markers = append(markers, OverviewMarker{OverviewTag, strings.TrimPrefix(ref, "refs/tags/"), at})
			}
		}
	}
	return markers, nil
}
