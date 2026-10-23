//go:build !nohost

package app

import (
	"context"
	"strconv"
	"time"

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostcontent"
)

type ReviewNote struct {
	Author    string `json:"author"`
	BodyHTML  string `json:"bodyHtml"`
	CreatedAt string `json:"createdAt"`
}

// ReviewThread is a review discussion on the checked-out branch's change,
// with its notes rendered through the one sanitizing renderer. Line is the
// new-side line it was left on, 0 for a thread on the change as a whole.
type ReviewThread struct {
	ID       string       `json:"id"`
	Path     string       `json:"path"`
	Line     int          `json:"line"`
	OldLine  int          `json:"oldLine"`
	Resolved bool         `json:"resolved"`
	Outdated bool         `json:"outdated"`
	Notes    []ReviewNote `json:"notes"`
}

// ReviewThreads is what the diff views and file lists read: the threads of
// the checked-out branch's change, and the unresolved, still-current count
// per file. Number is 0 when the branch has no change the cache knows.
type ReviewThreads struct {
	Enabled    bool           `json:"enabled"`
	Number     int            `json:"number"`
	Prefix     string         `json:"prefix"`
	Threads    []ReviewThread `json:"threads"`
	Unresolved map[string]int `json:"unresolved"`
	Total      int            `json:"total"`
}

const cacheThreads = "threads"

// ReviewThreads reads the cache only.
func (s *HostService) ReviewThreads(repoPath string) (ReviewThreads, error) {
	current, err := s.CurrentChange(repoPath)
	if err != nil || !current.Enabled {
		return ReviewThreads{}, err
	}
	_, info, _, _ := primaryHost(repoPath)
	if current.Change == nil {
		return renderThreads(info, 0, nil), nil
	}
	var threads []hostapi.Thread
	if cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheThreads, strconv.Itoa(current.Change.Number)); ok {
		threads = cached.([]hostapi.Thread)
	}
	return renderThreads(info, current.Change.Number, threads), nil
}

// ChangeThreads is any change's threads, for the detail's Threads tab:
// asked of the host once per change until a refresh.
func (s *HostService) ChangeThreads(repoPath string, number int) (ReviewThreads, error) {
	_, info, change, err := cachedChange(repoPath, number)
	if err != nil {
		return ReviewThreads{}, err
	}
	threads, err := cachedOrFetched(info, repoPath, cacheThreads, strconv.Itoa(change.Number), func(ctx context.Context, provider hostapi.Provider) ([]hostapi.Thread, error) {
		return provider.Threads(ctx, info.Project, change.Number, change.HeadSHA)
	})
	if err != nil {
		return ReviewThreads{}, err
	}
	return renderThreads(info, change.Number, threads), nil
}

func renderThreads(info HostInfo, number int, threads []hostapi.Thread) ReviewThreads {
	result := ReviewThreads{Enabled: true, Number: number, Prefix: changePrefix(info), Threads: []ReviewThread{}, Unresolved: map[string]int{}}
	webRoot := "https://" + info.Hostname + "/"
	for _, thread := range threads {
		rendered := ReviewThread{ID: thread.ID, Path: thread.Path, Line: thread.Line, OldLine: thread.OldLine, Resolved: thread.Resolved, Outdated: thread.Outdated}
		for _, note := range thread.Notes {
			rendered.Notes = append(rendered.Notes, ReviewNote{
				Author:    note.Author.Name,
				BodyHTML:  hostcontent.RenderMarkdown(note.Body, webRoot),
				CreatedAt: note.CreatedAt.UTC().Format(time.RFC3339),
			})
		}
		result.Threads = append(result.Threads, rendered)
		if !thread.Resolved {
			result.Total++
			if !thread.Outdated && thread.Path != "" {
				result.Unresolved[thread.Path]++
			}
		}
	}
	return result
}

// fillCurrentBranchThreads fetches the review threads of the checked-out
// branch's change on refresh, caches them, and writes the unresolved count
// into the change so markers and the sidebar show it.
func fillCurrentBranchThreads(ctx context.Context, provider hostapi.Provider, info HostInfo, repoPath, remote string, changes []hostapi.Change) {
	branch, ok := currentRemoteBranch(repoPath, remote)
	if !ok {
		return
	}
	for i := range changes {
		change := &changes[i]
		if change.SourceBranch != branch || change.SourceProject != "" {
			continue
		}
		threads, err := provider.Threads(ctx, info.Project, change.Number, change.HeadSHA)
		if err != nil {
			return
		}
		hostClient.Cache.Put(info.Hostname, repoPath, cacheThreads, strconv.Itoa(change.Number), threads)
		change.UnresolvedThreads = 0
		for _, thread := range threads {
			if !thread.Resolved {
				change.UnresolvedThreads++
			}
		}
		return
	}
}
