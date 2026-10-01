package changelog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/internal/gittest"
)

var commitTime = 1_700_000_000

// Distinct dates keep git log's newest-first order deterministic.
func commit(t *testing.T, dir, message string) {
	t.Helper()
	commitTime++
	date := fmt.Sprintf("%d +0000", commitTime)
	t.Setenv("GIT_AUTHOR_DATE", date)
	t.Setenv("GIT_COMMITTER_DATE", date)
	gittest.Run(t, dir, "commit", "--allow-empty", "-m", message)
}

func TestBuildFromLastTag(t *testing.T) {
	dir := gittest.InitRepo(t)
	commit(t, dir, "feat: before the tag")
	gittest.Run(t, dir, "tag", "v1.2.3")
	commit(t, dir, "feat(history): filter by type")
	commit(t, dir, "fix: keep lane color")
	commit(t, dir, "docs: update readme")
	commit(t, dir, "Plain subject")
	commit(t, dir, "refactor(config): rename keys\n\nBREAKING CHANGE: settings keys renamed")
	commit(t, dir, "perf: faster graph")
	gittest.Run(t, dir, "checkout", "-q", "-b", "side")
	commit(t, dir, "fix(diff): wrap long lines")
	gittest.Run(t, dir, "checkout", "-q", "main")
	gittest.Run(t, dir, "merge", "--no-ff", "-m", "Merge branch 'side'", "side")

	tag, err := git.LatestTag(context.Background(), dir, "HEAD")
	if err != nil || tag != "v1.2.3" {
		t.Fatalf("LatestTag = %q, %v; want v1.2.3", tag, err)
	}
	release, err := Build(context.Background(), dir, tag, "HEAD", "")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if release.SuggestedVersion != "2.0.0" {
		t.Errorf("SuggestedVersion = %q, want 2.0.0", release.SuggestedVersion)
	}
	if got := strings.Join(release.Sections, ","); got != "Breaking,Added,Changed,Fixed,Documentation,Other" {
		t.Errorf("Sections = %s", got)
	}
	if len(release.Entries) != 7 {
		t.Fatalf("got %d entries, want 7 (merge commit and pre-tag commit excluded)", len(release.Entries))
	}
	if e := release.Entries[0]; e.Author != "Test" || e.Date == "" {
		t.Errorf("entry author/date = %q, %q", e.Author, e.Date)
	}

	got := Markdown("2.0.0", "2026-09-27", release.Entries, MarkdownOptions{})
	want := `## [2.0.0] - 2026-09-27

### Breaking

- **config:** settings keys renamed

### Added

- **history:** filter by type

### Changed

- faster graph

### Fixed

- **diff:** wrap long lines
- keep lane color

### Documentation

- update readme

### Other

- Plain subject
`
	if got != want {
		t.Errorf("Markdown =\n%s\nwant\n%s", got, want)
	}
}

func TestBuildFromStartOfHistory(t *testing.T) {
	dir := gittest.InitRepo(t)
	commit(t, dir, "feat: first")
	commit(t, dir, "Plain second")
	if tag, err := git.LatestTag(context.Background(), dir, "HEAD"); err != nil || tag != "" {
		t.Errorf("LatestTag without tags = %q, %v", tag, err)
	}

	release, err := Build(context.Background(), dir, "", "HEAD", "")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(release.Entries) != 2 || release.SuggestedVersion != "" {
		t.Errorf("got %+v", release)
	}
	if e := release.Entries[0]; e.Type != "" || e.Section != Other || e.Description != "Plain second" {
		t.Errorf("plain commit entry = %+v", e)
	}
}

func TestBuildRejectsUnknownRef(t *testing.T) {
	dir := gittest.InitRepo(t)
	commit(t, dir, "feat: first")
	if _, err := Build(context.Background(), dir, "--output=x", "HEAD", ""); err == nil {
		t.Error("an option-like ref should fail as an unknown commit")
	}
}

func TestSuggestVersion(t *testing.T) {
	feat := Entry{Section: Added}
	fix := Entry{Section: Fixed}
	breaking := Entry{Section: Breaking}
	hidden := Entry{Section: "Chores"}
	tests := []struct {
		tag     string
		entries []Entry
		want    string
	}{
		{"v1.2.3", []Entry{fix}, "1.2.4"},
		{"1.2.3", []Entry{feat, fix}, "1.3.0"},
		{"v1.2.3", []Entry{breaking}, "2.0.0"},
		{"v0.4.1", []Entry{breaking}, "0.5.0"},
		{"v0.4.1", []Entry{feat}, "0.5.0"},
		{"v0.4.1", []Entry{hidden}, "0.4.2"},
		{"v0.1.0-alpha.1", []Entry{breaking}, "0.1.0"},
		{"release-3", []Entry{feat}, ""},
		{"", []Entry{feat}, ""},
	}
	for _, tt := range tests {
		if got := SuggestVersion(tt.tag, tt.entries); got != tt.want {
			t.Errorf("SuggestVersion(%q) = %q, want %q", tt.tag, got, tt.want)
		}
	}
}

