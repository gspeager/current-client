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
	Key    string `json:"key"`
	Title  string `json:"title"`
	State  string `json:"state"`
	// Status is the tracker's own name for the state, where it has one.
	Status string `json:"status"`
	WebURL string `json:"webUrl"`
	Known  bool   `json:"known"`
}

const (
	issueBatchCap = 50
)

// IssueDetails is the issue panel: the issue with its description rendered,
// and its comments. Both are asked for the first time the panel opens and
// kept until the next refresh.
type IssueDetails struct {
	Issue           hostapi.IssueDetails `json:"issue"`
	Reference       string               `json:"reference"`
	DescriptionHTML string               `json:"descriptionHtml"`
	Comments        []IssueComment       `json:"comments"`
	// Writable says the issue's tracker takes comments and state changes
	// from the app: GitLab and GitHub, not Jira.
	Writable bool `json:"writable"`
}

type IssueComment struct {
	Author    hostapi.User `json:"author"`
	BodyHTML  string       `json:"bodyHtml"`
	CreatedAt time.Time    `json:"createdAt"`
}

func (s *HostService) IssueDetails(repoPath, key string) (IssueDetails, error) {
	src, err := issueSourceFor(repoPath, key)
	if err != nil {
		return IssueDetails{}, err
	}
	issue, err := cachedFrom(src.cacheHost, repoPath, cacheIssueDetails, key, func(ctx context.Context) (hostapi.IssueDetails, error) {
		issue, err := src.tracker.IssueDetails(ctx, src.project, key)
		if err == nil {
			cacheIssue.put(src.cacheHost, repoPath, key, issueInfo(issue.Issue))
		}
		return issue, err
	})
	if err != nil {
		return IssueDetails{}, err
	}
	notes, err := cachedFrom(src.cacheHost, repoPath, cacheIssueComments, key, func(ctx context.Context) ([]hostapi.Note, error) {
		return src.tracker.IssueComments(ctx, src.project, key)
	})
	if err != nil {
		return IssueDetails{}, err
	}
	details := IssueDetails{Issue: issue, Reference: src.tracker.Reference(key), DescriptionHTML: hostcontent.RenderMarkdown(hostcontent.WikiHeadings(issue.Description), src.webRoot), Writable: src.writable}
	for _, n := range notes {
		details.Comments = append(details.Comments, IssueComment{Author: n.Author, BodyHTML: hostcontent.RenderMarkdown(hostcontent.WikiHeadings(n.Body), src.webRoot), CreatedAt: n.CreatedAt})
	}
	return details, nil
}

// issueSource is where one issue is read: a host's #123 from the host, any
// other key from the repository's tracker, such as a Jira site.
type issueSource struct {
	cacheHost, project, webRoot string
	tracker                     hostapi.IssueTracker
	writable                    bool
}

