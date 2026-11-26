//go:build !nohost

package app

import (
	"context"
	"strconv"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/hostapi"
)

// ReadinessFact is one thing that bears on whether a change can merge, from
// the cache or Git alone.
type ReadinessFact struct {
	Label string `json:"label"`
	Value string `json:"value"`
	// Level is ok, warn, info, or unknown when the fact isn't in the cache,
	// which is said rather than guessed.
	Level string `json:"level"`
}

// MergeReadiness is what the last refresh knows about merging a change, as of
// FetchedAt.
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
	_, fetchedAt, _ := cacheChanges.get(info.Hostname, repoPath, info.Project)
	facts := mapSlice(hostapi.ChangeFacts(change), func(f hostapi.Fact) ReadinessFact { return ReadinessFact(f) })
	add := func(label, value, level string) { facts = append(facts, ReadinessFact{label, value, level}) }
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
		add("Base branch", "up to date with "+change.TargetBranch, "ok")
	}
	if rules, _, ok := cacheRules.get(info.Hostname, repoPath, "all"); ok {
		if fact, ok := hostapi.ProtectionFact(rules, change.TargetBranch); ok {
			facts = append(facts, ReadinessFact(fact))
		}
	}
	return MergeReadiness{Facts: facts, FetchedAt: &fetchedAt}, nil
}

// MergeOptions is one request, for the host's merge settings, made when
// the merge dialog opens.
func (s *HostService) MergeOptions(repoPath string, number int) (hostapi.MergeOptions, error) {
	_, info, _, err := cachedChange(repoPath, number)
	if err != nil {
		return hostapi.MergeOptions{}, err
	}
	merger, err := capability[hostapi.Merger](info, "This host can't merge from the app.")
	if err != nil {
		return hostapi.MergeOptions{}, err
	}
	ctx, cancel := hostContext()
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
	merger, err := capability[hostapi.Merger](info, "This host can't merge from the app.")
	if err != nil {
		return MergeResult{}, err
	}
	ctx, cancel := hostContext()
	defer cancel()
	if err := merger.Merge(ctx, info.Project, number, hostapi.MergeRequest{Method: method, DeleteBranch: deleteBranch, HeadSHA: change.HeadSHA}); err != nil {
		return MergeResult{}, err
	}
	change.State = hostapi.ChangeMerged
	cacheMergedChange.put(info.Hostname, repoPath, strconv.Itoa(number), change)
	if cached, _, ok := cacheChanges.get(info.Hostname, repoPath, info.Project); ok {
		open := cached
		kept := open.changes[:0:0]
		for _, c := range open.changes {
			if c.Number != number {
				kept = append(kept, c)
			}
		}
		cacheChanges.put(info.Hostname, repoPath, info.Project, cachedChanges{kept, open.truncated})
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
