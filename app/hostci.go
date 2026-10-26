//go:build !nohost

package app

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/hostapi"
)

// CIMark is what a commit or branch tip shows for its pipeline. Status is
// empty when the host has no pipeline for the commit.
type CIMark struct {
	Status     string `json:"status"`
	PipelineID int64  `json:"pipelineId"`
	WebURL     string `json:"webUrl"`
}

const (
	cachePipeline = "pipeline"
	cacheJobs     = "jobs"
	// ciBatchCap bounds one call's fetching; ciParallel how many of those
	// run at once against the host.
	ciBatchCap = 50
	ciParallel = 4
	// ciTipsCap bounds the branch tips a refresh asks about.
	ciTipsCap = 25
)

// ciRunningTTL is how long a running or pending result stands before a read
// that fetches asks for it again: it will change, unlike a finished one.
var ciRunningTTL = 2 * time.Minute

// CIStatus is the cached pipeline result for each SHA that has one; it
// never contacts the host.
func (s *HostService) CIStatus(repoPath string, shas []string) (map[string]CIMark, error) {
	_, info, ok, err := primaryHost(repoPath)
	if err != nil || !ok || !info.Enabled {
		return map[string]CIMark{}, err
	}
	return cachedCI(info, repoPath, shas), nil
}

// FetchCIStatus is CIStatus that first asks the host, in parallel and up to
// the batch cap, for the SHAs it has no answer for yet. A commit without a
// pipeline is remembered too, so no SHA is asked for twice until a refresh.
func (s *HostService) FetchCIStatus(repoPath string, shas []string) (map[string]CIMark, error) {
	_, info, ok, err := primaryHost(repoPath)
	if err != nil || !ok || !info.Enabled {
		return map[string]CIMark{}, err
	}
	provider := providerFor(info)
	if provider == nil {
		return map[string]CIMark{}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fetchPipelines(ctx, provider, info, repoPath, unknownPipelines(info, repoPath, shas, ciBatchCap))
	return cachedCI(info, repoPath, shas), nil
}

// unknownPipelines is the SHAs worth asking about: never answered, or
// answered running or pending longer than ciRunningTTL ago.
func unknownPipelines(info HostInfo, repoPath string, shas []string, limit int) []string {
	var missing []string
	for _, sha := range shas {
		cached, at, ok := hostClient.Cache.Get(info.Hostname, repoPath, cachePipeline, sha)
		if ok {
			status := cached.(CIMark).Status
			if (status != string(hostapi.CIRunning) && status != string(hostapi.CIPending)) || time.Since(at) < ciRunningTTL {
				continue
			}
		}
		missing = append(missing, sha)
	}
	if len(missing) > limit {
		missing = missing[:limit]
	}
	return missing
}

// refreshBranchTipCI asks for the local branch tips' pipelines on a
// refresh, so the sidebar's marks follow a pipeline to its end without
// History having to scroll past the commit.
func refreshBranchTipCI(ctx context.Context, provider hostapi.Provider, info HostInfo, repoPath string) {
	branches, err := git.ListBranches(ctx, repoPath)
	if err != nil {
		return
	}
	var tips []string
	for _, b := range branches {
		if b.Tip != "" {
			tips = append(tips, b.Tip)
		}
	}
	fetchPipelines(ctx, provider, info, repoPath, unknownPipelines(info, repoPath, tips, ciTipsCap))
}

// PipelineJobs lists a pipeline's jobs, asked for once per pipeline until a
// refresh.
func (s *HostService) PipelineJobs(repoPath string, pipelineID int64) ([]hostapi.Job, error) {
	_, info, ok, err := primaryHost(repoPath)
	if err != nil || !ok || !info.Enabled {
		return nil, err
	}
	return cachedOrFetched(info, repoPath, cacheJobs, strconv.FormatInt(pipelineID, 10), func(ctx context.Context, provider hostapi.Provider) ([]hostapi.Job, error) {
		return provider.Jobs(ctx, info.Project, pipelineID)
	})
}

func cachedCI(info HostInfo, repoPath string, shas []string) map[string]CIMark {
	marks := map[string]CIMark{}
	for _, sha := range shas {
		if cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cachePipeline, sha); ok {
			if mark := cached.(CIMark); mark.Status != "" {
				marks[sha] = mark
			}
		}
	}
	return marks
}

// fetchPipelines asks for each SHA's latest pipeline, ciParallel at a time,
// and caches every answer including "none". A failed request is left
// uncached, so the next batch tries it again.
func fetchPipelines(ctx context.Context, provider hostapi.Provider, info HostInfo, repoPath string, shas []string) {
	inParallel(len(shas), func(i int) {
		pipeline, ok, err := provider.LatestPipeline(ctx, info.Project, shas[i])
		if err != nil {
			return
		}
		mark := CIMark{}
		if ok {
			mark = CIMark{Status: string(pipeline.Status), PipelineID: pipeline.ID, WebURL: pipeline.WebURL}
		}
		hostClient.Cache.Put(info.Hostname, repoPath, cachePipeline, shas[i], mark)
	})
}

// inParallel runs request for 0..count-1, ciParallel at a time, and waits
// for all of them.
func inParallel(count int, request func(i int)) {
	var wg sync.WaitGroup
	slots := make(chan struct{}, ciParallel)
	for i := range count {
		wg.Add(1)
		slots <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-slots }()
			request(i)
		}()
	}
	wg.Wait()
}

// fillChangeCI gives each open change the CI result of its head, which the
// list endpoint doesn't carry, through the same per-SHA cache.
func fillChangeCI(ctx context.Context, provider hostapi.Provider, info HostInfo, repoPath string, changes []hostapi.Change) {
	var shas []string
	for _, change := range changes {
		if change.HeadSHA != "" && change.CI == hostapi.CIUnknown {
			shas = append(shas, change.HeadSHA)
		}
	}
	if len(shas) > ciBatchCap {
		shas = shas[:ciBatchCap]
	}
	fetchPipelines(ctx, provider, info, repoPath, shas)
	for i := range changes {
		if cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cachePipeline, changes[i].HeadSHA); ok {
			changes[i].CI = hostapi.CIStatus(cached.(CIMark).Status)
		}
	}
}

// fillChangeReviews asks for each open change's reviews, up to the batch
// cap. A change whose request fails keeps the requested reviewers its list
// entry carried.
func fillChangeReviews(ctx context.Context, provider hostapi.Provider, info HostInfo, changes []hostapi.Change) {
	inParallel(min(len(changes), ciBatchCap), func(i int) {
		if reviews, err := provider.Reviews(ctx, info.Project, changes[i]); err == nil {
			changes[i].Reviews = reviews
		}
	})
}
