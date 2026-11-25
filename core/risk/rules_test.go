package risk

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/gitexec"
	"github.com/gspeager/current-client/core/internal/gittest"
)

// Each case is a commit and then a change to it, checked when staged and as a
// range once committed. "" deletes the file.
type ruleCase struct {
	name   string
	before map[string]string
	after  map[string]string
	want   []Finding
}

func checkBoth(t *testing.T, rule string, cases []ruleCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := newRepo(t)
			write := func(files map[string]string) {
				for _, name := range slices.Sorted(maps.Keys(files)) {
					if files[name] == "" {
						gittest.Run(t, dir, "rm", "-q", "--", name)
					} else {
						stage(t, dir, name, files[name])
					}
				}
			}
			write(c.before)
			gittest.Run(t, dir, "commit", "-q", "--allow-empty", "-m", "before")
			write(c.after)
			want := mapped(c.want, rule)
			staged := withRule(check(t, dir), rule)
			if !slices.Equal(staged, want) {
				t.Errorf("staged:\n got %+v\nwant %+v", staged, want)
			}
			gittest.Run(t, dir, "commit", "-q", "-m", "after")
			ranged := withRule(checkRange(t, dir, "HEAD~1", "HEAD").Findings, rule)
			if !slices.Equal(ranged, want) {
				t.Errorf("range:\n got %+v\nwant %+v", ranged, want)
			}
			if off := checkRange(t, dir, "HEAD~1", "HEAD", rule).Findings; len(withRule(off, rule)) != 0 {
				t.Errorf("turned off: %+v", off)
			}
		})
	}
}

func mapped(findings []Finding, rule string) []Finding {
	var all []Finding
	for _, finding := range findings {
		finding.Rule = rule
		all = append(all, finding)
	}
	return all
}

func checkRange(t *testing.T, dir, from, to string, off ...string) Report {
	t.Helper()
	report, err := CheckRange(context.Background(), dir, Range{From: from, To: to}, off)
	if err != nil {
		t.Fatal(err)
	}
	return report
}

func TestFewerTests(t *testing.T) {
	const two = "package a\n\nfunc TestOne(t *testing.T) {}\n\nfunc TestTwo(t *testing.T) {}\n"
	const one = "package a\n\nfunc TestOne(t *testing.T) {}\n"
	checkBoth(t, RuleFewerTests, []ruleCase{
		{"one removed", map[string]string{"a_test.go": two}, map[string]string{"a_test.go": one},
			[]Finding{{Path: "a_test.go", OldLine: 5, Message: "1 test fewer."}}},
		{"a test file deleted", map[string]string{"a_test.go": two}, map[string]string{"a_test.go": ""},
			[]Finding{{Path: "a_test.go", OldLine: 3, Message: "2 tests fewer."}}},
		{"a test renamed", map[string]string{"a_test.go": two}, map[string]string{"a_test.go": strings.Replace(two, "TestTwo", "TestThree", 1)}, nil},
		{"a test moved", map[string]string{"a_test.go": two}, map[string]string{"a_test.go": one, "b_test.go": "package a\n\nfunc TestTwo(t *testing.T) {}\n"}, nil},
		{"a test file moved", map[string]string{"a_test.go": two}, map[string]string{"a_test.go": "", "b/a_test.go": two}, nil},
		{"not a test file", map[string]string{"a.go": two}, map[string]string{"a.go": one}, nil},
		{"TypeScript", map[string]string{"a.test.ts": "test('a', () => {})\nit.each([1])('b', () => {})\n"}, map[string]string{"a.test.ts": "test('a', () => {})\n"},
			[]Finding{{Path: "a.test.ts", OldLine: 2, Message: "1 test fewer."}}},
		{"Rust, beside its code", map[string]string{"src/lib.rs": "#[test]\nfn a() {}\n#[tokio::test]\nasync fn b() {}\n"}, map[string]string{"src/lib.rs": "#[test]\nfn a() {}\n"},
			[]Finding{{Path: "src/lib.rs", OldLine: 3, Message: "1 test fewer."}}},
	})
}

