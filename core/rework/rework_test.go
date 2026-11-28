package rework

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/internal/gittest"
)

var ctx = context.Background()

func lines(n int, word string) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		b.WriteString(word + " " + strings.Repeat("x", i) + "\n")
	}
	return b.String()
}

// repo returns a repository whose base commit holds old.go, with HEAD at base.
func repo(t *testing.T) (dir, base string) {
	t.Helper()
	dir = gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "old.go", lines(4, "before"), "base")
	return dir, strings.TrimSpace(gittest.Output(t, dir, "rev-parse", "HEAD"))
}

func report(t *testing.T, dir, base string) Report {
	t.Helper()
	r, err := Range(ctx, dir, base, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestRewriteCountsLinesWrittenEarlierInTheRange(t *testing.T) {
	dir, base := repo(t)
	gittest.CommitFile(t, dir, "a.go", lines(5, "first"), "write")
	gittest.CommitFile(t, dir, "a.go", lines(2, "first")+lines(3, "second"), "rewrite three")
	r := report(t, dir, base)
	if r.Written != 8 || r.Rewritten != 3 {
		t.Fatalf("written %d rewritten %d, want 8 and 3", r.Written, r.Rewritten)
	}
	if len(r.Commits) != 2 || r.Commits[0].Rewritten != 0 || r.Commits[1].Rewritten != 3 {
		t.Fatalf("commits %+v", r.Commits)
	}
	if len(r.Files) != 1 || r.Files[0] != (File{Path: "a.go", Written: 8, Rewritten: 3, Commits: 2}) {
		t.Fatalf("files %+v", r.Files)
	}
}

func TestLinesFromBeforeTheRangeArentRework(t *testing.T) {
	dir, base := repo(t)
	gittest.CommitFile(t, dir, "old.go", lines(2, "new"), "replace what the base wrote")
	if r := report(t, dir, base); r.Rewritten != 0 || len(r.Files) != 0 {
		t.Fatalf("got %+v", r)
	}
}

func TestMovedLinesArentRework(t *testing.T) {
	dir, base := repo(t)
	block := "func movedAround() error {\n\treturn errors.New(\"distinctive enough to follow\")\n}\n"
	gittest.CommitFile(t, dir, "a.go", block, "write")
	gittest.WriteFile(t, dir, "a.go", "")
	gittest.WriteFile(t, dir, "b.go", "\t"+strings.ReplaceAll(block, "\n", "\n\t"))
	gittest.Run(t, dir, "add", "-A")
	gittest.Run(t, dir, "commit", "-q", "-m", "move, indented")
	if r := report(t, dir, base); r.Rewritten != 0 {
		t.Fatalf("rewritten %d, want 0", r.Rewritten)
	}
}

func TestADeletedFileIsReworkUnderItsLastName(t *testing.T) {
	dir, base := repo(t)
	gittest.CommitFile(t, dir, "gone.go", lines(3, "short-lived"), "write")
	gittest.Run(t, dir, "rm", "-q", "gone.go")
	gittest.Run(t, dir, "commit", "-q", "-m", "delete")
	r := report(t, dir, base)
	if r.Rewritten != 3 || len(r.Files) != 1 || r.Files[0].Path != "gone.go" {
		t.Fatalf("got %+v", r)
	}
}

func TestARenamedFileKeepsItsReworkUnderItsNewName(t *testing.T) {
	dir, base := repo(t)
	gittest.CommitFile(t, dir, "a.go", lines(6, "kept"), "write")
	gittest.Run(t, dir, "mv", "a.go", "b.go")
	gittest.Run(t, dir, "commit", "-q", "-m", "rename")
	gittest.CommitFile(t, dir, "b.go", lines(5, "kept")+"changed\n", "rewrite one")
	r := report(t, dir, base)
	if r.Rewritten != 1 || len(r.Files) != 1 || r.Files[0].Path != "b.go" {
		t.Fatalf("got %+v", r)
	}
}

func TestBlankLinesDontCount(t *testing.T) {
	dir, base := repo(t)
	gittest.CommitFile(t, dir, "a.go", "one\n\n\ntwo\n", "write")
	gittest.CommitFile(t, dir, "a.go", "one\ntwo\n", "close up")
	r := report(t, dir, base)
	if r.Written != 2 || r.Rewritten != 0 {
		t.Fatalf("written %d rewritten %d, want 2 and 0", r.Written, r.Rewritten)
	}
}

func TestAnEmptyRange(t *testing.T) {
	dir, base := repo(t)
	r := report(t, dir, base)
	if r.Written != 0 || len(r.Commits) != 0 || len(r.Files) != 0 || r.Capped {
		t.Fatalf("got %+v", r)
	}
}

func TestParseHunk(t *testing.T) {
	for header, want := range map[string]hunk{
		"@@ -3,2 +3,0 @@":      {3, 2, 0},
		"@@ -7 +7 @@ func a()": {7, 1, 1},
		"@@ -0,0 +1,4 @@":      {0, 0, 4},
	} {
		if got, ok := parseHunk(header); !ok || got != want {
			t.Errorf("%q: got %+v, %v", header, got, ok)
		}
	}
}

func TestShortLinesAreMovedOnlyBesideADistinctiveMovedLine(t *testing.T) {
	line := func(n int, text string) deletedLine { return deletedLine{no: n, text: text, written: true} }
	added := func() map[string]int {
		return map[string]int{"return errors.New(\"distinctive enough\")": 1, "}": 2, "return nil": 1}
	}
	moved := []deletedLine{line(4, "return errors.New(\"distinctive enough\")"), line(5, "}")}
	if n := countRewritten(moved, added()); n != 0 {
		t.Errorf("a brace beside a moved line: rewritten %d, want 0", n)
	}
	alone := []deletedLine{line(9, "return nil"), line(10, "}")}
	if n := countRewritten(alone, added()); n != 2 {
		t.Errorf("short lines on their own: rewritten %d, want 2", n)
	}
}

func TestAPathWithASpace(t *testing.T) {
	dir, base := repo(t)
	gittest.CommitFile(t, dir, "release notes.md", lines(3, "note"), "write")
	gittest.CommitFile(t, dir, "release notes.md", lines(1, "note"), "trim")
	r := report(t, dir, base)
	if r.Rewritten != 2 || len(r.Files) != 1 || r.Files[0].Path != "release notes.md" {
		t.Fatalf("got %+v", r)
	}
}

func TestANewFileUnderARenamedFilesOldName(t *testing.T) {
	dir, base := repo(t)
	gittest.CommitFile(t, dir, "util.go", lines(6, "util"), "write")
	gittest.Run(t, dir, "mv", "util.go", "text.go")
	gittest.Run(t, dir, "commit", "-q", "-m", "rename")
	gittest.CommitFile(t, dir, "util.go", lines(3, "fresh"), "a new util")
	gittest.CommitFile(t, dir, "util.go", lines(1, "fresh"), "trim it")
	r := report(t, dir, base)
	if len(r.Files) != 1 || r.Files[0].Path != "util.go" || r.Files[0].Rewritten != 2 {
		t.Fatalf("files %+v", r.Files)
	}
}

func TestOnlyTheNewestCommitsAreRead(t *testing.T) {
	readLimit = 2
	t.Cleanup(func() { readLimit = MostCommits })
	dir, base := repo(t)
	gittest.CommitFile(t, dir, "a.go", lines(4, "older"), "write, unread")
	gittest.CommitFile(t, dir, "b.go", "b\n", "read")
	gittest.CommitFile(t, dir, "a.go", lines(1, "older"), "rewrite what an unread commit wrote")
	r := report(t, dir, base)
	if !r.Capped || len(r.Commits) != 2 || r.Rewritten != 0 {
		t.Fatalf("got %+v", r)
	}
}

func TestManyHunks(t *testing.T) {
	dir, base := repo(t)
	var before, after strings.Builder
	for i := 0; i < 210; i++ {
		line := strings.Repeat("y", i+1)
		before.WriteString(line + "\n")
		if i%2 == 0 {
			after.WriteString(line + "\n")
		}
	}
	gittest.CommitFile(t, dir, "many.txt", before.String(), "write")
	gittest.CommitFile(t, dir, "many.txt", after.String(), "drop every other line")
	if r := report(t, dir, base); r.Rewritten != 105 {
		t.Fatalf("rewritten %d, want 105", r.Rewritten)
	}
}

func TestAReindentedLineIsStillItsFirstWriters(t *testing.T) {
	dir, base := repo(t)
	gittest.CommitFile(t, dir, "a.go", "if ok {\nrun()\n}\n", "write")
	gittest.CommitFile(t, dir, "a.go", "if ok {\n\trun()\n}\n", "indent")
	gittest.CommitFile(t, dir, "a.go", "if ok {\n\twalk()\n}\n", "rewrite")
	r := report(t, dir, base)
	if got := []int{r.Commits[1].Rewritten, r.Commits[2].Rewritten}; got[0] != 1 || got[1] != 1 {
		t.Fatalf("rewritten by the indent and the rewrite %v, want 1 and 1", got)
	}
}

func TestLinesAMergeBroughtArentRework(t *testing.T) {
	dir, base := repo(t)
	trunk := gittest.Output(t, dir, "rev-parse", "--abbrev-ref", "HEAD")
	gittest.CommitFile(t, dir, "a.go", lines(3, "first"), "write")
	gittest.Run(t, dir, "checkout", "-q", "-b", "side")
	gittest.CommitFile(t, dir, "a.go", lines(3, "first")+lines(4, "side"), "on the side")
	gittest.Run(t, dir, "checkout", "-q", trunk)
	gittest.Run(t, dir, "merge", "-q", "--no-ff", "-m", "merge", "side")
	gittest.CommitFile(t, dir, "a.go", lines(2, "first")+lines(1, "side"), "trim both")
	r := report(t, dir, base)
	if len(r.Commits) != 2 || r.Rewritten != 1 {
		t.Fatalf("got %+v, want the one line the range's own commit wrote", r)
	}
}

// blamed is Range with each deleted line's writer asked of git blame, the slow
// way that Range's following of lines has to agree with.
func blamed(t *testing.T, dir, base string) Report {
	t.Helper()
	commits, capped, err := readCommits(ctx, dir, base, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	read := map[string]bool{}
	for _, c := range commits {
		read[c.sha] = true
	}
	for _, c := range commits {
		for _, f := range c.files {
			if len(f.deleted) == 0 {
				continue
			}
			before, err := git.BlameAt(ctx, dir, f.oldPath, c.sha+"^")
			if err != nil {
				t.Fatal(err)
			}
			for i, d := range f.deleted {
				f.deleted[i].written = read[before[d.no-1].SHA]
			}
		}
	}
	return tally(commits, capped)
}

func TestFollowingLinesAgreesWithBlame(t *testing.T) {
	for _, limit := range []int{MostCommits, 25} {
		readLimit = limit
		t.Cleanup(func() { readLimit = MostCommits })
		rng := rand.New(rand.NewPCG(7, uint64(limit)))
		dir, base := repo(t)
		trunk := gittest.Output(t, dir, "rev-parse", "--abbrev-ref", "HEAD")
		names := []string{"a.go", "b.go", "c.go"}
		files := map[string][]string{}
		fresh := 0
		line := func() string {
			fresh++
			return fmt.Sprintf("line %d with enough letters to be distinctive", fresh)
		}
		for step := range 60 {
			switch step {
			case 10:
				gittest.Run(t, dir, "checkout", "-q", "-b", "side")
				gittest.CommitFile(t, dir, "old.go", lines(2, "before")+lines(3, "side"), "side: old.go")
				gittest.CommitFile(t, dir, "side.go", lines(5, "side"), "side: side.go")
				gittest.Run(t, dir, "checkout", "-q", trunk)
			case 30:
				gittest.Run(t, dir, "merge", "-q", "--no-ff", "-m", "merge", "side")
				for _, name := range []string{"old.go", "side.go"} {
					content, err := os.ReadFile(filepath.Join(dir, name))
					if err != nil {
						t.Fatal(err)
					}
					files[name] = strings.Split(strings.TrimSuffix(string(content), "\n"), "\n")
					names = append(names, name)
				}
			case 40:
				gittest.Run(t, dir, "mv", "b.go", "renamed.go")
				files["renamed.go"], names[1] = files["b.go"], "renamed.go"
			}
			name := names[rng.IntN(len(names))]
			l := slices.Clone(files[name])
			for range 1 + rng.IntN(4) {
				at := rng.IntN(len(l) + 1)
				switch kind := rng.IntN(5); {
				case kind == 0 || len(l) < 4:
					l = append(l[:at:at], append([]string{line(), "}", line()}, l[at:]...)...)
				case at == len(l):
				case kind == 1:
					l = append(l[:at:at], l[min(at+1+rng.IntN(3), len(l)):]...)
				case kind == 2:
					l[at] = line()
				case kind == 3:
					l[at] = "\t" + l[at]
				default:
					moved := l[at]
					l = append(l[:at:at], l[at+1:]...)
					to := rng.IntN(len(l) + 1)
					l = append(l[:to:to], append([]string{moved}, l[to:]...)...)
				}
			}
			if slices.Equal(l, files[name]) {
				l = append(l, line())
			}
			files[name] = l
			gittest.CommitFile(t, dir, name, strings.Join(l, "\n")+"\n", fmt.Sprintf("step %d", step))
		}
		got, want := report(t, dir, base), blamed(t, dir, base)
		if want.Rewritten == 0 || want.Capped != (limit < 60) {
			t.Fatalf("limit %d: the history has no rework to compare, or the wrong cap: %+v", limit, want)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("limit %d:\n got %+v\nwant %+v", limit, got, want)
		}
	}
}

func TestARangeIsReadInTwoGitProcesses(t *testing.T) {
	dir, base := repo(t)
	for i := range 6 {
		gittest.CommitFile(t, dir, fmt.Sprintf("f%d.go", i%2), lines(i+2, "text"), "change")
	}
	trace := filepath.Join(t.TempDir(), "trace")
	t.Setenv("GIT_TRACE2_EVENT", trace)
	report(t, dir, base)
	events, err := os.ReadFile(trace)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(events), `"event":"start"`); n != 2 {
		t.Fatalf("%d git processes, want 2 however many commits and files", n)
	}
}

func BenchmarkRange(b *testing.B) {
	dir := gittest.InitRepo(b)
	gittest.CommitFile(b, dir, "old.go", "base\n", "base")
	base := gittest.Output(b, dir, "rev-parse", "HEAD")
	content := make([][]string, 8)
	for step := range 40 {
		for f := range content {
			if step == 0 {
				content[f] = strings.Split(strings.TrimSuffix(lines(200, "first"), "\n"), "\n")
			}
			for at := step % 10; at < len(content[f]); at += 10 {
				content[f][at] = fmt.Sprintf("step %d rewrote line %d of this file", step, at)
			}
			gittest.WriteFile(b, dir, fmt.Sprintf("f%d.go", f), strings.Join(content[f], "\n")+"\n")
		}
		gittest.Run(b, dir, "add", "-A")
		gittest.Run(b, dir, "commit", "-q", "-m", fmt.Sprintf("step %d", step))
	}
	for b.Loop() {
		if _, err := Range(ctx, dir, base, "HEAD"); err != nil {
			b.Fatal(err)
		}
	}
}