func TestMarkdownUnreleasedHasNoDate(t *testing.T) {
	got := Markdown(Unreleased, "2026-09-27", []Entry{{Section: Added, Description: "x"}}, MarkdownOptions{})
	if want := "## [Unreleased]\n\n### Added\n\n- x\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

const existingChangelog = `# Changelog

Intro.

## [0.1.0] - 2026-01-01

### Added

- First release.

[0.1.0]: https://example.com/v0.1.0
`

func TestInsert(t *testing.T) {
	section := "## [0.2.0] - 2026-09-27\n\n### Fixed\n\n- x\n"

	t.Run("above the newest section", func(t *testing.T) {
		got, err := Insert(existingChangelog, "0.2.0", section, false)
		if err != nil {
			t.Fatal(err)
		}
		want := "# Changelog\n\nIntro.\n\n" + section + "\n" + existingChangelog[len("# Changelog\n\nIntro.\n\n"):]
		if got != want {
			t.Errorf("got\n%s\nwant\n%s", got, want)
		}
	})

	t.Run("same version refused", func(t *testing.T) {
		if _, err := Insert(existingChangelog, "0.1.0", section, false); !errors.Is(err, ErrSectionExists) {
			t.Errorf("err = %v, want ErrSectionExists", err)
		}
	})

	t.Run("same version replaced, link references kept", func(t *testing.T) {
		got, err := Insert(existingChangelog, "0.1.0", "## [0.1.0] - 2026-02-02\n\n- Replaced.\n", true)
		if err != nil {
			t.Fatal(err)
		}
		want := "# Changelog\n\nIntro.\n\n## [0.1.0] - 2026-02-02\n\n- Replaced.\n\n[0.1.0]: https://example.com/v0.1.0\n"
		if got != want {
			t.Errorf("got\n%s\nwant\n%s", got, want)
		}
	})

	t.Run("no sections yet", func(t *testing.T) {
		got, err := Insert("# Changelog\n\nIntro.\n", "0.2.0", section, false)
		if err != nil {
			t.Fatal(err)
		}
		if want := "# Changelog\n\nIntro.\n\n" + section; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("crlf kept", func(t *testing.T) {
		got, err := Insert("# Changelog\r\n\r\n## [0.1.0] - 2026-01-01\r\n", "0.2.0", section, false)
		if err != nil {
			t.Fatal(err)
		}
		want := "# Changelog\r\n\r\n## [0.2.0] - 2026-09-27\r\n\r\n### Fixed\r\n\r\n- x\r\n\r\n## [0.1.0] - 2026-01-01\r\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestWrite(t *testing.T) {
	section := "## [0.2.0] - 2026-09-27\n\n- x\n"

	t.Run("creates the file", func(t *testing.T) {
		dir := t.TempDir()
		if got, err := ExistingVersions(dir, section); err != nil || len(got) != 0 {
			t.Errorf("ExistingVersions without a file = %v, %v", got, err)
		}
		if err := Write(dir, section, false); err != nil {
			t.Fatal(err)
		}
		got, _ := os.ReadFile(filepath.Join(dir, FileName))
		if want := header + "\n" + section; string(got) != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("finds a lowercase file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "changelog.md")
		if err := os.WriteFile(path, []byte(existingChangelog), 0o644); err != nil {
			t.Fatal(err)
		}
		if got, err := ExistingVersions(dir, section+"\n## [0.1.0] - 2026-01-01\n"); err != nil || strings.Join(got, ",") != "0.1.0" {
			t.Errorf("ExistingVersions = %v, %v; want [0.1.0]", got, err)
		}
		if err := Write(dir, section, false); err != nil {
			t.Fatal(err)
		}
		got, _ := os.ReadFile(path)
		if want := "# Changelog\n\nIntro.\n\n" + section + "\n" + existingChangelog[len("# Changelog\n\nIntro.\n\n"):]; string(got) != want {
			t.Errorf("got\n%s", got)
		}
	})
}

func TestBuildSince(t *testing.T) {
	dir := gittest.InitRepo(t)
	commit(t, dir, "feat: old")
	gittest.Run(t, dir, "tag", "v1.0.0")
	commit(t, dir, "fix: also old")
	commitTime += 10 * 24 * 60 * 60
	since := fmt.Sprint(commitTime)
	commit(t, dir, "fix: recent")

	release, err := Build(context.Background(), dir, "", "HEAD", "@"+since)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(release.Entries) != 1 || release.Entries[0].Description != "recent" {
		t.Fatalf("entries = %+v, want only the recent commit", release.Entries)
	}
	if release.SuggestedVersion != "1.0.1" {
		t.Errorf("SuggestedVersion = %q, want 1.0.1 from the newest tag", release.SuggestedVersion)
	}
}

func TestMarkdownAuthorsAndDates(t *testing.T) {
	entries := []Entry{
		{Section: Added, Scope: "history", Description: "filter by type", Author: "Ada Lovelace", Date: "2026-09-20"},
		{Section: Fixed, Description: "keep lane color", Author: "Grace Hopper", CoAuthors: []string{"Ada Lovelace"}, Date: "2026-09-21"},
	}
	tests := []struct {
		name string
		opts MarkdownOptions
		want string
	}{
		{"neither", MarkdownOptions{}, "- **history:** filter by type"},
		{"authors", MarkdownOptions{Authors: true}, "- **history:** filter by type (Ada Lovelace)"},
		{"dates", MarkdownOptions{Dates: true}, "- **history:** filter by type (2026-09-20)"},
		{"both", MarkdownOptions{Authors: true, Dates: true}, "- **history:** filter by type (Ada Lovelace, 2026-09-20)"},
		{"co-authors", MarkdownOptions{Authors: true}, "- keep lane color (Grace Hopper, Ada Lovelace)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Markdown("1.0.0", "2026-09-27", entries, tt.opts)
			if !strings.Contains(got, tt.want+"\n") {
				t.Errorf("Markdown =\n%s\nwant a line %q", got, tt.want)
			}
		})
	}
}

func TestSectionFor(t *testing.T) {
	tests := []struct {
		commit git.ConventionalCommit
		want   string
	}{
		{git.ConventionalCommit{}, Other},
		{git.ConventionalCommit{Type: "docs", Breaking: true}, Breaking},
		{git.ConventionalCommit{Type: "feat"}, Added},
		{git.ConventionalCommit{Type: "revert"}, Changed},
		{git.ConventionalCommit{Type: "chore"}, "Chores"},
		{git.ConventionalCommit{Type: "content"}, "Content"},
	}
	for _, tt := range tests {
		if got := SectionFor(tt.commit); got != tt.want {
			t.Errorf("SectionFor(%+v) = %q, want %q", tt.commit, got, tt.want)
		}
	}
}

func TestMarkdownSectionOrder(t *testing.T) {
	got := Markdown(Unreleased, "", []Entry{
		{Section: Other, Description: "o"},
		{Section: "Design", Description: "d"},
		{Section: "Chores", Description: "c"},
		{Section: "Content", Description: "n"},
		{Section: Fixed, Description: "f"},
	}, MarkdownOptions{})
	var headings []string
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "### ") {
			headings = append(headings, strings.TrimPrefix(line, "### "))
		}
	}
	if want := "Fixed,Chores,Content,Design,Other"; strings.Join(headings, ",") != want {
		t.Errorf("sections = %v, want %s", headings, want)
	}
}

func TestWriteSeveralSections(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(existingChangelog), 0o644); err != nil {
		t.Fatal(err)
	}
	markdown := "## [Unreleased]\n\n- u\n\n## [0.3.0] - 2026-09-20\n\n- c\n\n## [0.2.0] - 2026-09-10\n\n- b\n"
	if err := Write(dir, markdown, false); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, FileName))
	want := "# Changelog\n\nIntro.\n\n" + markdown + "\n" + existingChangelog[len("# Changelog\n\nIntro.\n\n"):]
	if string(got) != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}

	if err := Write(dir, "## [0.2.0] - 2026-09-11\n\n- b2\n\n## [0.1.0] - x\n", false); !errors.Is(err, ErrSectionExists) {
		t.Errorf("rewriting existing versions without replace: err = %v", err)
	}
	if err := Write(dir, "no heading", false); !errors.Is(err, ErrNoSections) {
		t.Errorf("Markdown without a heading: err = %v", err)
	}
}

