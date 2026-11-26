package risk

import (
	"context"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/gspeager/current-client/core/deps"
	"github.com/gspeager/current-client/core/filekind"
	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
)

const (
	RuleSecret            = "secret"
	RuleConflictMarker    = "conflict-marker"
	RuleLargeFile         = "large-file"
	RuleLockfile          = "lockfile"
	RuleGenerated         = "generated"
	RuleFewerTests        = "fewer-tests"
	RuleSkippedTests      = "skipped-tests"
	RuleSilencedChecker   = "silenced-checker"
	RuleCI                = "ci"
	RuleAgentInstructions = "agent-instructions"
	RuleMigration         = "migration"
	RuleExports           = "exports"
)

const (
	largeFileBytes = 10 << 20
	generatedFiles = 20
	mostFindings   = 100
	// Past this many commits, only the range's total change is read for secrets.
	mostCommits = 500
	// Minified bundles aren't where a key is pasted, and regexes over them are slow.
	longestLineScanned = 8 << 10
)

// Finding's Line and OldLine are both 0 when it's about the whole file.
type Finding struct {
	Rule string
	Path string
	// Line is in the new version.
	Line    int
	Message string
	// OldLine is in the old version.
	OldLine int
}

// Report holds at most mostFindings findings, shared among the rules that match
// so a long list from one doesn't leave out another.
type Report struct {
	Findings []Finding
	// LeftOut counts each rule's findings past its share.
	LeftOut map[string]int
	// CommitsNotRead means the range has too many commits to read each for secrets.
	CommitsNotRead bool
}

// Range is what changes from the commit From to To.
type Range struct {
	From string
	// To is a commit or a tree.
	To string
	// Head is the range's last commit, when To is a tree.
	Head string
	// Env can name where the range's objects are.
	Env []string
}

// Check reports on what's staged, leaving out the rules in off.
func Check(ctx context.Context, repoPath string, off []string) ([]Finding, error) {
	report, err := CheckStaged(ctx, repoPath, off)
	return report.Findings, err
}

func CheckStaged(ctx context.Context, repoPath string, off []string) (Report, error) {
	found, err := inspect(ctx, change{repoPath: repoPath, diff: []string{"--cached"}, from: "HEAD", verb: "staged"}, off)
	return share(found.findings), err
}

// CheckRange reports on a range's total change. Secrets are looked for in each of
// its commits as well, since one removed again is still in what's pushed.
func CheckRange(ctx context.Context, repoPath string, r Range, off []string) (Report, error) {
	c := change{repoPath: repoPath, env: r.Env, diff: []string{r.From, r.To}, from: r.From, to: r.To, verb: "changed"}
	found, err := inspect(ctx, c, off)
	if err != nil || slices.Contains(off, RuleSecret) {
		return share(found.findings), err
	}
	head := r.Head
	if head == "" {
		head = r.To
	}
	removed, tooMany, err := removedSecrets(ctx, c, head, found.secrets)
	if err != nil {
		return Report{}, err
	}
	report := share(append(removed, found.findings...))
	report.CommitsNotRead = tooMany
	return report, nil
}

// change is one comparison: what's staged, or a range.
type change struct {
	repoPath string
	env      []string
	// diff is the arguments that say what git diff compares.
	diff []string
	// from and to are the revisions before and after; "" is the index.
	from, to string
	// verb says what was done to the files, for messages: "staged" or "changed".
	verb string
}

func (c change) run(ctx context.Context, stdin string, args ...string) (string, error) {
	result, err := gitexec.RunChecked(ctx, gitexec.Command{
		Dir:   c.repoPath,
		Env:   c.env,
		Args:  args,
		Stdin: strings.NewReader(stdin),
	})
	return result.Stdout, err
}

type found struct {
	findings []Finding
	// secrets holds the matched values, which never leave this package.
	secrets map[string]bool
}

func inspect(ctx context.Context, c change, off []string) (found, error) {
	on := func(rules ...string) bool {
		return slices.ContainsFunc(rules, func(rule string) bool { return !slices.Contains(off, rule) })
	}
	files, err := c.files(ctx)
	if err != nil || len(files) == 0 {
		return found{}, err
	}
	var present []string
	for _, file := range files {
		if file.Status != "D" {
			present = append(present, file.Path)
		}
	}
	sizes, err := c.sizes(ctx, append(present, siblings(present)...))
	if err != nil {
		return found{}, err
	}

	var findings []Finding
	secrets := map[string]bool{}
	for _, file := range present {
		if sizes[file] >= largeFileBytes {
			findings = append(findings, Finding{Rule: RuleLargeFile, Path: file, Message: fmt.Sprintf("Large file: %d MB.", sizes[file]>>20)})
		}
	}
	if on(RuleSecret, RuleConflictMarker, RuleFewerTests, RuleSkippedTests, RuleSilencedChecker) {
		lines, err := c.lines(ctx)
		if err != nil {
			return found{}, err
		}
		scanned, values := scanLines(lines.added, on(RuleSecret), on(RuleConflictMarker))
		findings, secrets = append(findings, scanned...), values
		findings = append(findings, fewerTests(lines)...)
		findings = append(findings, skippedTests(lines)...)
		findings = append(findings, silencedCheckers(lines)...)
	}
	if on(RuleLockfile) {
		lockfile, err := lockfileFindings(ctx, c, present, sizes)
		if err != nil {
			return found{}, err
		}
		findings = append(findings, lockfile...)
	}
	findings = append(findings, generatedFindings(present, c.verb)...)
	findings = append(findings, pathFindings(files)...)
	if on(RuleExports) {
		exports, err := exportFindings(ctx, c, files, sizes)
		if err != nil {
			return found{}, err
		}
		findings = append(findings, exports...)
	}
	findings = slices.DeleteFunc(findings, func(finding Finding) bool { return !on(finding.Rule) })
	return found{findings, secrets}, nil
}

