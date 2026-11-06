package risk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

// Built up so no scanner flags this file, including this project's own report.
const fakeGitHubToken = "gh" + "p_" + "aB3dE6gH9jK2mN5pQ8sT1vW4yZ7bC0eF3hI6"

func stage(t *testing.T, dir, name, content string) {
	t.Helper()
	full := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	gittest.Run(t, dir, "add", "--", name)
}

func newRepo(t *testing.T) string {
	t.Helper()
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "README.md", "hello\n", "initial")
	return dir
}

func check(t *testing.T, dir string, off ...string) []Finding {
	t.Helper()
	findings, err := Check(context.Background(), dir, off)
	if err != nil {
		t.Fatal(err)
	}
	return findings
}

func withRule(findings []Finding, rule string) []Finding {
	var matched []Finding
	for _, finding := range findings {
		if finding.Rule == rule {
			matched = append(matched, finding)
		}
	}
	return matched
}

func TestSecretIsReportedMaskedOnItsLine(t *testing.T) {
	dir := newRepo(t)
	stage(t, dir, "app/config.go", "package app\n\nconst token = \""+fakeGitHubToken+"\"\n")

	findings := withRule(check(t, dir), RuleSecret)
	if len(findings) != 1 || findings[0].Path != "app/config.go" || findings[0].Line != 3 {
		t.Fatalf("findings = %+v", findings)
	}
	if strings.Contains(findings[0].Message, fakeGitHubToken) || strings.Contains(findings[0].Message, fakeGitHubToken[4:]) {
		t.Fatalf("value not masked: %q", findings[0].Message)
	}
	if !strings.Contains(findings[0].Message, "github pat") || !strings.Contains(findings[0].Message, "ghp_…") {
		t.Fatalf("message = %q", findings[0].Message)
	}
}

func TestSecretsLeaveCommonFalsePositivesAlone(t *testing.T) {
	cases := []struct{ name, path, content string }{
		{"a commit hash", "notes.txt", "fixed in a94a8fe5ccb19ba61c4c0873d391e987982fbbd3\n"},
		{"a checksum", "SHA256SUMS", "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08  app.zip\n"},
		{"a test fixture", "core/testdata/tokens.txt", fakeGitHubToken + "\n"},
		{"a fixture folder", "web/__fixtures__/auth.json", fakeGitHubToken + "\n"},
		{"AWS's documented example key", "docs/aws.md", "AKIAIOSFODNN7EXAMPLE\n"},
		{"a variable name", "app.ts", "const apiKey = getApiKey(config)\n"},
		{"a line too long to be pasted by hand", "bundle.js", strings.Repeat("x", longestLineScanned) + fakeGitHubToken + "\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := newRepo(t)
			stage(t, dir, c.path, c.content)
			if findings := withRule(check(t, dir), RuleSecret); len(findings) != 0 {
				t.Fatalf("findings = %+v", findings)
			}
		})
	}
}

func TestOnlyStagedAdditionsAreRead(t *testing.T) {
	dir := newRepo(t)
	gittest.CommitFile(t, dir, "old.txt", fakeGitHubToken+"\n", "an old secret")
	stage(t, dir, "old.txt", "cleaned up\n")
	if err := os.WriteFile(filepath.Join(dir, "unstaged.txt"), []byte(fakeGitHubToken), 0o644); err != nil {
		t.Fatal(err)
	}
	if findings := check(t, dir); len(findings) != 0 {
		t.Fatalf("findings = %+v", findings)
	}
}

func TestConflictMarkerLeftIn(t *testing.T) {
	dir := newRepo(t)
	stage(t, dir, "a.txt", "one\n<<<<<<< HEAD\ntwo\n=======\nthree\n>>>>>>> side\n")

	findings := withRule(check(t, dir), RuleConflictMarker)
	if len(findings) != 2 || findings[0].Line != 2 || findings[1].Line != 6 {
		t.Fatalf("findings = %+v", findings)
	}
}

func TestLargeFileIsFlaggedAndItsContentSkipped(t *testing.T) {
	dir := newRepo(t)
	stage(t, dir, "data.bin", fakeGitHubToken+"\n"+strings.Repeat("a\n", largeFileBytes/2))

	findings := check(t, dir)
	if large := withRule(findings, RuleLargeFile); len(large) != 1 || large[0].Path != "data.bin" || large[0].Line != 0 {
		t.Fatalf("findings = %+v", findings)
	}
	if secrets := withRule(findings, RuleSecret); len(secrets) != 0 {
		t.Fatalf("a large file's content should be left out: %+v", secrets)
	}
}