func TestSkippedTests(t *testing.T) {
	checkBoth(t, RuleSkippedTests, []ruleCase{
		{"Go", map[string]string{"a_test.go": "package a\n"}, map[string]string{"a_test.go": "package a\n\tt.Skip(\"later\")\n"},
			[]Finding{{Path: "a_test.go", Line: 2, Message: `Test skipped: t.Skip("later")`}}},
		{"TypeScript", map[string]string{"a.test.ts": "it('a', f)\n"}, map[string]string{"a.test.ts": "it.skip('a', f)\n"},
			[]Finding{{Path: "a.test.ts", Line: 1, Message: "Test skipped: it.skip('a', f)"}}},
		{"one focused", map[string]string{"a.test.ts": "it('a', f)\n"}, map[string]string{"a.test.ts": "it.only('a', f)\n"},
			[]Finding{{Path: "a.test.ts", Line: 1, Message: "Other tests skipped: it.only('a', f)"}}},
		{"Python", map[string]string{"test_a.py": "x\n"}, map[string]string{"test_a.py": "@pytest.mark.skip(reason='x')\nx\n"},
			[]Finding{{Path: "test_a.py", Line: 1, Message: "Test skipped: @pytest.mark.skip(reason='x')"}}},
		{"already skipped, and moved", map[string]string{"a.test.ts": "it.skip('a', f)\n"}, map[string]string{"a.test.ts": "", "b.test.ts": "x\nit.skip('a', f)\n"}, nil},
		{"one no longer skipped", map[string]string{"a.test.ts": "it.skip('a', f)\n"}, map[string]string{"a.test.ts": "it('a', f)\n"}, nil},
		{"not a test file", map[string]string{"a.ts": "x\n"}, map[string]string{"a.ts": "it.skip('a', f)\n"}, nil},
	})
}

func TestSilencedChecker(t *testing.T) {
	checkBoth(t, RuleSilencedChecker, []ruleCase{
		{"ESLint", map[string]string{"a.ts": "x\n"}, map[string]string{"a.ts": "// eslint-disable-next-line no-console\nx\n"},
			[]Finding{{Path: "a.ts", Line: 1, Message: "Checker silenced: // eslint-disable-next-line no-console"}}},
		{"TypeScript", map[string]string{"a.ts": "x\n"}, map[string]string{"a.ts": "x // @ts-ignore\n"},
			[]Finding{{Path: "a.ts", Line: 1, Message: "Checker silenced: x // @ts-ignore"}}},
		{"Go", map[string]string{"a.go": "x\n"}, map[string]string{"a.go": "x() //nolint:errcheck\n"},
			[]Finding{{Path: "a.go", Line: 1, Message: "Checker silenced: x() //nolint:errcheck"}}},
		{"C#", map[string]string{"a.cs": "x\n"}, map[string]string{"a.cs": "#pragma warning disable CS0168\n"},
			[]Finding{{Path: "a.cs", Line: 1, Message: "Checker silenced: #pragma warning disable CS0168"}}},
		{"Python", map[string]string{"a.py": "x\n"}, map[string]string{"a.py": "import x  # noqa: F401\n"},
			[]Finding{{Path: "a.py", Line: 1, Message: "Checker silenced: import x  # noqa: F401"}}},
		{"already there, and moved", map[string]string{"a.ts": "// @ts-ignore\nx\n"}, map[string]string{"a.ts": "", "b.ts": "y\n// @ts-ignore\nx\n"}, nil},
		{"taken out", map[string]string{"a.ts": "// @ts-ignore\nx\n"}, map[string]string{"a.ts": "x\n"}, nil},
		{"written about", map[string]string{"a.md": "x\n"}, map[string]string{"a.md": "Use // eslint-disable-line here.\n"}, nil},
		{"a name that holds it", map[string]string{"a.ts": "x\n"}, map[string]string{"a.ts": "const eslintDisabled = 'eslint-disable'\n"}, nil},
	})
}

