//go:build !nohost

package app

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/hostapi"
)

// ReadinessFact is one thing that bears on whether a change can merge, from
// the cache or Git alone. Level is ok, warn, info, or unknown when the fact
// isn't in the cache, which is said rather than guessed.
type ReadinessFact struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Level string `json:"level"`
}

// MergeReadiness is what the last refresh knows about merging a change;
// FetchedAt is that refresh's time.
type MergeReadiness struct {
	Facts     []ReadinessFact `json:"facts"`
	FetchedAt *time.Time      `json:"fetchedAt"`
}

// MergeResult says what was merged and whether a local branch tracks the
// source branch, so the branch clean-up can offer to delete it.
type MergeResult struct {
	Number       int    `json:"number"`
	SourceBranch string `json:"sourceBranch"`
	LocalBranch  string `json:"localBranch"`
}

// MergeReadiness reads the cache and Git alone.
func (s *HostService) MergeReadiness(repoPath string, number int) (MergeReadiness, error) {
	remote, info, change, err := cachedChange(repoPath, number)
	if err != nil {
		return MergeReadiness{}, err
	}
	_, fetchedAt, _ := hostClient.Cache.Get(info.Hostname, repoPath, cacheChanges, info.Project)
	facts := []ReadinessFact{}
	add := func(label, value, level string) { facts = append(facts, ReadinessFact{label, value, level}) }
	if change.Draft {
		add("Draft", "marked as a draft", "warn")
	}
	switch change.CI {
	case hostapi.CISuccess:
		add("Checks", "passed", "ok")
	case hostapi.CIFailed:
		add("Checks", "failed", "warn")
	case hostapi.CIRunning, hostapi.CIPending:
		add("Checks", "still running", "warn")
	case "":
		add("Checks", "no result in the cache", "unknown")
	default:
		add("Checks", string(change.CI), "info")
	}
	approved, requested, changes := reviewNames(change.Reviews)
	switch {
	case len(changes) > 0:
		add("Reviews", "changes requested by "+joinNames(changes), "warn")
	case len(requested) > 0 && len(approved) > 0:
		add("Reviews", "approved by "+joinNames(approved)+"; still waiting on "+joinNames(requested), "warn")
	case len(requested) > 0:
		add("Reviews", "waiting on "+joinNames(requested), "warn")
	case len(approved) > 0:
		add("Reviews", "approved by "+joinNames(approved), "ok")
	default:
		add("Reviews", "none", "info")
	}
	if change.UnresolvedThreads > 0 {
		add("Threads", strconv.Itoa(change.UnresolvedThreads)+" unresolved", "warn")
	}
	switch {
	case !change.ConflictsKnown:
		add("Conflicts", "not known until the change is opened on the host", "unknown")
	case change.HasConflicts:
		add("Conflicts", "conflicts with "+change.TargetBranch, "warn")
	default:
		add("Conflicts", "none", "ok")
	}
	ctx := context.Background()
	target := remote.Name + "/" + change.TargetBranch
	if _, err := git.ResolveCommit(ctx, repoPath, change.HeadSHA); err != nil {
		add("Behind "+change.TargetBranch, "head not fetched; check it out to know", "unknown")
	} else if behind, err := git.CountCommits(ctx, repoPath, change.HeadSHA+".."+target); err != nil {
		add("Behind "+change.TargetBranch, target+" not fetched", "unknown")
	} else if behind == 1 {
		add("Behind "+change.TargetBranch, "1 commit, as of the last fetch", "warn")
	} else if behind > 1 {
		add("Behind "+change.TargetBranch, strconv.Itoa(behind)+" commits, as of the last fetch", "warn")
	} else {
		add("Behind "+change.TargetBranch, "up to date, as of the last fetch", "ok")
	}
	if cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheRules, "all"); ok {
		if rule, ok := hostapi.MatchRule(cached.([]hostapi.BranchRule), change.TargetBranch); ok {
			value := "protected; the host applies its rules on merge"
			if rule.Pattern != change.TargetBranch {
				value += " (rule " + rule.Pattern + ")"
			}
			add("Protection", value, "info")
		}
	}
	return MergeReadiness{Facts: facts, FetchedAt: &fetchedAt}, nil
}

func reviewNames(reviews []hostapi.Review) (approved, requested, changes []string) {
	for _, r := range reviews {
		switch r.State {
		case hostapi.ReviewApproved:
			approved = append(approved, r.Reviewer.Name)
		case hostapi.ReviewRequested:
			requested = append(requested, r.Reviewer.Name)
		case hostapi.ReviewChangesRequested:
			changes = append(changes, r.Reviewer.Name)
		}
	}
	return approved, requested, changes
}

func joinNames(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out
}

// MergeOptions is one request, for the host's merge settings, made when
// the merge dialog opens.
func (s *HostService) MergeOptions(repoPath string, number int) (hostapi.MergeOptions, error) {
	_, info, _, err := cachedChange(repoPath, number)
	if err != nil {
		return hostapi.MergeOptions{}, err
	}
	merger, ok := providerFor(info).(hostapi.Merger)
	if !ok {
		return hostapi.MergeOptions{}, errors.New("this host can't merge from the app")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return merger.MergeOptions(ctx, info.Project)
}

// MergeChange merges on the host, for the head the last refresh saw, then
// moves the change from the open list to the merged ones in the cache so
// the list and the branch clean-up follow without a refresh.
func (s *HostService) MergeChange(repoPath string, number int, method string, deleteBranch bool) (MergeResult, error) {
	remote, info, change, err := cachedChange(repoPath, number)
	if err != nil {
		return MergeResult{}, err
	}
	merger, ok := providerFor(info).(hostapi.Merger)
	if !ok {
		return MergeResult{}, errors.New("this host can't merge from the app")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := merger.Merge(ctx, info.Project, number, hostapi.MergeRequest{Method: method, DeleteBranch: deleteBranch, HeadSHA: change.HeadSHA}); err != nil {
		return MergeResult{}, err
	}
	change.State = hostapi.ChangeMerged
	hostClient.Cache.Put(info.Hostname, repoPath, cacheMergedChange, strconv.Itoa(number), change)
	if cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheChanges, info.Project); ok {
		open := cached.(cachedChanges)
		kept := open.changes[:0:0]
		for _, c := range open.changes {
			if c.Number != number {
				kept = append(kept, c)
			}
		}
		hostClient.Cache.Put(info.Hostname, repoPath, cacheChanges, info.Project, cachedChanges{kept, open.truncated})
	}
	result := MergeResult{Number: number, SourceBranch: change.SourceBranch}
	if change.SourceProject == "" {
		if branches, err := git.ListBranches(ctx, repoPath); err == nil {
			for _, b := range branches {
				if name, ok := cutRemote(b.Upstream, remote.Name); ok && name == change.SourceBranch {
					result.LocalBranch = b.Name
				}
			}
		}
	}
	return result, nil
}
