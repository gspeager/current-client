//go:build !nohost

package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/hostapi"
)

// ChangeDraft prefills the in-app form for a merge request / pull request
// from a branch: the title from its commits (one commit: its subject; more:
// the oldest's), the description from the commit list inside the
// repository's template when it has one.
type ChangeDraft struct {
	Branch      string `json:"branch"`
	Remote      string `json:"remote"`
	HasUpstream bool   `json:"hasUpstream"`
	Target      string `json:"target"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
}

type NewChangeInput struct {
	Target      string `json:"target"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Draft       bool   `json:"draft"`
}

func (s *HostService) ChangeDraft(repoPath, branch string) (ChangeDraft, error) {
	remote, info, ok, err := primaryHost(repoPath)
	if err != nil {
		return ChangeDraft{}, err
	}
	if !ok || !info.Enabled || providerFor(info) == nil {
		return ChangeDraft{}, errors.New("integration is off for this repository's host")
	}
	ctx := context.Background()
	draft := ChangeDraft{Branch: branch, Remote: remote.Name, Kind: changeKind(info)}
	branches, err := git.ListBranches(ctx, repoPath)
	if err != nil {
		return ChangeDraft{}, err
	}
	for _, b := range branches {
		if b.Name == branch {
			draft.HasUpstream = b.Upstream != ""
		}
	}
	if draft.Target, err = git.DefaultBranch(ctx, repoPath); err != nil || draft.Target == "" {
		draft.Target = "main"
	}
	commits, err := git.RangeHistory(ctx, repoPath, remote.Name+"/"+draft.Target, branch, "")
	if err != nil {
		commits, err = git.RangeHistory(ctx, repoPath, draft.Target, branch, "")
		if err != nil {
			return ChangeDraft{}, err
		}
	}
	if len(commits) > 0 {
		draft.Title = commits[len(commits)-1].Subject
	}
	var list strings.Builder
	for i := len(commits) - 1; i >= 0; i-- {
		fmt.Fprintf(&list, "- %s\n", commits[i].Subject)
	}
	draft.Description = strings.TrimRight(list.String(), "\n")
	if template := changeTemplate(repoPath, info); template != "" {
		draft.Description += "\n\n" + template
	}
	return draft, nil
}

// changeTemplate is the repository's merge request / pull request template,
// GitLab's Default.md or first template, GitHub's in any of its places.
func changeTemplate(repoPath string, info HostInfo) string {
	var candidates []string
	if isGitLab(info) {
		dir := filepath.Join(repoPath, ".gitlab", "merge_request_templates")
		candidates = append(candidates, filepath.Join(dir, "Default.md"))
		if entries, err := os.ReadDir(dir); err == nil {
			names := make([]string, 0, len(entries))
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".md") {
					names = append(names, e.Name())
				}
			}
			sort.Strings(names)
			for _, n := range names {
				candidates = append(candidates, filepath.Join(dir, n))
			}
		}
	} else {
		for _, p := range []string{".github/pull_request_template.md", ".github/PULL_REQUEST_TEMPLATE.md", "pull_request_template.md", "PULL_REQUEST_TEMPLATE.md", "docs/pull_request_template.md"} {
			candidates = append(candidates, filepath.Join(repoPath, filepath.FromSlash(p)))
		}
	}
	for _, path := range candidates {
		if data, err := os.ReadFile(path); err == nil {
			return strings.TrimSpace(string(data))
		}
	}
	return ""
}

// CreateChange is the one write to the host. The created change goes
// straight into the cache so the branch shows it without a refresh.
func (s *HostService) CreateChange(repoPath, branch string, input NewChangeInput) (hostapi.Change, error) {
	_, info, ok, err := primaryHost(repoPath)
	if err != nil {
		return hostapi.Change{}, err
	}
	if !ok || !info.Enabled {
		return hostapi.Change{}, errors.New("integration is off for this repository's host")
	}
	provider := providerFor(info)
	if provider == nil {
		return hostapi.Change{}, errors.New("no provider for this host yet")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	change, err := provider.CreateChange(ctx, info.Project, hostapi.NewChange{
		SourceBranch: remoteBranchOf(ctx, repoPath, branch),
		TargetBranch: input.Target,
		Title:        input.Title,
		Description:  input.Description,
		Draft:        input.Draft,
	})
	if err != nil {
		return hostapi.Change{}, err
	}
	cached := cachedChanges{}
	if existing, _, ok := hostClient.Cache.Get(info.Hostname, repoPath, cacheChanges, info.Project); ok {
		cached = existing.(cachedChanges)
	}
	cached.changes = append([]hostapi.Change{change}, cached.changes...)
	hostClient.Cache.Put(info.Hostname, repoPath, cacheChanges, info.Project, cached)
	return change, nil
}
