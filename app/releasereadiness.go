package app

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gspeager/current-client/core/changelog"
	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/internal/config"
)

// ReleaseCheck is one item of the release readiness list. Level is ok,
// warn, info or unknown; Evidence is what the verdict rests on.
type ReleaseCheck struct {
	Label    string   `json:"label"`
	Level    string   `json:"level"`
	Value    string   `json:"value"`
	Evidence []string `json:"evidence"`
}

type ReleaseReadiness struct {
	Ref      string         `json:"ref"`
	SHA      string         `json:"sha"`
	Version  string         `json:"version"`
	Previous string         `json:"previous"`
	Target   string         `json:"target"`
	Checks   []ReleaseCheck `json:"checks"`
}

const evidenceCap = 20

var versionTag = regexp.MustCompile(`^v?\d+\.\d+\.\d+`)

// ReleaseReadiness checks a candidate ref against the release before it.
// Empty version takes the one the changelog suggests; empty target, the
// default branch. Everything is read from Git, except CI and protection,
// which come from the host cache when the host is on. It publishes nothing.
func (s *ChangelogService) ReleaseReadiness(repoPath, ref, version, target string) (ReleaseReadiness, error) {
	ctx := context.Background()
	if ref == "" {
		ref = "HEAD"
	}
	sha, err := git.ResolveCommit(ctx, repoPath, ref)
	if err != nil {
		return ReleaseReadiness{}, err
	}
	r := ReleaseReadiness{Ref: ref, SHA: sha, Version: strings.TrimPrefix(strings.TrimSpace(version), "v"), Target: target}
	// A candidate that is already tagged is that tag's release, checked
	// against the tag before it.
	own := ""
	if r.Previous, err = git.LatestTag(ctx, repoPath, sha); err == nil && r.Previous != "" {
		if tagged, _ := git.ResolveCommit(ctx, repoPath, "refs/tags/"+r.Previous); tagged == sha {
			own = r.Previous
			r.Previous, _ = git.LatestTag(ctx, repoPath, sha+"^")
		}
	}
	release, err := changelog.Build(ctx, repoPath, r.Previous, sha, "")
	if err != nil {
		return ReleaseReadiness{}, err
	}
	if r.Version == "" && versionTag.MatchString(own) {
		r.Version = versionOf(own)
	}
	if r.Version == "" {
		r.Version = release.SuggestedVersion
	}
	if r.Target == "" {
		r.Target, _ = git.DefaultBranch(ctx, repoPath)
	}
	r.Checks = append(r.Checks, workingTreeCheck(ctx, repoPath))
	r.Checks = append(r.Checks, versionFileChecks(ctx, repoPath, sha, r.Version, versionOf(r.Previous))...)
	r.Checks = append(r.Checks, changelogCheck(ctx, repoPath, sha, r.Version), commitsCheck(r.Previous, release))
	r.Checks = append(r.Checks, releaseHostChecks(repoPath, sha, r.Target)...)
	return r, nil
}

func (s *ChangelogService) VersionFiles(repoPath string) []string {
	return configOrDefault().ReleaseVersionFiles[repoPath]
}

func (s *ChangelogService) SetVersionFiles(repoPath string, files []string) error {
	var kept []string
	for _, f := range files {
		if f = strings.TrimSpace(f); f != "" {
			kept = append(kept, filepath.ToSlash(f))
		}
	}
	return updateConfig(func(cfg *config.Config) {
		if cfg.ReleaseVersionFiles == nil {
			cfg.ReleaseVersionFiles = map[string][]string{}
		}
		if len(kept) == 0 {
			delete(cfg.ReleaseVersionFiles, repoPath)
			return
		}
		cfg.ReleaseVersionFiles[repoPath] = kept
	})
}

func versionOf(tag string) string {
	return strings.TrimPrefix(tag, "v")
}

func capped(lines []string) []string {
	if len(lines) <= evidenceCap {
		return lines
	}
	return append(lines[:evidenceCap:evidenceCap], fmt.Sprintf("and %d more", len(lines)-evidenceCap))
}

func workingTreeCheck(ctx context.Context, repoPath string) ReleaseCheck {
	check := ReleaseCheck{Label: "Working tree"}
	statuses, err := git.GetStatus(ctx, repoPath)
	switch {
	case err != nil:
		check.Level, check.Value = "unknown", err.Error()
	case len(statuses) == 0:
		check.Level, check.Value = "ok", "Clean"
	default:
		check.Level, check.Value = "warn", fmt.Sprintf("%d changed %s", len(statuses), pluralWord(len(statuses), "file", "files"))
		for _, st := range statuses {
			check.Evidence = append(check.Evidence, st.Path)
		}
		check.Evidence = capped(check.Evidence)
	}
	return check
}