func TestBuildReleases(t *testing.T) {
	dir := gittest.InitRepo(t)
	commit(t, dir, "feat: one")
	gittest.Run(t, dir, "tag", "v0.1.0")
	commit(t, dir, "fix: two")
	gittest.Run(t, dir, "tag", "-a", "v0.2.0", "-m", "0.2.0")
	gittest.Run(t, dir, "tag", "same-commit")
	commit(t, dir, "feat: three")

	sections, err := BuildReleases(context.Background(), dir, "", "HEAD", "", true)
	if err != nil {
		t.Fatalf("BuildReleases: %v", err)
	}
	var got []string
	for _, s := range sections {
		got = append(got, fmt.Sprintf("%s/%s/%d", s.Tag, s.Version, len(s.Entries)))
		if s.Tag != "" && s.Date == "" {
			t.Errorf("section %s has no date", s.Tag)
		}
	}
	// same-commit shares v0.2.0's commit; the version tag names the release.
	if want := "//1,v0.2.0/0.2.0/1,v0.1.0/0.1.0/1"; strings.Join(got, ",") != want {
		t.Errorf("sections = %v, want %s", got, want)
	}
	if sections[0].SuggestedVersion == "" && sections[1].Tag == "v0.2.0" {
		t.Errorf("untagged section should suggest a version from v0.2.0")
	}

	single, err := BuildReleases(context.Background(), dir, "v0.1.0", "HEAD", "", false)
	if err != nil || len(single) != 1 || single[0].Tag != "" || len(single[0].Entries) != 2 {
		t.Errorf("unsplit = %+v, %v; want one untagged section of 2 commits", single, err)
	}
}

