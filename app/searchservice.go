package app

import (
	"context"
	"slices"
	"strings"

	"github.com/gspeager/current-client/core/git"
)

type SearchService struct{}

type SearchOptions struct {
	Pattern    string `json:"pattern"`
	Rev        string `json:"rev"`
	IgnoreCase bool   `json:"ignoreCase"`
	WholeWord  bool   `json:"wholeWord"`
	Regexp     bool   `json:"regexp"`
}

type SearchMatch struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Text string `json:"text"`
}

type SearchResult struct {
	Matches   []SearchMatch `json:"matches"`
	Truncated bool          `json:"truncated"`
}

// Enough to find what's wanted; past this, the search needs narrowing.
const maxSearchMatches = 1000

// SearchContents takes Wails' per-call ctx so a new search can cancel the last.
func (s *SearchService) SearchContents(ctx context.Context, repoPath string, opts SearchOptions) (SearchResult, error) {
	matches, truncated, err := git.Grep(ctx, repoPath, git.GrepOptions{
		Pattern:    opts.Pattern,
		Rev:        opts.Rev,
		IgnoreCase: opts.IgnoreCase,
		WholeWord:  opts.WholeWord,
		Regexp:     opts.Regexp,
		MaxMatches: maxSearchMatches,
	})
	return SearchResult{Matches: mapSlice(matches, func(m git.GrepMatch) SearchMatch { return SearchMatch(m) }), Truncated: truncated}, err
}

// RepositoryMatches is what Go to anything finds in one repository.
type RepositoryMatches struct {
	Files    []string      `json:"files"`
	Branches []BranchMatch `json:"branches"`
	Commits  []CommitInfo  `json:"commits"`
	Notes    []NoteInfo    `json:"notes"`
}

type BranchMatch struct {
	Name         string `json:"name"`
	CheckoutName string `json:"checkoutName"`
	Remote       bool   `json:"remote"`
	Current      bool   `json:"current"`
}

// Go to anything's own limits, for each repository.
const (
	mostFileMatches   = 20
	mostBranchMatches = 10
	mostCommitMatches = 10
	mostNoteMatches   = 10
)

// InRepository finds file names, branches, commits on the current branch and notes
// in one repository, for Go to anything across the open ones. Ignored and
// untracked files aren't listed. ctx is Wails' per call, so a new query cancels it.
func (s *SearchService) InRepository(ctx context.Context, repoPath, query string) (RepositoryMatches, error) {
	query = strings.TrimSpace(query)
	matches := RepositoryMatches{Files: []string{}, Branches: []BranchMatch{}, Commits: []CommitInfo{}, Notes: []NoteInfo{}}
	if query == "" {
		return matches, nil
	}
	lower := strings.ToLower(query)
	has := func(texts ...string) bool {
		return slices.ContainsFunc(texts, func(text string) bool { return strings.Contains(strings.ToLower(text), lower) })
	}
	files, err := git.FindTracked(ctx, repoPath, query, mostFileMatches)
	if err != nil {
		return matches, err
	}
	matches.Files = append(matches.Files, files...)
	if matches.Branches, err = branchMatches(ctx, repoPath, has); err != nil {
		return matches, err
	}
	commits, err := git.SearchHistory(ctx, repoPath, query, mostCommitMatches)
	if err != nil {
		return matches, err
	}
	matches.Commits = commitInfos(commits)
	notes, err := (&NotesService{}).List(repoPath)
	if err != nil {
		return matches, err
	}
	for _, note := range notes {
		if len(matches.Notes) < mostNoteMatches && has(note.Text, note.Target, note.Subject) {
			matches.Notes = append(matches.Notes, note)
		}
	}
	return matches, nil
}

func branchMatches(ctx context.Context, repoPath string, has func(...string) bool) ([]BranchMatch, error) {
	local, err := git.ListBranches(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	remote, err := git.ListRemote(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	remotes, err := git.ListRemotes(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	found := []BranchMatch{}
	for _, b := range local {
		if has(b.Name) {
			found = append(found, BranchMatch{Name: b.Name, CheckoutName: b.Name, Current: b.Current})
		}
	}
	for _, name := range remote {
		if has(name) {
			found = append(found, BranchMatch{Name: name, CheckoutName: localNameOf(name, remotes), Remote: true})
		}
	}
	return found[:min(len(found), mostBranchMatches)], nil
}

// The name a remote branch is checked out as, as the frontend's localNameFor:
// after the longest remote name it starts with, else after its first slash.
func localNameOf(name string, remotes []git.Remote) string {
	longest := ""
	for _, r := range remotes {
		if strings.HasPrefix(name, r.Name+"/") && len(r.Name) > len(longest) {
			longest = r.Name
		}
	}
	if longest != "" {
		return name[len(longest)+1:]
	}
	_, local, _ := strings.Cut(name, "/")
	return local
}
