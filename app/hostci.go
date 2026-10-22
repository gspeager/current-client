//go:build !nohost

package app

import (
	"context"
	"strconv"
	"sync"
	"time"

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
)

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
	var missing []string
	for _, sha := range shas {
		if _, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cachePipeline, sha); !ok {
			missing = append(missing, sha)
		}
	}
	if len(missing) > ciBatchCap {
		missing = missing[:ciBatchCap]
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	fetchPipelines(ctx, provider, info, repoPath, missing)
	return cachedCI(info, repoPath, shas), nil
}

// PipelineJobs lists a pipeline's jobs, asked for once per pipeline until a
// refresh.
func (s *HostService) PipelineJobs(repoPath string, pipelineID int64) ([]hostapi.Job, error) {
	_, info, ok, err := primaryHost(repoPath)
	if err != nil || !ok || !info.Enabled {
		return nil, err
	}
	provider := providerFor(info)
	if provider == nil {
		return nil, nil
	}
	key := strconv.FormatInt(pipelineID, 10)
	if cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheJobs, key); ok {
		return cached.([]hostapi.Job), nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	jobs, err := provider.Jobs(ctx, info.Project, pipelineID)
	if err != nil {
		return nil, err
	}
	hostClient.Cache.Put(info.Hostname, repoPath, cacheJobs, key, jobs)
	return jobs, nil
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
	var wg sync.WaitGroup
	slots := make(chan struct{}, ciParallel)
	for _, sha := range shas {
		wg.Add(1)
		slots <- struct{}{}
		go func(sha string) {
			defer wg.Done()
			defer func() { <-slots }()
			pipeline, ok, err := provider.LatestPipeline(ctx, info.Project, sha)
			if err != nil {
				return
			}
			mark := CIMark{}
			if ok {
				mark = CIMark{Status: string(pipeline.Status), PipelineID: pipeline.ID, WebURL: pipeline.WebURL}
			}
			hostClient.Cache.Put(info.Hostname, repoPath, cachePipeline, sha, mark)
		}(sha)
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