func TestLockfileOutOfStepWithItsManifest(t *testing.T) {
	const manifest = `{"dependencies":{"a":"1.0.0"}}`
	const bumped = `{"dependencies":{"a":"1.1.0"}}`
	setup := func(t *testing.T) string {
		dir := newRepo(t)
		stage(t, dir, "web/package.json", manifest)
		stage(t, dir, "web/package-lock.json", `{"packages":{}}`)
		gittest.Run(t, dir, "commit", "-m", "manifest and lockfile")
		return dir
	}

	t.Run("manifest without lockfile", func(t *testing.T) {
		dir := setup(t)
		stage(t, dir, "web/package.json", bumped)
		findings := withRule(check(t, dir), RuleLockfile)
		if len(findings) != 1 || findings[0].Path != "web/package.json" || !strings.Contains(findings[0].Message, "package-lock.json") {
			t.Fatalf("findings = %+v", findings)
		}
	})
	t.Run("lockfile without manifest", func(t *testing.T) {
		dir := setup(t)
		stage(t, dir, "web/package-lock.json", `{"packages":{"node_modules/b":{"version":"1"}}}`)
		findings := withRule(check(t, dir), RuleLockfile)
		if len(findings) != 1 || findings[0].Path != "web/package-lock.json" || !strings.Contains(findings[0].Message, "package.json") {
			t.Fatalf("findings = %+v", findings)
		}
	})
	t.Run("both", func(t *testing.T) {
		dir := setup(t)
		stage(t, dir, "web/package.json", bumped)
		stage(t, dir, "web/package-lock.json", `{"packages":{"node_modules/a":{"version":"1.1.0"}}}`)
		if findings := withRule(check(t, dir), RuleLockfile); len(findings) != 0 {
			t.Fatalf("findings = %+v", findings)
		}
	})
	t.Run("manifest change that isn't a package", func(t *testing.T) {
		dir := setup(t)
		stage(t, dir, "web/package.json", `{"version":"2","dependencies":{"a":"1.0.0"}}`)
		if findings := withRule(check(t, dir), RuleLockfile); len(findings) != 0 {
			t.Fatalf("findings = %+v", findings)
		}
	})
	t.Run("manifest with no lockfile in the repository", func(t *testing.T) {
		dir := newRepo(t)
		gittest.CommitFile(t, dir, "Cargo.toml", "[dependencies]\na = \"1\"\n", "manifest")
		stage(t, dir, "Cargo.toml", "[dependencies]\na = \"2\"\n")
		if findings := withRule(check(t, dir), RuleLockfile); len(findings) != 0 {
			t.Fatalf("findings = %+v", findings)
		}
	})
}

func TestSweepingChangeToGeneratedFiles(t *testing.T) {
	dir := newRepo(t)
	for i := range generatedFiles - 1 {
		stage(t, dir, fmt.Sprintf("dist/%d.js", i), "x\n")
	}
	if findings := withRule(check(t, dir), RuleGenerated); len(findings) != 0 {
		t.Fatalf("below the threshold: %+v", findings)
	}
	stage(t, dir, "app.min.js", "x\n")
	findings := withRule(check(t, dir), RuleGenerated)
	if len(findings) != 1 || !strings.Contains(findings[0].Message, fmt.Sprint(generatedFiles)) {
		t.Fatalf("findings = %+v", findings)
	}
}

func TestARuleCanBeTurnedOff(t *testing.T) {
	dir := newRepo(t)
	stage(t, dir, "a.txt", "<<<<<<< HEAD\n"+fakeGitHubToken+"\n")
	if findings := check(t, dir, RuleSecret, RuleConflictMarker); len(findings) != 0 {
		t.Fatalf("findings = %+v", findings)
	}
	if findings := check(t, dir, RuleSecret); len(findings) != 1 || findings[0].Rule != RuleConflictMarker {
		t.Fatalf("findings = %+v", findings)
	}
}

func TestNothingStagedHasNoFindings(t *testing.T) {
	if findings := check(t, newRepo(t)); len(findings) != 0 {
		t.Fatalf("findings = %+v", findings)
	}
}

func TestANameWithSpecialCharactersKeepsItsPath(t *testing.T) {
	dir := newRepo(t)
	stage(t, dir, "café \"x\".txt", "<<<<<<< HEAD\n")
	findings := check(t, dir)
	if len(findings) != 1 || findings[0].Path != "café \"x\".txt" {
		t.Fatalf("findings = %+v", findings)
	}
}

func TestAllRulesCompile(t *testing.T) {
	if count := len(loadSecretRules().rules); count < 200 {
		t.Fatalf("only %d of gitleaks's rules loaded", count)
	}
}

func TestMaskNeverShowsAShortValueOrAWholeOne(t *testing.T) {
	if got := mask("short"); got != "…" {
		t.Fatalf("mask(short) = %q", got)
	}
	if got := mask("abcdefghijklmnop"); got != "abcd…" {
		t.Fatalf("mask(long) = %q", got)
	}
}

func BenchmarkScanLines(b *testing.B) {
	lines := make([]addedLine, 10000)
	for i := range lines {
		lines[i] = addedLine{"app.go", i + 1, fmt.Sprintf("func handler%d(apiKey string) error { return nil }", i)}
	}
	for b.Loop() {
		scanLines(lines, true, true)
	}
}