func issueSourceFor(repoPath, key string) (issueSource, error) {
	if _, err := strconv.Atoi(key); err != nil {
		provider, site, ok := jiraFor(repoPath)
		if !ok {
			return issueSource{}, errors.New("integration is off for this repository's issue tracker")
		}
		return trackerSource(provider, site), nil
	}
	_, info, err := requireHost(repoPath)
	if err != nil {
		return issueSource{}, err
	}
	src := issueSourceOf(info)
	if src.tracker == nil {
		return issueSource{}, errors.New("no issue tracker for this host yet")
	}
	return src, nil
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

// IssueTitles is what the last fetch learned about each number, by number.
func (s *HostService) IssueTitles(repoPath string, numbers []int) (map[int]IssueInfo, error) {
	src, ok, err := hostIssueSource(repoPath)
	if err != nil || !ok {
		return map[int]IssueInfo{}, err
	}
	return byNumber(src.titles(repoPath, numberKeys(numbers))), nil
}

// FetchIssueTitles asks the host for the numbers not yet known, as
// issueSource.fetchTitles does.
func (s *HostService) FetchIssueTitles(repoPath string, numbers []int) (map[int]IssueInfo, error) {
	src, ok, err := hostIssueSource(repoPath)
	if err != nil || !ok || src.tracker == nil {
		return map[int]IssueInfo{}, err
	}
	return byNumber(src.fetchTitles(repoPath, numberKeys(numbers))), nil
}

func hostIssueSource(repoPath string) (issueSource, bool, error) {
	_, info, ok, err := enabledHost(repoPath)
	if err != nil || !ok {
		return issueSource{}, false, err
	}
	return issueSourceOf(info), true, nil
}

func issueSourceOf(info HostInfo) issueSource {
	tracker := issueTrackerFor(info)
	_, writable := tracker.(hostapi.IssueWriter)
	return issueSource{cacheHost: info.Hostname, project: info.Project, webRoot: webRoot(info), tracker: tracker, writable: writable}
}

func numberKeys(numbers []int) []string {
	return mapSlice(numbers, strconv.Itoa)
}

func byNumber(issues map[string]IssueInfo) map[int]IssueInfo {
	numbered := map[int]IssueInfo{}
	for key, issue := range issues {
		if number, err := strconv.Atoi(key); err == nil {
			numbered[number] = issue
		}
	}
	return numbered
}

// titles is what the cache knows about each key, by key.
func (src issueSource) titles(repoPath string, keys []string) map[string]IssueInfo {
	issues := map[string]IssueInfo{}
	for _, key := range keys {
		if cached, _, ok := cacheIssue.get(src.cacheHost, repoPath, key); ok {
			issues[key] = cached
		}
	}
	return issues
}

// fetchTitles asks for the keys not yet known, up to the batch cap, four at a
// time, and remembers a key the tracker hasn't as known with no title, so it
// isn't asked for again until a refresh.
func (src issueSource) fetchTitles(repoPath string, keys []string) map[string]IssueInfo {
	var missing []string
	seen := map[string]bool{}
	for _, key := range keys {
		if seen[key] {
			continue
		}
		seen[key] = true
		if _, _, ok := cacheIssue.get(src.cacheHost, repoPath, key); !ok {
			missing = append(missing, key)
		}
	}
	missing = missing[:min(len(missing), issueBatchCap)]
	ctx, cancel := hostContext()
	defer cancel()
	inParallel(len(missing), func(i int) {
		key := missing[i]
		issue, err := src.tracker.IssueByKey(ctx, src.project, key)
		switch {
		case err == nil:
			src.remember(repoPath, []hostapi.Issue{issue})
		case hostapi.IsNotFound(err):
			unknown := IssueInfo{Known: true}
			if number, err := strconv.Atoi(key); err == nil {
				unknown.Number = number
			} else {
				unknown.Key = key
			}
			cacheIssue.put(src.cacheHost, repoPath, key, unknown)
		}
	})
	return src.titles(repoPath, keys)
}

// remember keeps each issue for hover cards. A host's key is its number.
func (src issueSource) remember(repoPath string, issues []hostapi.Issue) {
	for _, is := range issues {
		cacheIssue.put(src.cacheHost, repoPath, is.Key, issueInfo(is))
	}
}

func issueInfo(is hostapi.Issue) IssueInfo {
	return IssueInfo{Number: is.Number, Key: is.Key, Title: is.Title, State: is.State, Status: is.Status, WebURL: is.WebURL, Known: true}
}

// AssignedIssues is the issue picker's list, asked of the host when the
// picker first opens and read from the cache until a refresh. Each title is
// kept for hover cards too.
func (s *HostService) AssignedIssues(repoPath string) ([]hostapi.Issue, error) {
	if provider, site, ok := jiraFor(repoPath); ok {
		return trackerAssigned(repoPath, provider, site)
	}
	remote, info, ok, err := enabledHost(repoPath)
	if err != nil || !ok {
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
		issueSourceOf(info).remember(repoPath, issues)
		return issues, nil
	})
}
