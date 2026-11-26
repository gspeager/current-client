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
	Branch   string `json:"branch"`
	Upstream string `json:"upstream"`
	Enabled  bool   `json:"enabled"`
	// Proposable is false on the default branch and on a branch with no
	// commits beyond it, where there is nothing to open a change for.
	Proposable bool `json:"proposable"`
	// Checked is false until the host's list has been read, when "no change"
	// isn't known yet.
	Checked bool `json:"checked"`
	// Kind is the host's word, "pull request" or "merge request".
	Kind       string        `json:"kind"`
	Change     *BranchChange `json:"change"`
	CI         *CIMark       `json:"ci"`
	FailedJobs []string      `json:"failedJobs"`
}

func (s *HostService) CurrentChange(repoPath string) (CurrentChange, error) {
	return currentChange(repoPath)
}

func currentChange(repoPath string) (CurrentChange, error) {
	_, info, ok, err := enabledHost(repoPath)
	if err != nil {
		return CurrentChange{}, err
	}
	current, found, err := currentBranch(repoPath)
	if err != nil || !found {
		return CurrentChange{}, err
	}
	result := CurrentChange{Branch: current.Name, Upstream: current.Upstream, Enabled: ok && providerFor(info) != nil, Kind: changeKind(info)}
	if !result.Enabled {
		return result, nil
	}
	_, _, result.Checked = cacheChanges.get(info.Hostname, repoPath, info.Project)
	var change BranchChange
	if current.Upstream != "" {
		changes, err := branchChanges(repoPath)
		if err != nil {
			return CurrentChange{}, err
		}
		change = changes[current.Upstream]
	}
	if change.State == "" {
		result.Proposable = proposable(repoPath, current.Name)
		return result, nil
	}
	result.Change = &change
	if cached, _, ok := cachePipeline.get(info.Hostname, repoPath, change.HeadSHA); ok {
		if mark := cached; mark.Status != "" {
			result.CI = &mark
			if jobs, _, ok := cacheJobs.get(info.Hostname, repoPath, strconv.FormatInt(mark.PipelineID, 10)); ok {
				for _, job := range jobs {
					if job.Status == hostapi.CIFailed {
						result.FailedJobs = append(result.FailedJobs, job.Name)
					}
				}
			}
		}
	}
	return result, nil
}

// proposable is true when the default branch is unknown, so the section
// still offers a change rather than guessing there is nothing to propose.
func proposable(repoPath, branch string) bool {
	ctx := context.Background()
	base, err := git.DefaultBranch(ctx, repoPath)
	if err != nil || base == "" {
		return true
	}
	if base == branch {
		return false
	}
	ahead, err := git.CountCommits(ctx, repoPath, "refs/heads/"+base+"..HEAD")
	return err != nil || ahead > 0
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
		cached, _, ok := cachePipeline.get(info.Hostname, repoPath, change.HeadSHA)
		if !ok {
			return
		}
		mark := cached
		if jobs, err := provider.Jobs(ctx, info.Project, mark.PipelineID); err == nil {
			cacheJobs.put(info.Hostname, repoPath, strconv.FormatInt(mark.PipelineID, 10), jobs)
		}
		return
	}
}
