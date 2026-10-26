//go:build !nohost

package app

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"time"

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostcontent"
)

// IssueInfo is what a hover card shows for an issue reference. Title is
// empty, with Known true, for a number the host has no issue for.
type IssueInfo struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	WebURL string `json:"webUrl"`
	Known  bool   `json:"known"`
}

const (
	cacheIssue         = "issue"
	cacheAssigned      = "assigned"
	cacheIssueDetails  = "issueDetails"
	cacheIssueComments = "issueComments"
	issueBatchCap      = 50
)

// IssueDetails is the issue panel: the issue with its description rendered,
// and its comments. Both are asked for the first time the panel opens and
// kept until the next refresh.
type IssueDetails struct {
	Issue           hostapi.IssueDetails `json:"issue"`
	Reference       string               `json:"reference"`
	DescriptionHTML string               `json:"descriptionHtml"`
	Comments        []IssueComment       `json:"comments"`
}

type IssueComment struct {
	Author    hostapi.User `json:"author"`
	BodyHTML  string       `json:"bodyHtml"`
	CreatedAt time.Time    `json:"createdAt"`
}

func (s *HostService) IssueDetails(repoPath, key string) (IssueDetails, error) {
	_, info, ok, err := primaryHost(repoPath)
	if err != nil {
		return IssueDetails{}, err
	}
	if !ok || !info.Enabled {
		return IssueDetails{}, errors.New("integration is off for this repository's host")
	}
	tracker := issueTrackerFor(info)
	if tracker == nil {
		return IssueDetails{}, errors.New("no issue tracker for this host yet")
	}
	issue, err := cachedOrFetched(info, repoPath, cacheIssueDetails, key, func(ctx context.Context, _ hostapi.Provider) (hostapi.IssueDetails, error) {
		issue, err := tracker.IssueDetails(ctx, info.Project, key)
		if err == nil {
			hostClient.Cache.Put(info.Hostname, repoPath, cacheIssue, key,
				IssueInfo{Number: issue.Number, Title: issue.Title, State: issue.State, WebURL: issue.WebURL, Known: true})
		}
		return issue, err
	})
	if err != nil {
		return IssueDetails{}, err
	}
	notes, err := cachedOrFetched(info, repoPath, cacheIssueComments, key, func(ctx context.Context, _ hostapi.Provider) ([]hostapi.Note, error) {
		return tracker.IssueComments(ctx, info.Project, key)
	})
	if err != nil {
		return IssueDetails{}, err
	}
	webRoot := "https://" + info.Hostname + "/"
	details := IssueDetails{Issue: issue, Reference: tracker.Reference(key), DescriptionHTML: hostcontent.RenderMarkdown(issue.Description, webRoot)}
	for _, n := range notes {
		details.Comments = append(details.Comments, IssueComment{Author: n.Author, BodyHTML: hostcontent.RenderMarkdown(n.Body, webRoot), CreatedAt: n.CreatedAt})
	}
	return details, nil
}

var closingWords = `(?i)\b(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?):?\s+`

// closedIssues is the references a description says the change closes,
// each once, in order: "Closes #12", "fixes #7", "Resolves: #3".
func closedIssues(description string, tracker hostapi.IssueTracker) []string {
	pattern := regexp.MustCompile(closingWords + tracker.KeyPattern().String())
	var refs []string
	seen := map[string]bool{}
	for _, m := range pattern.FindAllStringSubmatch(description, -1) {
		ref := tracker.Reference(m[1])
		if !seen[ref] {
			seen[ref] = true
			refs = append(refs, ref)
		}
	}
	return refs
}

// IssueTitles reads the cache only: what the last fetch learned about each
// number, keyed by number.
func (s *HostService) IssueTitles(repoPath string, numbers []int) (map[int]IssueInfo, error) {
	_, info, ok, err := primaryHost(repoPath)
	if err != nil || !ok || !info.Enabled {
		return map[int]IssueInfo{}, err
	}
	return cachedIssues(info, repoPath, numbers), nil
}

