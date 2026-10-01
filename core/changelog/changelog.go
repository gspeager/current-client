package changelog

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/gspeager/current-client/core/git"
)

const (
	Breaking = "Breaking"
	Added    = "Added"
	Changed  = "Changed"
	Fixed    = "Fixed"
	// Commits that don't follow Conventional Commits.
	Other = "Other"

	Unreleased = "Unreleased"
)

// Types without a Keep a Changelog section get one named after them; a custom
// type (not listed here) is capitalized.
var typeSections = map[string]string{
	"feat":     Added,
	"fix":      Fixed,
	"perf":     Changed,
	"revert":   Changed,
	"docs":     "Documentation",
	"style":    "Style",
	"refactor": "Refactoring",
	"test":     "Tests",
	"build":    "Build",
	"ci":       "CI",
	"chore":    "Chores",
}

var sectionOrder = []string{Breaking, Added, Changed, Fixed, "Documentation", "Style", "Refactoring", "Tests", "Build", "CI", "Chores"}

// Entry.Type is empty for a commit that doesn't follow Conventional Commits;
// its Description is then the whole subject.
type Entry struct {
	SHA         string
	Type        string
	Scope       string
	Description string
	Section     string
	Author      string
	CoAuthors   []string
	Date        string // YYYY-MM-DD, local time
}

type MarkdownOptions struct {
	Authors bool
	Dates   bool
}

// Release.Sections lists the sections its entries fall into, in Markdown order.
type Release struct {
	SuggestedVersion string
	Entries          []Entry
	Sections         []string
}

// An empty from means all of to's history. With since set, the version is
// suggested from the newest tag reachable from to, as there is no from tag.
func Build(ctx context.Context, repoPath, from, to, since string) (Release, error) {
	commits, err := git.RangeHistory(ctx, repoPath, from, to, since)
	if err != nil {
		return Release{}, err
	}
	base, err := versionBase(ctx, repoPath, from, to, since)
	if err != nil {
		return Release{}, err
	}
	return newRelease(base, commits), nil
}

func versionBase(ctx context.Context, repoPath, from, to, since string) (string, error) {
	if since == "" {
		return from, nil
	}
	return git.LatestTag(ctx, repoPath, to)
}

// newRelease skips merge commits: they record how work arrived, not the work.
func newRelease(base string, commits []git.HistoryEntry) Release {
	var entries []Entry
	bySection := map[string][]string{}
	for _, c := range commits {
		if len(c.ParentSHAs) > 1 {
			continue
		}
		e := entryFor(c)
		entries = append(entries, e)
		bySection[e.Section] = append(bySection[e.Section], c.SHA)
	}
	return Release{SuggestedVersion: SuggestVersion(base, entries), Entries: entries, Sections: orderedSections(bySection)}
}

func entryFor(c git.HistoryEntry) Entry {
	e := Entry{SHA: c.SHA, Description: c.Subject, Section: Other, Author: c.AuthorName, CoAuthors: coAuthorNames(c.Trailers), Date: c.Date.Format("2006-01-02")}
	cc := git.ParseConventional(c.Subject, c.Body)
	if cc.Type == "" {
		return e
	}
	e.Type, e.Scope, e.Description, e.Section = cc.Type, cc.Scope, cc.Description, SectionFor(cc)
	if cc.BreakingNote != "" {
		e.Description = cc.BreakingNote
	}
	return e
}

func SectionFor(c git.ConventionalCommit) string {
	switch {
	case c.Type == "":
		return Other
	case c.Breaking:
		return Breaking
	case typeSections[c.Type] != "":
		return typeSections[c.Type]
	}
	return strings.ToUpper(c.Type[:1]) + c.Type[1:]
}

