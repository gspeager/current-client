//go:build !nohost

package app

import (
	"context"
	"errors"
	"strconv"

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

func (s *HostService) RetryPipeline(repoPath string, pipelineID int64) ([]hostapi.Job, error) {
	return retryCI(repoPath, pipelineID, func(ctx context.Context, retrier hostapi.CIRetrier, project string, jobs []hostapi.Job) error {
		return retrier.RetryPipeline(ctx, project, pipelineID, jobs)
	})
}

// retryCI works from the jobs a click already fetched. After the retry the
// pipeline's marks read running, so History's next read asks again once
// ciRunningTTL passes, and the jobs are asked for again now.
func retryCI(repoPath string, pipelineID int64, retry func(context.Context, hostapi.CIRetrier, string, []hostapi.Job) error) ([]hostapi.Job, error) {
	_, info, err := requireHost(repoPath)
	if err != nil {
		return nil, err
	}
	retrier, err := capability[hostapi.CIRetrier](info, "this host's CI can't be re-run from the app")
	if err != nil {
		return nil, err
	}
	key := strconv.FormatInt(pipelineID, 10)
	cached, _, ok := cacheJobs.get(info.Hostname, repoPath, key)
	if !ok {
		return nil, errors.New("open the pipeline's jobs first")
	}
	ctx, cancel := hostContext()
	defer cancel()
	if err := retry(ctx, retrier, info.Project, cached); err != nil {
		return nil, err
	}
	for sha, value := range cachePipeline.entries(info.Hostname, repoPath) {
		if mark := value; mark.PipelineID == pipelineID {
			mark.Status = string(hostapi.CIRunning)
			cachePipeline.put(info.Hostname, repoPath, sha, mark)
		}
	}
	jobs, err := providerFor(info).Jobs(ctx, info.Project, pipelineID)
	if err != nil {
		cacheJobs.clear(info.Hostname, repoPath)
		return nil, err
	}
	cacheJobs.put(info.Hostname, repoPath, key, jobs)
	return jobs, nil
}