func TestPathRules(t *testing.T) {
	one := func(path string, content string) map[string]string { return map[string]string{path: content} }
	t.Run("CI files", func(t *testing.T) {
		checkBoth(t, RuleCI, []ruleCase{
			{"changed", one(".github/workflows/ci.yml", "a\n"), one(".github/workflows/ci.yml", "b\n"), []Finding{{Path: ".github/workflows/ci.yml", Message: "CI file changed."}}},
			{"added", nil, one(".gitlab-ci.yml", "a\n"), []Finding{{Path: ".gitlab-ci.yml", Message: "CI file added."}}},
			{"deleted", one("Jenkinsfile", "a\n"), one("Jenkinsfile", ""), []Finding{{Path: "Jenkinsfile", Message: "CI file deleted."}}},
			{"another file there", nil, one(".github/ISSUE_TEMPLATE/bug.yml", "a\n"), nil},
			{"one in a folder of samples", nil, one("docs/.gitlab-ci.yml", "a\n"), nil},
		})
	})
	t.Run("agent instructions", func(t *testing.T) {
		checkBoth(t, RuleAgentInstructions, []ruleCase{
			{"changed", one("AGENTS.md", "a\n"), one("AGENTS.md", "b\n"), []Finding{{Path: "AGENTS.md", Message: "Agent instructions changed."}}},
			{"added in a folder", nil, one("web/.cursor/rules/a.mdc", "a\n"), []Finding{{Path: "web/.cursor/rules/a.mdc", Message: "Agent instructions added."}}},
			{"a like name", nil, one("docs/AGENTS.md.txt", "a\n"), nil},
		})
	})
	t.Run("migrations", func(t *testing.T) {
		checkBoth(t, RuleMigration, []ruleCase{
			{"added", nil, one("db/migrate/20260101_add_users.rb", "a\n"), []Finding{{Path: "db/migrate/20260101_add_users.rb", Message: "Database migration added."}}},
			{"changed", one("app/migrations/0001_initial.py", "a\n"), one("app/migrations/0001_initial.py", "b\n"), []Finding{{Path: "app/migrations/0001_initial.py", Message: "Database migration changed."}}},
			{"Flyway", nil, one("src/main/resources/sql/V2__users.sql", "a\n"), []Finding{{Path: "src/main/resources/sql/V2__users.sql", Message: "Database migration added."}}},
			{"a test of one", nil, one("app/migrations/tests/test_one.py", "a\n"), nil},
			{"its notes", nil, one("app/migrations/README.md", "a\n"), nil},
			{"a like name", nil, one("src/migrations.ts", "a\n"), nil},
		})
	})
}

func TestExports(t *testing.T) {
	const a = "package a\n\nfunc Kept() {}\n\nfunc Old() {}\n\ntype T[K any] struct{}\n\nfunc (t *T[K]) Method() {}\n\nfunc hidden() {}\n"
	checkBoth(t, RuleExports, []ruleCase{
		{"removed and added", map[string]string{"pkg/a.go": a}, map[string]string{"pkg/a.go": strings.Replace(a, "Old", "New", 1)},
			[]Finding{{Path: "pkg/a.go", OldLine: 5, Message: "pkg no longer exports Old."}, {Path: "pkg/a.go", Line: 5, Message: "pkg now exports New."}}},
		{"a method and grouped constants", map[string]string{"pkg/a.go": a}, map[string]string{"pkg/a.go": strings.Replace(a, "Method", "Renamed", 1) + "\nconst (\n\tOne = iota\n\ttwo\n)\n"},
			[]Finding{{Path: "pkg/a.go", OldLine: 9, Message: "pkg no longer exports T.Method."}, {Path: "pkg/a.go", Line: 9, Message: "pkg now exports T.Renamed, One."}}},
		{"a package deleted", map[string]string{"pkg/a.go": a}, map[string]string{"pkg/a.go": ""},
			[]Finding{{Path: "pkg/a.go", OldLine: 3, Message: "pkg no longer exports Kept, Old, T, T.Method."}}},
		{"moved within the package", map[string]string{"pkg/a.go": a}, map[string]string{"pkg/a.go": "package a\n\nfunc Kept() {}\n", "pkg/b.go": strings.Replace(a, "func Kept() {}\n\n", "", 1)}, nil},
		{"an unexported one", map[string]string{"pkg/a.go": a}, map[string]string{"pkg/a.go": strings.Replace(a, "hidden", "concealed", 1)}, nil},
		{"an internal package", map[string]string{"internal/pkg/a.go": a}, map[string]string{"internal/pkg/a.go": "package a\n"}, nil},
		{"a command", map[string]string{"cmd/a.go": "package main\n\nfunc Old() {}\n"}, map[string]string{"cmd/a.go": "package main\n"}, nil},
		{"a test", map[string]string{"pkg/a_test.go": a}, map[string]string{"pkg/a_test.go": "package a\n"}, nil},
		{"a package's index", map[string]string{"web/package.json": "{}", "web/src/index.ts": "export { a, b as c } from './a'\nexport default x\n"},
			map[string]string{"web/src/index.ts": "export { a } from './a'\nexport default x\nexport * from './d'\nexport const e = 1\n"},
			[]Finding{{Path: "web/src/index.ts", OldLine: 1, Message: "web/src no longer exports c."}, {Path: "web/src/index.ts", Line: 3, Message: "web/src now exports everything from './d', e."}}},
		{"an index that isn't a package's", map[string]string{"web/package.json": "{}", "web/src/ui/index.ts": "export const a = 1\n"}, map[string]string{"web/src/ui/index.ts": "export const b = 1\n"}, nil},
	})
}

