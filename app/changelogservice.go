package app

import (
	"context"
	"encoding/base64"

	"github.com/gspeager/current-client/core/changelog"
	"github.com/gspeager/current-client/core/git"
)

type ChangelogService struct{}

type ChangelogEntry struct {
	SHA         string   `json:"sha"`
	Type        string   `json:"type"`
	Scope       string   `json:"scope"`
	Description string   `json:"description"`
	Section     string   `json:"section"`
	Author      string   `json:"author"`
	CoAuthors   []string `json:"coAuthors"`
	Date        string   `json:"date"`
}

type ChangelogMarkdownOptions struct {
	Authors bool `json:"authors"`
	Dates   bool `json:"dates"`
}

// Tag is empty for the commits after the newest tag; the frontend supplies
// that section's version.
type ChangelogReleaseSection struct {
	Tag              string           `json:"tag"`
	Version          string           `json:"version"`
	Date             string           `json:"date"`
	SuggestedVersion string           `json:"suggestedVersion"`
	Entries          []ChangelogEntry `json:"entries"`
	Sections         []string         `json:"sections"`
}

type ChangelogMarkdownSection struct {
	Version string           `json:"version"`
	Date    string           `json:"date"`
	Entries []ChangelogEntry `json:"entries"`
}

func toEntries(entries []ChangelogEntry) []changelog.Entry {
	return mapSlice(entries, func(e ChangelogEntry) changelog.Entry { return changelog.Entry(e) })
}

// LatestTag is the default start of a changelog range; "" means there are no
// tags, or no commits yet.
func (s *ChangelogService) LatestTag(repoPath, ref string) (string, error) {
	if !hasCommits(repoPath) {
		return "", nil
	}
	return git.LatestTag(context.Background(), repoPath, ref)
}

func hasCommits(repoPath string) bool {
	_, err := git.ResolveCommit(context.Background(), repoPath, "HEAD")
	return err == nil
}

// since is a git --since value; empty means no time limit. split cuts the
// range at every tag inside it.
// Build returns no sections before the first commit.
func (s *ChangelogService) Build(repoPath, from, to, since string, split bool) ([]ChangelogReleaseSection, error) {
	if !hasCommits(repoPath) {
		return []ChangelogReleaseSection{}, nil
	}
	sections, err := changelog.BuildReleases(context.Background(), repoPath, from, to, since, split)
	if err != nil {
		return nil, err
	}
	return mapSlice(sections, func(r changelog.ReleaseSection) ChangelogReleaseSection {
		return ChangelogReleaseSection{
			Tag:              r.Tag,
			Version:          r.Version,
			Date:             r.Date,
			SuggestedVersion: r.SuggestedVersion,
			Entries:          mapSlice(r.Entries, func(e changelog.Entry) ChangelogEntry { return ChangelogEntry(e) }),
			Sections:         r.Sections,
		}
	}), nil
}

func (s *ChangelogService) Markdown(sections []ChangelogMarkdownSection, opts ChangelogMarkdownOptions) string {
	return changelog.MarkdownReleases(mapSlice(sections, func(m ChangelogMarkdownSection) changelog.MarkdownSection {
		return changelog.MarkdownSection{Version: m.Version, Date: m.Date, Entries: toEntries(m.Entries)}
	}), changelog.MarkdownOptions(opts))
}

func (s *ChangelogService) ExistingVersions(repoPath, markdown string) ([]string, error) {
	return changelog.ExistingVersions(repoPath, markdown)
}

func (s *ChangelogService) Write(repoPath, markdown string, replace bool) error {
	return changelog.Write(repoPath, markdown, replace)
}

func (s *ChangelogService) RenderHTML(markdown string) (string, error) {
	return changelog.HTML(markdown)
}

// The Save methods report false when the dialog is cancelled.

func (s *ChangelogService) SaveMarkdown(markdown, filename string) (bool, error) {
	return saveFile("Save Markdown", filename, "Markdown", "*.md", []byte(markdown))
}

func (s *ChangelogService) SaveHTML(markdown, filename string) (bool, error) {
	doc, err := changelog.HTMLDocument("Changelog", markdown)
	if err != nil {
		return false, err
	}
	return saveFile("Save as HTML", filename, "HTML", "*.html", []byte(doc))
}

// The frontend renders the PDF (Go has no maintained PDF library); bindings
// carry bytes as base64.
func (s *ChangelogService) SavePDF(pdfBase64, filename string) (bool, error) {
	pdf, err := base64.StdEncoding.DecodeString(pdfBase64)
	if err != nil {
		return false, err
	}
	return saveFile("Save as PDF", filename, "PDF", "*.pdf", pdf)
}
