//go:build !nohost

package app

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/gspeager/current-client/core/hostapi"
)

// RetryJob runs one failed job again: one request on a click, then the
// pipeline's jobs are read again so the list shows it queued.
func (s *HostService) RetryJob(repoPath string, pipelineID, jobID int64) ([]hostapi.Job, error) {
	return retryCI(repoPath, pipelineID, func(ctx context.Context, retrier hostapi.CIRetrier, project string, jobs []hostapi.Job) error {
		for _, job := range jobs {
			if job.ID == jobID {
				return retrier.RetryJob(ctx, project, job)
			}
		}
		return errors.New("that job isn't in the pipeline's last job list; open the checks again")
	})
}

// RetryPipeline runs every failed job of a pipeline again.
func (s *HostService) RetryPipeline(repoPath string, pipelineID int64) ([]hostapi.Job, error) {
	return retryCI(repoPath, pipelineID, func(ctx context.Context, retrier hostapi.CIRetrier, project string, jobs []hostapi.Job) error {
		return retrier.RetryPipeline(ctx, project, pipelineID, jobs)
	})
}

// retryCI works from the jobs a click already fetched. After the retry the
// pipeline's marks read running, so History's next read asks again once
// ciRunningTTL passes, and the jobs are asked for again now.
func retryCI(repoPath string, pipelineID int64, retry func(context.Context, hostapi.CIRetrier, string, []hostapi.Job) error) ([]hostapi.Job, error) {
	_, info, ok, err := primaryHost(repoPath)
	if err != nil {
		return nil, err
	}
	if !ok || !info.Enabled {
		return nil, errors.New("integration is off for this repository's host")
	}
	retrier, ok := providerFor(info).(hostapi.CIRetrier)
	if !ok {
		return nil, errors.New("this host's CI can't be re-run from the app")
	}
	key := strconv.FormatInt(pipelineID, 10)
	cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheJobs, key)
	if !ok {
		return nil, errors.New("open the pipeline's jobs first")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := retry(ctx, retrier, info.Project, cached.([]hostapi.Job)); err != nil {
		return nil, err
	}
	for sha, value := range hostClient.Cache.Entries(info.Hostname, repoPath, cachePipeline) {
		if mark := value.(CIMark); mark.PipelineID == pipelineID {
			mark.Status = string(hostapi.CIRunning)
			hostClient.Cache.Put(info.Hostname, repoPath, cachePipeline, sha, mark)
		}
	}
	jobs, err := providerFor(info).Jobs(ctx, info.Project, pipelineID)
	if err != nil {
		hostClient.Cache.ClearKind(info.Hostname, repoPath, cacheJobs)
		return nil, err
	}
	hostClient.Cache.Put(info.Hostname, repoPath, cacheJobs, key, jobs)
	return jobs, nil
}
