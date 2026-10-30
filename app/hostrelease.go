//go:build !nohost

package app

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/gspeager/current-client/core/changelog"
	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/hostapi"
)

// ReleaseDraft prefills the release form for a tag. NotesFrom is
// "changelog" when the changelog at the tag has a section for it, else
// "commits": the commits since Since, the tag before it ("" for none).
type ReleaseDraft struct {
	Tag       string `json:"tag"`
	Name      string `json:"name"`
	Notes     string `json:"notes"`
	NotesFrom string `json:"notesFrom"`
	Since     string `json:"since"`
	Remote    string `json:"remote"`
}

type NewReleaseInput struct {
	Tag   string `json:"tag"`
	Name  string `json:"name"`
	Notes string `json:"notes"`
}

// CanPublishReleases says whether a tag offers Publish release on host. It
// reads no cache and asks no host.
func (s *HostService) CanPublishReleases(repoPath string) bool {
	_, _, _, err := releasePublisher(repoPath)
	return err == nil
}

// ReleaseDraft reads Git only.
func (s *HostService) ReleaseDraft(repoPath, tag string) (ReleaseDraft, error) {
	remote, _, _, err := releasePublisher(repoPath)
	if err != nil {
		return ReleaseDraft{}, err
	}
	ctx := context.Background()
	ref := "refs/tags/" + tag
	if _, err := git.ResolveCommit(ctx, repoPath, ref); err != nil {
		return ReleaseDraft{}, err
	}
	draft := ReleaseDraft{Tag: tag, Name: tag, Remote: remote.Name}
	file := filepath.Base(changelog.Path(repoPath))
	if text, found, err := git.FileAt(ctx, repoPath, ref, file); err == nil && found {
		if notes, ok := changelog.SectionNotes(string(text), tag); ok {
			draft.Notes, draft.NotesFrom = notes, "changelog"
			return draft, nil
		}
	}
	if since, err := git.LatestTag(ctx, repoPath, ref+"^"); err == nil {
		draft.Since = since
	}
	release, err := changelog.Build(ctx, repoPath, draft.Since, ref, "")
	if err != nil {
		return ReleaseDraft{}, err
	}
	markdown := changelog.Markdown(tag, "", release.Entries, changelog.MarkdownOptions{})
	_, body, _ := strings.Cut(markdown, "\n")
	draft.Notes, draft.NotesFrom = strings.TrimSpace(body), "commits"
	return draft, nil
}

// PublishRelease is one request on a click, for a tag the caller has pushed.
func (s *HostService) PublishRelease(repoPath string, input NewReleaseInput) (hostapi.PublishedRelease, error) {
	_, info, publisher, err := releasePublisher(repoPath)
	if err != nil {
		return hostapi.PublishedRelease{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = input.Tag
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return publisher.PublishRelease(ctx, info.Project, hostapi.NewRelease{Tag: input.Tag, Name: name, Notes: input.Notes})
}

func releasePublisher(repoPath string) (git.Remote, HostInfo, hostapi.ReleasePublisher, error) {
	remote, info, ok, err := primaryHost(repoPath)
	if err != nil {
		return git.Remote{}, HostInfo{}, nil, err
	}
	if !ok || !info.Enabled {
		return git.Remote{}, HostInfo{}, nil, errors.New("integration is off for this repository's host")
	}
	publisher, ok := providerFor(info).(hostapi.ReleasePublisher)
	if !ok {
		return git.Remote{}, HostInfo{}, nil, errors.New("this host has no releases the app can publish")
	}
	return remote, info, publisher, nil
}
