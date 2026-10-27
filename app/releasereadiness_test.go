package app

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func checkNamed(t *testing.T, r ReleaseReadiness, label string) ReleaseCheck {
	t.Helper()
	for _, c := range r.Checks {
		if c.Label == label {
			return c
		}
	}
	t.Fatalf("no %q check in %+v", label, r.Checks)
	return ReleaseCheck{}
}

func TestReleaseReadinessReportsEachItemWithItsEvidence(t *testing.T) {
	useTempSettings(t)
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q")
	write := func(name, text string) {
		os.MkdirAll(filepath.Dir(filepath.Join(repo, name)), 0o755)
		os.WriteFile(filepath.Join(repo, name), []byte(text), 0o644)
		gitIn(t, repo, "add", name)
	}
	commit := func(message string) {
		gitIn(t, repo, "-c", "user.name=T", "-c", "user.email=t@example.com", "commit", "-q", "--allow-empty", "-m", message)
	}
	write("build/config.yml", "version: 0.4.0\n")
	write("build/app.xml", "<v>0.4.0.0</v>\n")
	commit("chore: release 0.4.0")
	gitIn(t, repo, "tag", "-a", "v0.4.0", "-m", "0.4.0")
	commit("feat: issues")
	write("build/config.yml", "version: 0.5.0\n")
	write("CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n\n- Issues.\n")
	commit("chore: bump")
	var s ChangelogService

	r, err := s.ReleaseReadiness(repo, "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if r.Version != "0.5.0" || r.Previous != "v0.4.0" || r.Ref != "HEAD" {
		t.Fatalf("readiness = %+v", r)
	}
	if c := checkNamed(t, r, "Working tree"); c.Level != "ok" {
		t.Errorf("working tree = %+v", c)
	}
	if c := checkNamed(t, r, "Version files"); c.Level != "unknown" {
		t.Errorf("no files set = %+v", c)
	}
	if c := checkNamed(t, r, "Changelog"); c.Level != "warn" || len(c.Evidence) != 1 {
		t.Errorf("changelog with only Unreleased = %+v", c)
	}
	if c := checkNamed(t, r, "Commits since v0.4.0"); c.Level != "info" || len(c.Evidence) != 2 {
		t.Errorf("commits = %+v", c)
	}

	if err := s.SetVersionFiles(repo, []string{" build/config.yml", "build/app.xml", "", "missing.txt"}); err != nil {
		t.Fatal(err)
	}
	if files := s.VersionFiles(repo); !reflect.DeepEqual(files, []string{"build/config.yml", "build/app.xml", "missing.txt"}) {
		t.Fatalf("VersionFiles = %v", files)
	}
	os.WriteFile(filepath.Join(repo, "CHANGELOG.md"), []byte("# Changelog\n\n## [0.5.0] - 2026-11-01\n\n- Issues.\n"), 0o644)
	r, _ = s.ReleaseReadiness(repo, "HEAD", "v0.5.0", "main")
	if c := checkNamed(t, r, "build/config.yml"); c.Level != "ok" || !reflect.DeepEqual(c.Evidence, []string{"line 1: 0.5.0"}) {
		t.Errorf("config.yml = %+v", c)
	}
	if c := checkNamed(t, r, "build/app.xml"); c.Level != "warn" || c.Value != "No 0.5.0" || !reflect.DeepEqual(c.Evidence, []string{"line 1: still 0.4.0"}) {
		t.Errorf("app.xml = %+v", c)
	}
	if c := checkNamed(t, r, "missing.txt"); c.Level != "warn" {
		t.Errorf("missing file = %+v", c)
	}
	// The changelog is read at the candidate, so an uncommitted fix doesn't
	// count, and the working tree says it isn't clean.
	if c := checkNamed(t, r, "Changelog"); c.Level != "warn" {
		t.Errorf("uncommitted changelog = %+v", c)
	}
	if c := checkNamed(t, r, "Working tree"); c.Level != "warn" || !reflect.DeepEqual(c.Evidence, []string{"CHANGELOG.md"}) {
		t.Errorf("dirty tree = %+v", c)
	}

	gitIn(t, repo, "add", "CHANGELOG.md")
	commit("docs: 0.5.0")
	gitIn(t, repo, "tag", "-a", "v0.5.0", "-m", "0.5.0")
	commit("feat: more after the tag")
	r, _ = s.ReleaseReadiness(repo, "v0.4.0", "", "")
	if r.Version != "0.4.0" {
		t.Fatalf("a tagged candidate is its own tag's version: %+v", r)
	}
	r, _ = s.ReleaseReadiness(repo, "v0.5.0", "", "")
	if r.Previous != "v0.4.0" || r.Version != "0.5.0" {
		t.Fatalf("an already tagged candidate compares with the tag before: %+v", r)
	}
	if c := checkNamed(t, r, "Changelog"); c.Level != "ok" {
		t.Errorf("committed changelog = %+v", c)
	}
}

func TestVersionPatternMatchesTheWholeVersion(t *testing.T) {
	p := versionPattern("0.5.0")
	for text, want := range map[string]bool{"0.5.0": true, `"0.5.0.0"`: true, "v0.5.0,": true, "10.5.0": false, "1.0.5.0": false, "0.5.01": false} {
		if p.MatchString(text) != want {
			t.Errorf("%q: %v", text, !want)
		}
	}
}
