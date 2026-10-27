//go:build !nohost

package app

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gspeager/current-client/core/hostapi"
)

type NewIssueInput struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Labels      []string `json:"labels"`
}

// CanWriteIssues says whether the issue panel and diffs offer to create,
// comment on and close issues. It reads no cache and asks no host.
func (s *HostService) CanWriteIssues(repoPath string) bool {
	_, _, err := issueWriter(repoPath)
	return err == nil
}

// CreateIssue sends one request on a click. The new issue goes into the
// hover-card cache so its reference has a title at once.
func (s *HostService) CreateIssue(repoPath string, input NewIssueInput) (hostapi.Issue, error) {
	info, writer, err := issueWriter(repoPath)
	if err != nil {
		return hostapi.Issue{}, err
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return hostapi.Issue{}, errors.New("an issue needs a title")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	issue, err := writer.CreateIssue(ctx, info.Project, hostapi.NewIssue{Title: title, Description: input.Description, Labels: input.Labels})
	if err != nil {
		return hostapi.Issue{}, err
	}
	rememberIssueTitles(info, repoPath, []hostapi.Issue{issue})
	return issue, nil
}

// CommentOnIssue adds the host's answer to the cached comments, so the
// panel shows it without asking again.
func (s *HostService) CommentOnIssue(repoPath, key, body string) (IssueDetails, error) {
	info, writer, err := issueWriter(repoPath)
	if err != nil {
		return IssueDetails{}, err
	}
	if strings.TrimSpace(body) == "" {
		return IssueDetails{}, errors.New("a comment needs text")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	note, err := writer.CommentOnIssue(ctx, info.Project, key, body)
	if err != nil {
		return IssueDetails{}, err
	}
	if cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheIssueComments, key); ok {
		notes := append(append([]hostapi.Note{}, cached.([]hostapi.Note)...), note)
		hostClient.Cache.Put(info.Hostname, repoPath, cacheIssueComments, key, notes)
	}
	return s.IssueDetails(repoPath, key)
}

// SetIssueState closes or reopens an issue and keeps what the host answered.
// A closed issue leaves the assigned list, which holds open issues only.
func (s *HostService) SetIssueState(repoPath, key string, open bool) (IssueDetails, error) {
	info, writer, err := issueWriter(repoPath)
	if err != nil {
		return IssueDetails{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	issue, err := writer.SetIssueState(ctx, info.Project, key, open)
	if err != nil {
		return IssueDetails{}, err
	}
	hostClient.Cache.Put(info.Hostname, repoPath, cacheIssueDetails, key, issue)
	rememberIssueTitles(info, repoPath, []hostapi.Issue{issue.Issue})
	if !open {
		if cached, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheAssigned, info.Project); ok {
			var still []hostapi.Issue
			for _, is := range cached.([]hostapi.Issue) {
				if strconv.Itoa(is.Number) != key {
					still = append(still, is)
				}
			}
			hostClient.Cache.Put(info.Hostname, repoPath, cacheAssigned, info.Project, still)
		}
	}
	return s.IssueDetails(repoPath, key)
}

func issueWriter(repoPath string) (HostInfo, hostapi.IssueWriter, error) {
	_, info, ok, err := primaryHost(repoPath)
	if err != nil {
		return HostInfo{}, nil, err
	}
	if !ok || !info.Enabled {
		return HostInfo{}, nil, errors.New("integration is off for this repository's host")
	}
	writer, ok := issueTrackerFor(info).(hostapi.IssueWriter)
	if !ok {
		return HostInfo{}, nil, errors.New("issues on this tracker are changed on the tracker itself")
	}
	return info, writer, nil
}