func TestMarkdownReleases(t *testing.T) {
	got := MarkdownReleases([]MarkdownSection{
		{Version: Unreleased, Entries: []Entry{{Section: Added, Description: "u"}}},
		{Version: "0.2.0", Date: "2026-09-20", Entries: []Entry{{Section: Fixed, Description: "f"}}},
	}, MarkdownOptions{})
	want := "## [Unreleased]\n\n### Added\n\n- u\n\n## [0.2.0] - 2026-09-20\n\n### Fixed\n\n- f\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestOnePerCommit(t *testing.T) {
	tags := []git.DatedTag{
		{Name: "deploy-42", SHA: "c"}, {Name: "v2.0.0", SHA: "c"},
		{Name: "v1.0.0", SHA: "b"}, {Name: "marker", SHA: "b"},
		{Name: "old", SHA: "a"},
	}
	var got []string
	for _, tag := range onePerCommit(tags) {
		got = append(got, tag.Name)
	}
	if want := "v2.0.0,v1.0.0,old"; strings.Join(got, ",") != want {
		t.Errorf("onePerCommit = %v, want %s", got, want)
	}
}

// A branch started before v1 but merged after it ships in v2, even though its
// commits are older than v1's tag.
func TestBuildReleasesAssignsMergedWorkToTheReleaseThatShipsIt(t *testing.T) {
	dir := gittest.InitRepo(t)
	commit(t, dir, "feat: base")
	gittest.Run(t, dir, "checkout", "-q", "-b", "early")
	commit(t, dir, "feat: early branch work")
	gittest.Run(t, dir, "checkout", "-q", "main")
	gittest.Run(t, dir, "merge", "-q", "--no-ff", "-m", "Merge early", "early")
	gittest.Run(t, dir, "checkout", "-q", "-b", "late", "main")
	commit(t, dir, "feat: late branch work")
	gittest.Run(t, dir, "checkout", "-q", "main")
	commit(t, dir, "fix: main before v1")
	gittest.Run(t, dir, "tag", "v1.0.0")
	gittest.Run(t, dir, "merge", "-q", "--no-ff", "-m", "Merge late", "late")
	gittest.Run(t, dir, "tag", "v2.0.0")

	sections, err := BuildReleases(context.Background(), dir, "", "HEAD", "", true)
	if err != nil {
		t.Fatalf("BuildReleases: %v", err)
	}
	got := map[string][]string{}
	for _, s := range sections {
		for _, e := range s.Entries {
			got[s.Tag] = append(got[s.Tag], e.Description)
		}
	}
	if want := []string{"late branch work"}; !reflect.DeepEqual(got["v2.0.0"], want) {
		t.Errorf("v2.0.0 = %v, want %v", got["v2.0.0"], want)
	}
	v1 := strings.Join(got["v1.0.0"], ",")
	for _, d := range []string{"base", "early branch work", "main before v1"} {
		if !strings.Contains(v1, d) {
			t.Errorf("v1.0.0 = %s, missing %q", v1, d)
		}
	}
	if strings.Contains(v1, "late branch work") || strings.Contains(v1, "Merge") {
		t.Errorf("v1.0.0 = %s, has late work or a merge", v1)
	}
	if _, untagged := got[""]; untagged {
		t.Error("HEAD is tagged, so there should be no untagged section")
	}
}

func TestBuildReadsCoAuthors(t *testing.T) {
	dir := gittest.InitRepo(t)
	commit(t, dir, "feat: pair work\n\nCo-authored-by: Grace Hopper <grace@example.com>")

	release, err := Build(context.Background(), dir, "", "HEAD", "")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(release.Entries) != 1 || !reflect.DeepEqual(release.Entries[0].CoAuthors, []string{"Grace Hopper"}) {
		t.Errorf("Entries = %+v, want one entry co-authored by Grace Hopper", release.Entries)
	}
}
