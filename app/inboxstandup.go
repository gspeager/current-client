package app

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/gspeager/current-client/core/git"
)

// StandupRepo is what the user did in one open repository since a time:
// their commits on any branch from Git, and with the host on, their changes
// and reviews from the last refresh's cache. Nothing is asked of a host.
type StandupRepo struct {
	Path        string          `json:"path"`
	Name        string          `json:"name"`
	Commits     []StandupCommit `json:"commits"`
	Error       string          `json:"error"`
	HostEnabled bool            `json:"hostEnabled"`
	FetchedAt   *time.Time      `json:"fetchedAt"`
	Prefix      string          `json:"prefix"`
	Opened      []StandupChange `json:"opened"`
	Merged      []StandupChange `json:"merged"`
	Reviewed    []StandupChange `json:"reviewed"`
}

// StandupChange is a merge request / pull request; Review is the user's
// review on it (approved, changes_requested, commented) in Reviewed.
type StandupChange struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	WebURL    string    `json:"webUrl"`
	UpdatedAt time.Time `json:"updatedAt"`
	Review    string    `json:"review"`
}

type StandupCommit struct {
	SHA     string    `json:"sha"`
	Subject string    `json:"subject"`
	Date    time.Time `json:"date"`
}

const standupCommitCap = 200

// Standup reads each repository's own identity (user.email) and keeps the
// commits it authored on any branch, remote branch or tag, merges left out.
// since is an RFC 3339 time.
func (s *InboxService) Standup(paths []string, since string) ([]StandupRepo, error) {
	from, err := time.Parse(time.RFC3339, since)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	repos := make([]StandupRepo, 0, len(paths))
	for _, path := range paths {
		repo := StandupRepo{Path: path, Name: filepath.Base(path), Commits: []StandupCommit{}}
		commits, err := ownCommits(ctx, path, since)
		if err != nil {
			repo.Error = err.Error()
		}
		repo.Commits = append(repo.Commits, commits...)
		standupHost(&repo, from)
		repos = append(repos, repo)
	}
	return repos, nil
}

func ownCommits(ctx context.Context, repoPath, since string) ([]StandupCommit, error) {
	me := strings.ToLower(git.CurrentUser(ctx, repoPath).Email)
	if me == "" {
		return nil, nil
	}
	entries, err := git.History(ctx, repoPath, standupCommitCap, 0, git.HistoryFilter{Ref: "--all", Since: since})
	if err != nil {
		return nil, err
	}
	var commits []StandupCommit
	for _, e := range entries {
		if len(e.ParentSHAs) < 2 && strings.ToLower(e.AuthorEmail) == me {
			commits = append(commits, StandupCommit{SHA: e.SHA, Subject: e.Subject, Date: e.Date})
		}
	}
	return commits, nil
}