// share gives every rule the same share of mostFindings, and the part one rule
// doesn't use to the others.
func share(findings []Finding) Report {
	if len(findings) <= mostFindings {
		return Report{Findings: findings}
	}
	byRule := map[string]int{}
	var rules []string
	for _, finding := range findings {
		if byRule[finding.Rule] == 0 {
			rules = append(rules, finding.Rule)
		}
		byRule[finding.Rule]++
	}
	kept := map[string]int{}
	for left := mostFindings; left > 0; {
		for _, rule := range rules {
			if left > 0 && kept[rule] < byRule[rule] {
				kept[rule]++
				left--
			}
		}
	}
	report := Report{LeftOut: map[string]int{}}
	for _, finding := range findings {
		if kept[finding.Rule] > 0 {
			kept[finding.Rule]--
			report.Findings = append(report.Findings, finding)
		} else {
			report.LeftOut[finding.Rule]++
		}
	}
	return report
}

func (c change) files(ctx context.Context) ([]git.ChangedFile, error) {
	args := append([]string{"diff", "--name-status", "-z", "--no-renames", "--no-ext-diff", "--no-relative"}, c.diff...)
	output, err := c.run(ctx, "", args...)
	if err != nil {
		return nil, err
	}
	return git.ParseNameStatus(output)
}

// siblings names the other half of each manifest or lockfile.
func siblings(files []string) []string {
	var names []string
	for _, file := range files {
		dir, name := path.Split(file)
		for _, lock := range deps.LocksOf(name) {
			names = append(names, dir+lock)
		}
		if manifest := deps.ManifestOf(name); manifest != "" {
			names = append(names, dir+manifest)
		}
	}
	return names
}

func (c change) sizes(ctx context.Context, paths []string) (map[string]int64, error) {
	var input strings.Builder
	var asked []string
	for _, file := range paths {
		if !strings.Contains(file, "\n") {
			input.WriteString(c.to + ":" + file + "\n")
			asked = append(asked, file)
		}
	}
	output, err := c.run(ctx, input.String(), "cat-file", "--batch-check=%(objecttype) %(objectsize)")
	if err != nil {
		return nil, err
	}
	sizes := map[string]int64{}
	for i, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
		kind, size, ok := strings.Cut(line, " ")
		if !ok || kind != "blob" || i >= len(asked) {
			continue
		}
		if bytes, err := strconv.ParseInt(size, 10, 64); err == nil {
			sizes[asked[i]] = bytes
		}
	}
	return sizes, nil
}

func lockfileFindings(ctx context.Context, c change, files []string, sizes map[string]int64) ([]Finding, error) {
	var findings []Finding
	for _, file := range files {
		dir, name := path.Split(file)
		if manifest := deps.ManifestOf(name); manifest != "" {
			if _, tracked := sizes[dir+manifest]; tracked && !slices.Contains(files, dir+manifest) {
				findings = append(findings, Finding{Rule: RuleLockfile, Path: file, Message: fmt.Sprintf("%s changed without %s.", name, manifest)})
			}
			continue
		}
		var missing string
		for _, lock := range deps.LocksOf(name) {
			if slices.Contains(files, dir+lock) {
				missing = ""
				break
			}
			if _, tracked := sizes[dir+lock]; tracked && missing == "" {
				missing = lock
			}
		}
		if missing == "" {
			continue
		}
		changed, err := packagesChanged(ctx, c, file)
		if err != nil {
			return nil, err
		}
		if changed {
			findings = append(findings, Finding{Rule: RuleLockfile, Path: file, Message: fmt.Sprintf("Packages changed in %s without %s.", name, missing)})
		}
	}
	return findings, nil
}

// packagesChanged ignores a manifest's scripts and metadata, which don't touch the
// lockfile. A manifest that doesn't parse can't be ruled out, so it counts as changed.
func packagesChanged(ctx context.Context, c change, file string) (bool, error) {
	before, after := git.RevPath{Rev: c.from, Path: file}, git.RevPath{Rev: c.to, Path: file}
	files, err := git.FilesAtWith(ctx, c.repoPath, c.env, []git.RevPath{before, after})
	if err != nil {
		return false, err
	}
	changes, err := deps.Summarize(file, files[before], files[after])
	return err != nil || len(changes) > 0, nil
}

func generatedFindings(files []string, verb string) []Finding {
	var first string
	count := 0
	for _, file := range files {
		if filekind.IsGenerated(file) {
			if count == 0 {
				first = file
			}
			count++
		}
	}
	if count < generatedFiles {
		return nil
	}
	return []Finding{{Rule: RuleGenerated, Path: first, Message: fmt.Sprintf("%d generated files %s.", count, verb)}}
}
