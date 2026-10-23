//go:build !nohost

package app

import (
	"context"
	"strconv"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/hostapi"
)

// CurrentChange is the checked-out branch's merge request / pull request,
// for the section at the top of the Branches sidebar. Change is nil when
// the branch has none that the cache knows; Enabled false hides the section.
type CurrentChange struct {
	Branch     string        `json:"branch"`
	Upstream   string        `json:"upstream"`
	Enabled    bool          `json:"enabled"`
	Change     *BranchChange `json:"change"`
	CI         *CIMark       `json:"ci"`
	FailedJobs []string      `json:"failedJobs"`
}

// CurrentChange reads the cache only.
func (s *HostService) CurrentChange(repoPath string) (CurrentChange, error) {
	_, info, ok, err := primaryHost(repoPath)
	if err != nil {
		return CurrentChange{}, err
	}
	current, found, err := currentBranch(repoPath)
	if err != nil || !found {
		return CurrentChange{}, err
	}
	result := CurrentChange{Branch: current.Name, Upstream: current.Upstream, Enabled: ok && info.Enabled && providerFor(info) != nil}
	if !result.Enabled || current.Upstream == "" {
		return result, nil
	}
	changes, err := s.BranchChanges(repoPath)
	if err != nil {
		return CurrentChange{}, err
	}
	change, ok := changes[current.Upstream]
	if !ok || change.State == "" {
		return result, nil
	}
	result.Change = &change
	if cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cachePipeline, change.HeadSHA); ok {
		if mark := cached.(CIMark); mark.Status != "" {
			result.CI = &mark
			if jobs, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheJobs, strconv.FormatInt(mark.PipelineID, 10)); ok {
				for _, job := range jobs.([]hostapi.Job) {
					if job.Status == hostapi.CIFailed {
						result.FailedJobs = append(result.FailedJobs, job.Name)
					}
				}
			}
		}
	}
	return result, nil
}

func currentBranch(repoPath string) (git.Branch, bool, error) {
	branches, err := git.ListBranches(context.Background(), repoPath)
	if err != nil {
		return git.Branch{}, false, err
	}
	for _, b := range branches {
		if b.Current {
			return b, true, nil
		}
	}
	return git.Branch{}, false, nil
}

// currentRemoteBranch is the checked-out branch's name on remote, when its
// upstream is there.
func currentRemoteBranch(repoPath, remote string) (string, bool) {
	current, found, err := currentBranch(repoPath)
	if err != nil || !found {
		return "", false
	}
	return cutRemote(current.Upstream, remote)
}

// fetchCurrentBranchJobs makes a refresh also bring the failed jobs of the
// checked-out branch's change, so the sidebar section can name them without
// a click.
func fetchCurrentBranchJobs(ctx context.Context, provider hostapi.Provider, info HostInfo, repoPath, remote string, changes []hostapi.Change) {
	branch, ok := currentRemoteBranch(repoPath, remote)
	if !ok {
		return
	}
	for _, change := range changes {
		if change.SourceBranch != branch || change.SourceProject != "" || change.CI != hostapi.CIFailed {
			continue
		}
		cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cachePipeline, change.HeadSHA)
		if !ok {
			return
		}
		mark := cached.(CIMark)
		if jobs, err := provider.Jobs(ctx, info.Project, mark.PipelineID); err == nil {
			hostClient.Cache.Put(info.Hostname, repoPath, cacheJobs, strconv.FormatInt(mark.PipelineID, 10), jobs)
		}
		return
	}
}