func TestARangeIsItsTotalChangeAndEachCommitsSecrets(t *testing.T) {
	dir := newRepo(t)
	base := gittest.Run(t, dir, "rev-parse", "HEAD")
	stage(t, dir, "kept.txt", fakeGitHubToken+"\n")
	stage(t, dir, "conflict.txt", "<<<<<<< HEAD\n")
	stage(t, dir, "removed.txt", "first\n"+strings.Replace(fakeGitHubToken, "aB3", "zZ9", 1)+"\n")
	gittest.Run(t, dir, "commit", "-q", "-m", "add")
	added := gittest.Run(t, dir, "rev-parse", "--short", "HEAD")
	stage(t, dir, "conflict.txt", "resolved\n")
	stage(t, dir, "removed.txt", "first\n")
	gittest.Run(t, dir, "commit", "-q", "-m", "clean up")

	report := checkRange(t, dir, base, "HEAD")
	if markers := withRule(report.Findings, RuleConflictMarker); len(markers) != 0 {
		t.Errorf("a marker that's gone = %+v", markers)
	}
	secrets := withRule(report.Findings, RuleSecret)
	if len(secrets) != 2 || secrets[0].Path != "removed.txt" || secrets[0].Line != 0 || !strings.HasSuffix(secrets[0].Message, "added in "+added+" and removed since.") {
		t.Fatalf("secrets = %+v", secrets)
	}
	if secrets[1].Path != "kept.txt" || secrets[1].Line != 1 {
		t.Errorf("secrets = %+v", secrets)
	}
	for _, secret := range secrets {
		if strings.Contains(secret.Message, fakeGitHubToken[8:]) {
			t.Errorf("value not masked: %q", secret.Message)
		}
	}
	if off := checkRange(t, dir, base, "HEAD", RuleSecret); len(off.Findings) != 0 {
		t.Errorf("turned off = %+v", off.Findings)
	}
}

func TestARangeToATree(t *testing.T) {
	dir := newRepo(t)
	head := gittest.Run(t, dir, "rev-parse", "HEAD")
	stage(t, dir, "a.txt", "<<<<<<< HEAD\n")
	stage(t, dir, "pkg/a.go", "package a\n\nfunc A() {}\n")
	tree := gittest.Run(t, dir, "write-tree")
	report, err := CheckRange(context.Background(), dir, Range{From: head, To: tree, Head: head}, nil)
	if err != nil || len(report.Findings) != 2 {
		t.Errorf("report = %+v, %v", report, err)
	}
}

func TestALongListKeepsSomeOfEveryRule(t *testing.T) {
	dir := newRepo(t)
	stage(t, dir, "a.txt", strings.Repeat("<<<<<<< HEAD\n", 2*mostFindings))
	for i := range 60 {
		stage(t, dir, fmt.Sprintf(".github/workflows/%d.yml", i), "x\n")
	}
	stage(t, dir, "AGENTS.md", "x\n")
	report, err := CheckStaged(context.Background(), dir, nil)
	if err != nil || len(report.Findings) != mostFindings {
		t.Fatalf("report = %d findings, %v", len(report.Findings), err)
	}
	kept := map[string]int{}
	for _, finding := range report.Findings {
		kept[finding.Rule]++
	}
	if kept[RuleAgentInstructions] != 1 || kept[RuleCI] < 49 || kept[RuleConflictMarker] < 49 {
		t.Errorf("kept = %v", kept)
	}
	if report.LeftOut[RuleConflictMarker] != 2*mostFindings-kept[RuleConflictMarker] || report.LeftOut[RuleCI] != 60-kept[RuleCI] || report.LeftOut[RuleAgentInstructions] != 0 {
		t.Errorf("left out = %v", report.LeftOut)
	}
}

func TestTooManyCommitsToReadEach(t *testing.T) {
	dir := newRepo(t)
	base := gittest.Run(t, dir, "rev-parse", "HEAD")
	var commits strings.Builder
	for i := range mostCommits + 1 {
		fmt.Fprintf(&commits, "commit refs/heads/many\nmark :%d\ncommitter t <t@example.com> %d +0000\ndata 0\n", i+1, i)
		if i == 0 {
			fmt.Fprintf(&commits, "from %s\n", base)
		}
		fmt.Fprintf(&commits, "M 100644 inline n\ndata <<END\n%d\nEND\n\n", i)
	}
	if _, err := gitexec.NewExecutor("").RunChecked(context.Background(), gitexec.Command{
		Dir: dir, Args: []string{"fast-import", "--quiet"}, Stdin: strings.NewReader(commits.String()),
	}); err != nil {
		t.Fatal(err)
	}
	if report := checkRange(t, dir, base, "many"); !report.CommitsNotRead {
		t.Error("the report doesn't say commits weren't read")
	}
	if report := checkRange(t, dir, "many~"+fmt.Sprint(mostCommits), "many"); report.CommitsNotRead {
		t.Error("the limit itself should be read")
	}
}
