package structure

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

func describeShape(shape Shape) []string {
	var all []string
	for _, a := range shape.Areas {
		all = append(all, strings.TrimSpace(fmt.Sprintf("area %q %d files +%d -%d %s", a.Path, len(a.Files), a.Added, a.Removed, a.State)))
	}
	for _, l := range shape.Links {
		all = append(all, fmt.Sprintf("link %s > %s %d %s", l.From, l.To, l.Count, l.State))
	}
	for _, r := range shape.ReadFirst {
		all = append(all, fmt.Sprintf("read first %s %s %d", r.Path, r.Why, r.Count))
	}
	for _, r := range shape.Skim {
		all = append(all, strings.TrimSpace(fmt.Sprintf("skim %s %s %d", r.Path, r.Why, r.Count)))
	}
	slices.Sort(all)
	return all
}

func TestCompareChange(t *testing.T) {
	dir := repo(t, layered)
	runGit(t, dir, "tag", "base")
	commit(t, dir, files{
		"app/start.go":   "package app\n\nimport \"a/extra\"\n\nfunc start() { extra.X() }\n",
		"app/run.go":     "package app\n\nimport \"a/core/deep/store\"\n\nfunc Run() { store.Open(); start() }\n",
		"extra/x.go":     "package extra\n\nimport \"a/core/text\"\n\nfunc X() { text.Cut() }\n",
		"go.sum":         "x\n",
		"core/text/t.go": "package text\n\n// Cut cuts.\nfunc Cut() {}\n",
	}, "docs/only/here/a.md")
	runGit(t, dir, "mv", "core/deep/store/s.go", "core/deep/store/store.go")
	runGit(t, dir, "commit", "-q", "-m", "rename")
	changes := []Changed{
		{Path: "app/start.go", Status: "M", Added: 3, Removed: 3},
		{Path: "app/run.go", Status: "M", Added: 2, Removed: 3},
		{Path: "extra/x.go", Status: "A", Added: 5},
		{Path: "go.sum", Status: "A", Added: 1},
		{Path: "core/text/t.go", Status: "M", Added: 1},
		{Path: "docs/only/here/a.md", Status: "D", Removed: 1},
		{Path: "core/deep/store/store.go", OrigPath: "core/deep/store/s.go", Status: "R"},
	}
	shape := CompareChange(build(t, dir, "HEAD"), build(t, dir, "base"), changes)
	expect(t, "shape", describeShape(shape),
		`area "" 1 files +1 -0`, `area "app" 2 files +5 -6`, `area "core/deep/store" 1 files +0 -0`,
		`area "core/text" 1 files +1 -0`, `area "docs/only/here" 1 files +0 -1 removed`, `area "extra" 1 files +5 -0 added`,
		"link app > extra 1 added", "link extra > core/text 1 added", "link app > core/text 2 removed",
		// By how many of the change's areas use it.
		"read first core/text used 2", "read first app used 1", "read first core/deep/store used 1", "read first extra used 1",
		"skim go.sum lockfile 0", "skim core/deep/store/store.go renamed 0", "skim core/text few 1", `skim  few 1`)
}

func TestCompareAChangeInNoLanguageRead(t *testing.T) {
	dir := repo(t, files{"src/main.rs": "mod a;\n"})
	runGit(t, dir, "tag", "base")
	commit(t, dir, files{"src/a.rs": "fn a() {}\n", "lib/b.rs": "fn b() {}\n"})
	now, then := build(t, dir, "HEAD"), build(t, dir, "base")
	shape := CompareChange(now, then, []Changed{{Path: "src/a.rs", Status: "A", Added: 1}, {Path: "lib/b.rs", Status: "A", Added: 1}})
	expect(t, "shape", describeShape(shape), `area "lib" 1 files +1 -0 added`, `area "src" 1 files +1 -0`, "skim src few 1")
	if len(now.Languages) != 0 {
		t.Errorf("languages = %v", now.Languages)
	}
	// Without the graphs, as in a partial clone, it's areas alone.
	expect(t, "without graphs", describeShape(CompareChange(nil, nil, []Changed{{Path: "lib/b.rs", Status: "A", Added: 1}})),
		`area "lib" 1 files +1 -0`, "skim lib few 1")
}

func TestReadFirstAFileMuchOfTheCodeUses(t *testing.T) {
	content := files{"go.mod": "module a\n", "base/b.go": "package base\n\nfunc B() {}\n"}
	for i := range widelyUsed {
		content[fmt.Sprintf("u%d/u.go", i)] = "package u\n\nimport \"a/base\"\n\nfunc U() { base.B() }\n"
	}
	dir := repo(t, content)
	runGit(t, dir, "tag", "base")
	commit(t, dir, files{"base/b.go": "package base\n\n// B is b.\nfunc B() {}\n"})
	shape := CompareChange(build(t, dir, "HEAD"), build(t, dir, "base"), []Changed{{Path: "base/b.go", Status: "M", Added: 1}})
	if !slices.Contains(describeShape(shape), fmt.Sprintf("read first base/b.go used %d", widelyUsed)) {
		t.Errorf("shape = %q", describeShape(shape))
	}
}
