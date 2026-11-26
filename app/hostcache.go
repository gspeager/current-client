//go:build !nohost

package app

import (
	"context"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/hostapi"
)

// A cacheKind is one kind of host answer and the one type it holds, so a read
// can't take another type. Reading the cache never makes a request: what's in
// it came from a fetch or a refresh.
type cacheKind[T any] string

func (k cacheKind[T]) get(host, repo, id string) (T, time.Time, bool) {
	value, at, ok := hostClient.Cache.Get(host, repo, string(k), id)
	typed, isT := value.(T)
	return typed, at, ok && isT
}

func (k cacheKind[T]) put(host, repo, id string, value T) {
	hostClient.Cache.Put(host, repo, string(k), id, value)
}

func (k cacheKind[T]) entries(host, repo string) map[string]T {
	typed := map[string]T{}
	for id, value := range hostClient.Cache.Entries(host, repo, string(k)) {
		if v, ok := value.(T); ok {
			typed[id] = v
		}
	}
	return typed
}

func (k cacheKind[T]) clear(host, repo string) {
	hostClient.Cache.ClearKind(host, repo, string(k))
}

type clearable interface{ clear(host, repo string) }

func clearKinds(host, repo string, kinds ...clearable) {
	for _, kind := range kinds {
		kind.clear(host, repo)
	}
}

var (
	cacheRemoteRefs = cacheKind[git.RemoteChanges]("ls-remote")
	cacheChanges    = cacheKind[cachedChanges]("changes")
	cacheClosed     = cacheKind[cachedChanges]("closed")
	cacheUser       = cacheKind[hostapi.User]("user")
	cacheFiles      = cacheKind[[]hostapi.ChangeFile]("files")
	cacheCommits    = cacheKind[[]hostapi.ChangeCommit]("commits")
	cacheThreads    = cacheKind[[]hostapi.Thread]("threads")

	cachePipeline = cacheKind[CIMark]("pipeline")
	cacheJobs     = cacheKind[[]hostapi.Job]("jobs")
	cacheJobLog   = cacheKind[JobLog]("joblog")
	cacheRules    = cacheKind[[]hostapi.BranchRule]("rules")

	cacheIssue         = cacheKind[IssueInfo]("issue")
	cacheAssigned      = cacheKind[[]hostapi.Issue]("assigned")
	cacheIssueDetails  = cacheKind[hostapi.IssueDetails]("issueDetails")
	cacheIssueComments = cacheKind[[]hostapi.Note]("issueComments")
	cacheProjectIssues = cacheKind[[]hostapi.Issue]("projectIssues")

	cacheForecast     = cacheKind[ForecastReport]("forecast")
	cacheDeployments  = cacheKind[[]hostapi.Deployment]("deployments")
	cacheMergedChange = cacheKind[hostapi.Change]("merged")
	cacheNotices      = cacheKind[[]HostNotice]("notices")
	cacheTodos        = cacheKind[[]hostapi.Todo]("todos")
	cacheTodosNotice  = cacheKind[string]("todosNotice")
)

// cachedFrom is cachedOrFetched for any cache host, such as a tracker's.
func cachedFrom[T any](cacheHost, repoPath string, kind cacheKind[T], key string, fetch func(context.Context) (T, error)) (T, error) {
	if cached, _, ok := kind.get(cacheHost, repoPath, key); ok {
		return cached, nil
	}
	ctx, cancel := hostContext()
	defer cancel()
	value, err := fetch(ctx)
	if err != nil {
		var none T
		return none, err
	}
	kind.put(cacheHost, repoPath, key, value)
	return value, nil
}