// FetchIssueTitles asks the host for the numbers not yet known, up to the
// batch cap, four at a time, and remembers a missing issue as known with no
// title so it isn't asked for again until a refresh.
func (s *HostService) FetchIssueTitles(repoPath string, numbers []int) (map[int]IssueInfo, error) {
	_, info, ok, err := primaryHost(repoPath)
	if err != nil || !ok || !info.Enabled {
		return map[int]IssueInfo{}, err
	}
	tracker := issueTrackerFor(info)
	if tracker == nil {
		return map[int]IssueInfo{}, nil
	}
	var missing []int
	seen := map[int]bool{}
	for _, number := range numbers {
		if seen[number] {
			continue
		}
		seen[number] = true
		if _, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheIssue, strconv.Itoa(number)); !ok {
			missing = append(missing, number)
		}
	}
	if len(missing) > issueBatchCap {
		missing = missing[:issueBatchCap]
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	inParallel(len(missing), func(i int) {
		number := missing[i]
		issue, err := tracker.IssueByKey(ctx, info.Project, strconv.Itoa(number))
		var hostErr *hostapi.HostError
		switch {
		case err == nil:
			hostClient.Cache.Put(info.Hostname, repoPath, cacheIssue, strconv.Itoa(number),
				IssueInfo{Number: number, Title: issue.Title, State: issue.State, WebURL: issue.WebURL, Known: true})
		case errors.As(err, &hostErr) && hostErr.Status == 404:
			hostClient.Cache.Put(info.Hostname, repoPath, cacheIssue, strconv.Itoa(number), IssueInfo{Number: number, Known: true})
		}
	})
	return cachedIssues(info, repoPath, numbers), nil
}

// AssignedIssues is the issue picker's list, asked of the host when the
// picker first opens and read from the cache until a refresh. Each title is
// kept for hover cards too.
func (s *HostService) AssignedIssues(repoPath string) ([]hostapi.Issue, error) {
	remote, info, ok, err := primaryHost(repoPath)
	if err != nil || !ok || !info.Enabled {
		return nil, err
	}
	return cachedOrFetched(info, repoPath, cacheAssigned, info.Project, func(ctx context.Context, provider hostapi.Provider) ([]hostapi.Issue, error) {
		me, err := currentUser(ctx, provider, info, repoPath, remote.Name)
		if err != nil {
			return nil, err
		}
		tracker, ok := provider.(hostapi.IssueTracker)
		if !ok {
			return nil, nil
		}
		issues, err := tracker.AssignedIssues(ctx, info.Project, me.Username)
		if err != nil {
			return nil, err
		}
		for _, is := range issues {
			hostClient.Cache.Put(info.Hostname, repoPath, cacheIssue, strconv.Itoa(is.Number),
				IssueInfo{Number: is.Number, Title: is.Title, State: is.State, WebURL: is.WebURL, Known: true})
		}
		return issues, nil
	})
}

func cachedIssues(info HostInfo, repoPath string, numbers []int) map[int]IssueInfo {
	issues := map[int]IssueInfo{}
	for _, number := range numbers {
		if cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheIssue, strconv.Itoa(number)); ok {
			issues[number] = cached.(IssueInfo)
		}
	}
	return issues
}

// CachedPipelineJobs is a pipeline's jobs if a click already fetched them;
// a hover card reads this and never fetches.
func (s *HostService) CachedPipelineJobs(repoPath string, pipelineID int64) ([]hostapi.Job, error) {
	_, info, ok, err := primaryHost(repoPath)
	if err != nil || !ok || !info.Enabled {
		return nil, err
	}
	if cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheJobs, strconv.FormatInt(pipelineID, 10)); ok {
		return cached.([]hostapi.Job), nil
	}
	return nil, nil
}
