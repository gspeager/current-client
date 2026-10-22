//go:build !nohost

package app

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/overlap"
)

// ForecastConflict is an open change whose merge with the checked-out
// branch would conflict, with the files.
type ForecastConflict struct {
	Number       int      `json:"number"`
	Prefix       string   `json:"prefix"`
	Title        string   `json:"title"`
	Author       string   `json:"author"`
	SourceBranch string   `json:"sourceBranch"`
	Files        []string `json:"files"`
	WebURL       string   `json:"webUrl"`
}

// ForecastReport is the last refresh's forecast for the checked-out branch:
// which of the open changes targeting the same base would conflict with it.
// Checked may be less than Total when the cap cut the list.
type ForecastReport struct {
	Enabled   bool               `json:"enabled"`
	Branch    string             `json:"branch"`
	Base      string             `json:"base"`
	Checked   int                `json:"checked"`
	Total     int                `json:"total"`
	Conflicts []ForecastConflict `json:"conflicts"`
}

const (
	cacheForecast = "forecast"
	// forecastCap bounds the heads fetched and merged per refresh (decided
	// 2026-10-22), newest first.
	forecastCap = 20
)

// Forecast reads the cache only.
func (s *HostService) Forecast(repoPath string) (ForecastReport, error) {
	_, info, ok, err := primaryHost(repoPath)
	if err != nil || !ok || !info.Enabled {
		return ForecastReport{}, err
	}
	if cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheForecast, "current"); ok {
		return cached.(ForecastReport), nil
	}
	return ForecastReport{Enabled: true, Conflicts: []ForecastConflict{}}, nil
}

// refreshForecast runs at the end of a refresh: it picks the open changes
// that target the checked-out branch's base, newest first up to the cap,
// fetches their heads into refs/current-client/ in one fetch, and merges
// each with HEAD in memory. Nothing about local work goes to the host.
func refreshForecast(ctx context.Context, info HostInfo, repoPath, remote string, changes []hostapi.Change) {
	current, found, err := currentBranch(repoPath)
	if err != nil || !found {
		return
	}
	report := ForecastReport{Enabled: true, Branch: current.Name, Conflicts: []ForecastConflict{}}
	ownBranch, _ := strings.CutPrefix(current.Upstream, remote+"/")
	for _, change := range changes {
		if change.SourceProject == "" && change.SourceBranch == ownBranch && ownBranch != "" {
			report.Base = change.TargetBranch
		}
	}
	if report.Base == "" {
		if report.Base, err = git.DefaultBranch(ctx, repoPath); err != nil || report.Base == "" {
			return
		}
	}

	var candidates []hostapi.Change
	for _, change := range changes {
		if change.TargetBranch != report.Base || change.HeadSHA == "" {
			continue
		}
		if change.SourceProject == "" && change.SourceBranch == ownBranch {
			continue
		}
		candidates = append(candidates, change)
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].UpdatedAt.After(candidates[j].UpdatedAt) })
	report.Total = len(candidates)
	if len(candidates) > forecastCap {
		candidates = candidates[:forecastCap]
	}
	report.Checked = len(candidates)

	refs := map[int]string{}
	tips := make([]string, 0, len(candidates))
	for _, change := range candidates {
		refs[change.Number] = changeHeadRef(info, change.Number)
		tips = append(tips, change.HeadSHA)
	}
	if err := git.FetchHostChangeHeads(ctx, repoPath, remote, refs); err != nil {
		return
	}
	files, err := overlap.PredictTips(ctx, repoPath, tips)
	if err != nil {
		if !errors.Is(err, git.ErrMergeTreeUnsupported) {
			return
		}
		report.Checked = 0
		files = make([][]string, len(tips))
	}
	prefix := changePrefix(info)
	for i, change := range candidates {
		if len(files[i]) == 0 {
			continue
		}
		report.Conflicts = append(report.Conflicts, ForecastConflict{
			Number:       change.Number,
			Prefix:       prefix,
			Title:        change.Title,
			Author:       change.Author.Name,
			SourceBranch: change.SourceBranch,
			Files:        files[i],
			WebURL:       change.WebURL,
		})
	}
	hostClient.Cache.Put(info.Hostname, repoPath, cacheForecast, "current", report)
}
