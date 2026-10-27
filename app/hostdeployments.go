//go:build !nohost

package app

import (
	"context"
	"strconv"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/hostapi"
)

const cacheDeployments = "deployments"

// EnvironmentInfo is an environment's latest deployment and whether its
// commit is in this repository, which Git needs to say what it contains.
type EnvironmentInfo struct {
	hostapi.Deployment
	Fetched bool `json:"fetched"`
}

// Deployments is the environments as the last refresh found them. Enabled
// is false, with nothing else, while the host is off.
type Deployments struct {
	Enabled      bool              `json:"enabled"`
	FetchedAt    *time.Time        `json:"fetchedAt"`
	Environments []EnvironmentInfo `json:"environments"`
}

// Reach is whether a commit is in an environment: "in" when it is an
// ancestor of the deployed commit, "out" when it isn't, "unknown" when the
// deployed commit or the commit itself isn't fetched.
type Reach struct {
	Environment string `json:"environment"`
	State       string `json:"state"`
	Production  bool   `json:"production"`
}

// refreshDeployments is the refresh's one read of the environments; a
// failed one keeps the last answer.
func refreshDeployments(ctx context.Context, provider hostapi.Provider, info HostInfo, repoPath string) {
	deployer, ok := provider.(hostapi.Deployer)
	if !ok {
		return
	}
	if deployments, err := deployer.Deployments(ctx, info.Project); err == nil {
		hostClient.Cache.Put(info.Hostname, repoPath, cacheDeployments, info.Project, deployments)
	}
}

func cachedDeployments(repoPath string) ([]hostapi.Deployment, *time.Time, bool) {
	_, info, ok, err := primaryHost(repoPath)
	if err != nil || !ok || !info.Enabled {
		return nil, nil, false
	}
	cached, at, found := hostClient.Cache.Get(info.Hostname, repoPath, cacheDeployments, info.Project)
	if !found {
		return nil, nil, true
	}
	return cached.([]hostapi.Deployment), &at, true
}

// Deployments reads the cache and Git only.
func (s *HostService) Deployments(repoPath string) (Deployments, error) {
	deployments, fetchedAt, enabled := cachedDeployments(repoPath)
	result := Deployments{Enabled: enabled, FetchedAt: fetchedAt, Environments: []EnvironmentInfo{}}
	ctx := context.Background()
	for _, d := range deployments {
		_, err := git.ResolveCommit(ctx, repoPath, d.SHA+"^{commit}")
		result.Environments = append(result.Environments, EnvironmentInfo{Deployment: d, Fetched: err == nil})
	}
	return result, nil
}

// Reach says, for each commit, where it is among the environments the last
// refresh found, from Git alone: one walk per environment.
func (s *HostService) Reach(repoPath string, commits []string) (map[string][]Reach, error) {
	deployments, _, _ := cachedDeployments(repoPath)
	return reachOf(context.Background(), repoPath, deployments, commits)
}

func reachOf(ctx context.Context, repoPath string, deployments []hostapi.Deployment, commits []string) (map[string][]Reach, error) {
	result := map[string][]Reach{}
	for _, d := range deployments {
		reached, err := git.Reached(ctx, repoPath, d.SHA, commits)
		if err != nil {
			return nil, err
		}
		for _, c := range commits {
			state := "unknown"
			if in, known := reached[c]; known {
				state = map[bool]string{true: "in", false: "out"}[in]
			}
			result[c] = append(result[c], Reach{Environment: d.Environment, State: state, Production: d.Production})
		}
	}
	return result, nil
}

// undeployedWindow is how far back the Inbox looks for merged work that
// hasn't reached production: older merges are history, not a to-do.
const undeployedWindow = 30 * 24 * time.Hour

// undeployedRows reads the cache and Git only: the user's changes merged in
// the last month, from the Closed list and merges made in the app, whose
// commit isn't in a production environment yet. One the deployed commit or
// the merge isn't fetched for is left out, since Git can't tell.
func undeployedRows(repoPath string, info HostInfo, list ChangeList) []InboxRow {
	deployments, _, _ := cachedDeployments(repoPath)
	var production []hostapi.Deployment
	for _, d := range deployments {
		if d.Production {
			production = append(production, d)
		}
	}
	if len(production) == 0 || list.Me == "" {
		return nil
	}
	closed := list
	closed.Changes = nil
	fillChangeList(&closed, info, repoPath, cacheClosed)
	changes := closed.Changes
	for _, value := range hostClient.Cache.Entries(info.Hostname, repoPath, cacheMergedChange) {
		changes = append(changes, value.(hostapi.Change))
	}
	var commits []string
	mine := map[string]hostapi.Change{}
	for _, c := range changes {
		if c.State == hostapi.ChangeMerged && c.Author.Username == list.Me && c.MergeCommitSHA != "" &&
			time.Since(c.UpdatedAt) < undeployedWindow && mine[c.MergeCommitSHA].Number == 0 {
			mine[c.MergeCommitSHA] = c
			commits = append(commits, c.MergeCommitSHA)
		}
	}
	reach, err := reachOf(context.Background(), repoPath, production, commits)
	if err != nil {
		return nil
	}
	var rows []InboxRow
	for _, sha := range commits {
		for _, r := range reach[sha] {
			if r.State != "out" {
				continue
			}
			c := mine[sha]
			rows = append(rows, InboxRow{ID: "undeployed:" + strconv.Itoa(c.Number) + ":" + r.Environment, Kind: "undeployed", Number: c.Number, Prefix: list.Prefix,
				Title: c.Title, Detail: r.Environment, WebURL: c.WebURL, FetchedAt: list.FetchedAt})
			break
		}
	}
	return rows
}
