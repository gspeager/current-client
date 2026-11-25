package risk

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/gspeager/current-client/core/filekind"
)

// A test as each language declares one, and how it's told not to run.
type testSyntax struct {
	declared *regexp.Regexp
	skipped  *regexp.Regexp
	// Tests sit beside code in the same file, so any file can have them.
	anywhere bool
}

var (
	goTests = &testSyntax{
		declared: regexp.MustCompile(`^func (Test|Fuzz|Example)(\p{Lu}|_|\()`),
		skipped:  regexp.MustCompile(`\b[tbf]\.(Skip|Skipf|SkipNow)\(`),
	}
	scriptTests = &testSyntax{
		declared: regexp.MustCompile(`^\s*[xf]?(it|test)(\.(each|only|skip|todo|concurrent|sequential|failing|fails|skipIf|runIf|for))*(\(|\x60)`),
		skipped:  regexp.MustCompile(`\b(it|test|describe|suite)\.(skip|todo|only|skipIf|failing)\b|^\s*(xit|xtest|xdescribe|fit|fdescribe)\(`),
	}
	pythonTests = &testSyntax{
		declared: regexp.MustCompile(`^\s*(async )?def test_?\w*\(`),
		skipped:  regexp.MustCompile(`@(pytest\.mark|unittest)\.(skip|skipif|skipIf|skipUnless|xfail|expectedFailure)\b|\bpytest\.(skip|xfail)\(|\bself\.skipTest\(`),
	}
	jvmTests = &testSyntax{
		declared: regexp.MustCompile(`^\s*@(Test|ParameterizedTest|RepeatedTest|TestFactory)\b`),
		skipped:  regexp.MustCompile(`^\s*@(Disabled|Ignore)\b`),
	}
	dotnetTests = &testSyntax{
		declared: regexp.MustCompile(`^\s*\[(Fact|Theory|Test|TestCase|TestMethod)\b`),
		skipped:  regexp.MustCompile(`^\s*\[((Fact|Theory)\(.*\bSkip\s*=|Ignore\b)`),
	}
	rustTests = &testSyntax{
		declared: regexp.MustCompile(`^\s*#\[(\w+::)*test\b`),
		skipped:  regexp.MustCompile(`^\s*#\[ignore\b`),
		anywhere: true,
	}
	rubyTests = &testSyntax{
		declared: regexp.MustCompile(`^\s*(it|specify|test)[ (]["']|^\s*def test_`),
		skipped:  regexp.MustCompile(`^\s*(xit|xdescribe|xcontext|pending|skip)\b`),
	}
	swiftTests = &testSyntax{
		declared: regexp.MustCompile(`^\s*func test\w*\(|^\s*@Test\b`),
		skipped:  regexp.MustCompile(`\bXCTSkip|@Test\(.*\.disabled\(`),
	}
	phpTests = &testSyntax{
		declared: regexp.MustCompile(`\bfunction test\w*\(|#\[Test\]`),
		skipped:  regexp.MustCompile(`\bmarkTest(Skipped|Incomplete)\(`),
	}
)

var testSyntaxes = map[string]*testSyntax{
	".go": goTests,
	".js": scriptTests, ".jsx": scriptTests, ".mjs": scriptTests, ".cjs": scriptTests,
	".ts": scriptTests, ".tsx": scriptTests, ".mts": scriptTests, ".cts": scriptTests,
	".py":   pythonTests,
	".java": jvmTests, ".kt": jvmTests, ".kts": jvmTests, ".scala": jvmTests, ".groovy": jvmTests,
	".cs": dotnetTests, ".fs": dotnetTests, ".vb": dotnetTests,
	".rs":    rustTests,
	".rb":    rubyTests,
	".swift": swiftTests,
	".php":   phpTests,
}

// testSyntaxOf answers for a file once, since it's asked about every line.
func testSyntaxOf() func(file string) *testSyntax {
	known := map[string]*testSyntax{}
	return func(file string) *testSyntax {
		syntax, seen := known[file]
		if !seen {
			syntax = testSyntaxes[path.Ext(file)]
			if syntax != nil && !syntax.anywhere && filekind.Of(file) != filekind.Test {
				syntax = nil
			}
			known[file] = syntax
		}
		return syntax
	}
}

