package changelog

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const FileName = "CHANGELOG.md"

const header = "# Changelog\n\nAll notable changes to this project are listed here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).\n"

var ErrSectionExists = errors.New("the changelog already has a section for this version")

var (
	versionHeading = regexp.MustCompile(`(?m)^## `)
	linkReferences = regexp.MustCompile(`(?m)^\[[^\]]+\]: \S`)
)

// Insert puts section above the newest existing one. A section for the same
// version is replaced only when replace is set; the link references that
// close a Keep a Changelog file stay at the end.
func Insert(existing, version, section string, replace bool) (string, error) {
	crlf := strings.Contains(existing, "\r\n")
	text := strings.ReplaceAll(existing, "\r\n", "\n")
	section = strings.TrimRight(section, "\n") + "\n\n"
	if strings.TrimSpace(text) == "" {
		text = header + "\n"
	}

	start, end := sectionBounds(text, version)
	if start >= 0 && !replace {
		return "", ErrSectionExists
	}
	if start < 0 {
		start, end = insertionPoint(text), insertionPoint(text)
	}
	out := text[:start]
	if out != "" && !strings.HasSuffix(out, "\n\n") {
		out = strings.TrimRight(out, "\n") + "\n\n"
	}
	out += section + text[end:]
	out = strings.TrimRight(out, "\n") + "\n"
	if crlf {
		out = strings.ReplaceAll(out, "\n", "\r\n")
	}
	return out, nil
}

func sectionBounds(text, version string) (int, int) {
	heading := regexp.MustCompile(`(?m)^## \[?` + regexp.QuoteMeta(version) + `\]?(\s|$)`)
	loc := heading.FindStringIndex(text)
	if loc == nil {
		return -1, -1
	}
	end := len(text)
	if next := versionHeading.FindStringIndex(text[loc[1]:]); next != nil {
		end = loc[1] + next[0]
	} else if refs := linkReferences.FindStringIndex(text[loc[1]:]); refs != nil {
		end = loc[1] + refs[0]
	}
	return loc[0], end
}

func insertionPoint(text string) int {
	if loc := versionHeading.FindStringIndex(text); loc != nil {
		return loc[0]
	}
	if loc := linkReferences.FindStringIndex(text); loc != nil {
		return loc[0]
	}
	return len(text)
}

// Path finds the repository's changelog whatever its case, or names a new
// CHANGELOG.md when there is none.
func Path(repoPath string) string {
	entries, _ := os.ReadDir(repoPath)
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(e.Name(), FileName) {
			return filepath.Join(repoPath, e.Name())
		}
	}
	return filepath.Join(repoPath, FileName)
}

var ErrNoSections = errors.New(`the Markdown has no "## " version heading`)

var sectionHeading = regexp.MustCompile(`(?m)^## \[?([^\]\s]+)\]?`)

type versionedSection struct {
	version string
	text    string
}

// splitSections cuts Markdown at its "## " version headings; anything before
// the first heading is dropped.
func splitSections(markdown string) []versionedSection {
	text := strings.ReplaceAll(markdown, "\r\n", "\n")
	locs := sectionHeading.FindAllStringSubmatchIndex(text, -1)
	sections := make([]versionedSection, len(locs))
	for i, loc := range locs {
		end := len(text)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		sections[i] = versionedSection{version: text[loc[2]:loc[3]], text: text[loc[0]:end]}
	}
	return sections
}

func readChangelog(repoPath string) (string, error) {
	existing, err := os.ReadFile(Path(repoPath))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	return string(existing), err
}

// Write inserts every section of markdown, oldest first so the newest ends up
// on top.
func Write(repoPath, markdown string, replace bool) error {
	sections := splitSections(markdown)
	if len(sections) == 0 {
		return ErrNoSections
	}
	out, err := readChangelog(repoPath)
	if err != nil {
		return err
	}
	for i := len(sections) - 1; i >= 0; i-- {
		if out, err = Insert(out, sections[i].version, sections[i].text, replace); err != nil {
			return err
		}
	}
	return os.WriteFile(Path(repoPath), []byte(out), 0o644)
}

// ExistingVersions lists the versions in markdown that the changelog already
// has a section for.
func ExistingVersions(repoPath, markdown string) ([]string, error) {
	existing, err := readChangelog(repoPath)
	if err != nil {
		return nil, err
	}
	existing = strings.ReplaceAll(existing, "\r\n", "\n")
	var versions []string
	for _, s := range splitSections(markdown) {
		if start, _ := sectionBounds(existing, s.version); start >= 0 {
			versions = append(versions, s.version)
		}
	}
	return versions, nil
}