var semverTag = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(-[0-9A-Za-z.-]+)?$`)

// SuggestVersion returns "" when lastTag isn't a semantic version. A
// prerelease tag suggests its own release (v0.1.0-alpha.1 → 0.1.0).
func SuggestVersion(lastTag string, entries []Entry) string {
	m := semverTag.FindStringSubmatch(lastTag)
	if m == nil {
		return ""
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])
	if m[4] == "" {
		major, minor, patch = bump(major, minor, patch, entries)
	}
	return fmt.Sprintf("%d.%d.%d", major, minor, patch)
}

func bump(major, minor, patch int, entries []Entry) (int, int, int) {
	has := map[string]bool{}
	for _, e := range entries {
		has[e.Section] = true
	}
	switch {
	case has[Breaking] && major > 0:
		return major + 1, 0, 0
	case has[Breaking] || has[Added]:
		return major, minor + 1, 0
	default:
		return major, minor, patch + 1
	}
}

// Markdown lists every entry it is given; the caller filters. An Unreleased
// heading takes no date.
func Markdown(version, date string, entries []Entry, opts MarkdownOptions) string {
	var b strings.Builder
	if version == Unreleased {
		b.WriteString("## [" + Unreleased + "]\n")
	} else {
		fmt.Fprintf(&b, "## [%s] - %s\n", version, date)
	}
	lines := map[string][]string{}
	for _, e := range entries {
		lines[e.Section] = append(lines[e.Section], markdownLine(e, opts))
	}
	for _, section := range orderedSections(lines) {
		b.WriteString("\n### " + section + "\n\n" + strings.Join(lines[section], "\n") + "\n")
	}
	return b.String()
}

func markdownLine(e Entry, opts MarkdownOptions) string {
	line := "- " + e.Description
	if e.Scope != "" {
		line = fmt.Sprintf("- **%s:** %s", e.Scope, e.Description)
	}
	var extra []string
	if opts.Authors && e.Author != "" {
		extra = append(append(extra, e.Author), e.CoAuthors...)
	}
	if opts.Dates && e.Date != "" {
		extra = append(extra, e.Date)
	}
	if len(extra) > 0 {
		line += " (" + strings.Join(extra, ", ") + ")"
	}
	return line
}

// orderedSections puts the known sections first, then custom types
// alphabetically, then Other.
func orderedSections(lines map[string][]string) []string {
	var order, custom []string
	for _, s := range sectionOrder {
		if lines[s] != nil {
			order = append(order, s)
		}
	}
	for s := range lines {
		if s != Other && !slices.Contains(sectionOrder, s) {
			custom = append(custom, s)
		}
	}
	slices.Sort(custom)
	order = append(order, custom...)
	if lines[Other] != nil {
		order = append(order, Other)
	}
	return order
}

// ReleaseSection is one release's share of a range. Tag is empty for the
// commits after the newest tag, whose version the caller chooses.
type ReleaseSection struct {
	Tag     string
	Version string
	Date    string
	Release
}

// BuildReleases returns the range as one untagged section, or, with split,
// cut at every tag inside it, newest first. Tags with no commits of their own
// (before since, or only merges) are left out.
//
// Splitting reads the range once and assigns each commit to the oldest release
// that contains it by walking parents from each tag, oldest tag first; one git
// log per release took seconds on a repository with a hundred tags.
func BuildReleases(ctx context.Context, repoPath, from, to, since string, split bool) ([]ReleaseSection, error) {
	if !split {
		release, err := Build(ctx, repoPath, from, to, since)
		return []ReleaseSection{{Release: release}}, err
	}
	tags, err := git.TagsInRange(ctx, repoPath, from, to)
	if err != nil {
		return nil, err
	}
	tags = onePerCommit(tags)
	commits, err := git.RangeHistory(ctx, repoPath, from, to, since)
	if err != nil {
		return nil, err
	}
	toSHA, err := git.ResolveCommit(ctx, repoPath, to)
	if err != nil {
		return nil, err
	}

	// Release 0 is the untagged top; release i is tags[i-1].
	byID := make(map[string]git.HistoryEntry, len(commits))
	for _, c := range commits {
		byID[c.SHA] = c
	}
	releaseOf := make(map[string]int, len(commits))
	claim := func(start string, release int) {
		stack := []string{start}
		for len(stack) > 0 {
			sha := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			c, inRange := byID[sha]
			if _, claimed := releaseOf[sha]; claimed || !inRange {
				continue
			}
			releaseOf[sha] = release
			stack = append(stack, c.ParentSHAs...)
		}
	}
	for i := len(tags) - 1; i >= 0; i-- {
		claim(tags[i].SHA, i+1)
	}
	claim(toSHA, 0)

	grouped := make([][]git.HistoryEntry, len(tags)+1)
	for _, c := range commits {
		grouped[releaseOf[c.SHA]] = append(grouped[releaseOf[c.SHA]], c)
	}
	top := from
	if len(tags) > 0 {
		top = tags[0].Name
	} else if top, err = versionBase(ctx, repoPath, from, to, since); err != nil {
		return nil, err
	}

	var sections []ReleaseSection
	for i, group := range grouped {
		section := ReleaseSection{Release: newRelease(top, group)}
		if i > 0 {
			t := tags[i-1]
			section = ReleaseSection{Tag: t.Name, Version: tagVersion(t.Name), Date: t.Date, Release: newRelease("", group)}
		}
		if len(section.Entries) > 0 || len(tags) == 0 {
			sections = append(sections, section)
		}
	}
	return sections, nil
}

// onePerCommit keeps one tag per commit (they arrive next to each other),
// preferring a version tag, so v1.0.0 names the release rather than a
// deploy or marker tag on the same commit.
func onePerCommit(tags []git.DatedTag) []git.DatedTag {
	var kept []git.DatedTag
	for _, t := range tags {
		last := len(kept) - 1
		switch {
		case last < 0 || kept[last].SHA != t.SHA:
			kept = append(kept, t)
		case !semverTag.MatchString(kept[last].Name) && semverTag.MatchString(t.Name):
			kept[last] = t
		}
	}
	return kept
}

// tagVersion drops the "v" from a semantic-version tag (v1.2.0 → 1.2.0).
func tagVersion(tag string) string {
	if semverTag.MatchString(tag) {
		return strings.TrimPrefix(tag, "v")
	}
	return tag
}

type MarkdownSection struct {
	Version string
	Date    string
	Entries []Entry
}

func MarkdownReleases(sections []MarkdownSection, opts MarkdownOptions) string {
	parts := make([]string, len(sections))
	for i, s := range sections {
		parts[i] = Markdown(s.Version, s.Date, s.Entries, opts)
	}
	return strings.Join(parts, "\n")
}

func coAuthorNames(trailers []git.Trailer) []string {
	var names []string
	for _, a := range git.CoAuthors(trailers) {
		names = append(names, a.Name)
	}
	return names
}
