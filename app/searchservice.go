package app

import (
	"context"

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