// fewerTests counts over the whole change, so moving or renaming a test isn't
// losing one. It names the files that lost some.
func fewerTests(lines changedLines) []Finding {
	lost := map[string]int{}
	first := map[string]int{}
	var order []string
	total := 0
	syntaxOf := testSyntaxOf()
	count := func(all []addedLine, by int) {
		for _, line := range all {
			syntax := syntaxOf(line.path)
			if syntax == nil || !syntax.declared.MatchString(line.text) {
				continue
			}
			if by > 0 && first[line.path] == 0 {
				first[line.path] = line.line
				order = append(order, line.path)
			}
			lost[line.path] += by
			total += by
		}
	}
	count(lines.removed, 1)
	count(lines.added, -1)
	if total <= 0 {
		return nil
	}
	var findings []Finding
	for _, file := range order {
		// Another file may have gained some of what this one lost.
		if fewer := min(lost[file], total); fewer > 0 {
			total -= fewer
			findings = append(findings, Finding{
				Rule: RuleFewerTests, Path: file, OldLine: first[file], Message: fmt.Sprintf("%s fewer.", tests(fewer)),
			})
		}
	}
	return findings
}

func tests(n int) string {
	if n == 1 {
		return "1 test"
	}
	return fmt.Sprintf("%d tests", n)
}

// A focused test leaves the file's others unrun.
var focused = regexp.MustCompile(`\.only\b|^\s*f(it|describe)\(`)

func skippedTests(lines changedLines) []Finding {
	var findings []Finding
	syntaxOf := testSyntaxOf()
	for _, line := range lines.newlyAdded() {
		if syntax := syntaxOf(line.path); syntax != nil && syntax.skipped.MatchString(line.text) {
			what := "Test skipped: "
			if focused.MatchString(line.text) {
				what = "Other tests skipped: "
			}
			findings = append(findings, Finding{Rule: RuleSkippedTests, Path: line.path, Line: line.line, Message: what + snippet(line.text)})
		}
	}
	return findings
}

// What each checker reads, as it's written in a comment or an attribute.
var silenced = regexp.MustCompile(`(?:(?://|/\*|#|--|<!--|\{/\*)\s*(` +
	`eslint-disable(?:-next-line|-line)?|stylelint-disable\S*|biome-ignore|tslint:disable\S*|oxlint-disable\S*|` +
	`@ts-ignore|@ts-expect-error|@ts-nocheck|` +
	`nolint|lint:ignore|lint:file-ignore|#nosec|nosec|NOSONAR|` +
	`noqa|type: ?ignore|pylint: ?disable|pyright: ?ignore|mypy: ?ignore-errors|ruff: ?noqa|` +
	`rubocop: ?disable|shellcheck disable|hadolint ignore|swiftlint:disable\S*|phpcs:(?:ignore|disable)|@phpstan-ignore\S*|@psalm-suppress|` +
	`checkov:skip|tfsec:ignore|trivy:ignore|codeql\[|semgrep: ?ignore|nosemgrep)` +
	`|(#pragma warning (?:disable|\(disable))` +
	`|(@SuppressWarnings|@Suppress\(|@SuppressLint|\[SuppressMessage|#!?\[allow\())`)

// Every pattern above holds one of these, and few lines do.
var silencedWords = []string{
	"isable", "gnore", "uppress", "nolint", "nosec", "nosem", "noqa", "NOSONAR", "nocheck", "expect-error", "skip", "allow(", "codeql[",
}

func mayBeSilenced(text string) bool {
	return slices.ContainsFunc(silencedWords, func(word string) bool { return strings.Contains(text, word) })
}

// A checker is often written about, too.
var prose = regexp.MustCompile(`(?i)\.(md|mdx|markdown|txt|rst|adoc)$`)

func silencedCheckers(lines changedLines) []Finding {
	var findings []Finding
	for _, line := range lines.newlyAdded() {
		if !mayBeSilenced(line.text) || len(line.text) > longestLineScanned || prose.MatchString(line.path) || filekind.IsGenerated(line.path) {
			continue
		}
		if silenced.MatchString(line.text) {
			findings = append(findings, Finding{Rule: RuleSilencedChecker, Path: line.path, Line: line.line, Message: "Checker silenced: " + snippet(line.text)})
		}
	}
	return findings
}

const longestSnippet = 80

func snippet(line string) string {
	text := []rune(strings.TrimSpace(line))
	if len(text) > longestSnippet {
		return string(text[:longestSnippet-1]) + "…"
	}
	return string(text)
}