func pluralWord(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// versionPattern finds version as a whole: 0.5.0 in "0.5.0" and "0.5.0.0",
// not in "10.5.0" or "1.0.5.0".
func versionPattern(version string) *regexp.Regexp {
	return regexp.MustCompile(`(^|[^0-9.])` + regexp.QuoteMeta(version) + `([^0-9]|$)`)
}

func linesWith(text string, pattern *regexp.Regexp) []int {
	var lines []int
	for i, line := range strings.Split(text, "\n") {
		if pattern.MatchString(line) {
			lines = append(lines, i+1)
		}
	}
	return lines
}

// versionFileChecks reads every configured file at the candidate in one
// process: each must name the version, and none may still name the last.
func versionFileChecks(ctx context.Context, repoPath, sha, version, previous string) []ReleaseCheck {
	files := configOrDefault().ReleaseVersionFiles[repoPath]
	if len(files) == 0 {
		return []ReleaseCheck{{Label: "Version files", Level: "unknown", Value: "None set for this repository"}}
	}
	if version == "" {
		return []ReleaseCheck{{Label: "Version files", Level: "unknown", Value: "No version to look for"}}
	}
	revs := make([]git.RevPath, len(files))
	for i, f := range files {
		revs[i] = git.RevPath{Rev: sha, Path: f}
	}
	contents, err := git.FilesAt(ctx, repoPath, revs)
	if err != nil {
		return []ReleaseCheck{{Label: "Version files", Level: "unknown", Value: err.Error()}}
	}
	checks := make([]ReleaseCheck, 0, len(files))
	for _, rev := range revs {
		check := ReleaseCheck{Label: rev.Path}
		data, ok := contents[rev]
		if !ok {
			check.Level, check.Value = "warn", "Not in the candidate"
			checks = append(checks, check)
			continue
		}
		text := strings.ReplaceAll(string(data), "\r\n", "\n")
		found := linesWith(text, versionPattern(version))
		var stale []int
		if previous != "" && previous != version {
			stale = linesWith(text, versionPattern(previous))
		}
		for _, n := range found {
			check.Evidence = append(check.Evidence, fmt.Sprintf("line %d: %s", n, version))
		}
		for _, n := range stale {
			check.Evidence = append(check.Evidence, fmt.Sprintf("line %d: still %s", n, previous))
		}
		check.Evidence = capped(check.Evidence)
		switch {
		case len(found) == 0:
			check.Level, check.Value = "warn", "No "+version
		case len(stale) > 0:
			check.Level, check.Value = "warn", "Has "+version+", and still "+previous
		default:
			check.Level, check.Value = "ok", "Has "+version
		}
		checks = append(checks, check)
	}
	return checks
}

// changelogCheck reads the changelog as committed at the candidate.
func changelogCheck(ctx context.Context, repoPath, sha, version string) ReleaseCheck {
	file := filepath.Base(changelog.Path(repoPath))
	check := ReleaseCheck{Label: "Changelog"}
	data, found, err := git.FileAt(ctx, repoPath, sha, file)
	switch {
	case err != nil:
		check.Level, check.Value = "unknown", err.Error()
	case !found:
		check.Level, check.Value = "warn", "No "+file+" in the candidate"
	case version == "":
		check.Level, check.Value = "unknown", "No version to look for"
	default:
		notes, ok := changelog.SectionNotes(string(data), version)
		if ok {
			check.Level, check.Value = "ok", file+" has a section for "+version
			if notes == "" {
				check.Level, check.Value = "warn", file+"'s section for "+version+" is empty"
			}
			return check
		}
		check.Level, check.Value = "warn", file+" has no section for "+version
		if _, unreleased := changelog.SectionNotes(string(data), changelog.Unreleased); unreleased {
			check.Evidence = []string{"It has [" + changelog.Unreleased + "], to rename to [" + version + "]"}
		}
	}
	return check
}

func commitsCheck(previous string, release changelog.Release) ReleaseCheck {
	since := "the first commit"
	if previous != "" {
		since = previous
	}
	check := ReleaseCheck{Label: "Commits since " + since}
	n := len(release.Entries)
	if n == 0 {
		check.Level, check.Value = "warn", "None, so nothing to release"
		return check
	}
	check.Level, check.Value = "info", fmt.Sprintf("%d %s, merges left out", n, pluralWord(n, "commit", "commits"))
	for _, e := range release.Entries {
		check.Evidence = append(check.Evidence, e.SHA[:min(7, len(e.SHA))]+" "+e.Description)
	}
	check.Evidence = capped(check.Evidence)
	return check
}
