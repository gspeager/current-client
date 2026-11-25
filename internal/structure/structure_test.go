package structure

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type files map[string]string

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func repo(t *testing.T, content files) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
	commit(t, dir, content)
	return dir
}

func commit(t *testing.T, dir string, content files, removed ...string) {
	t.Helper()
	for name, text := range content {
		file := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range removed {
		runGit(t, dir, "rm", "-q", name)
	}
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-q", "-m", "files")
}

func build(t *testing.T, dir, rev string) *Graph {
	t.Helper()
	graph, err := Build(context.Background(), dir, nil, rev)
	if err != nil {
		t.Fatal(err)
	}
	return graph
}

func linksOf(graph *Graph) []string {
	var links []string
	for _, l := range graph.Links {
		links = append(links, graph.Files[l[0]]+" > "+graph.Files[l[1]])
	}
	slices.Sort(links)
	return links
}

func expect(t *testing.T, what string, got []string, want ...string) {
	t.Helper()
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("%s:\n got %s\nwant %s", what, strings.Join(got, "\n     "), strings.Join(want, "\n     "))
	}
}

func TestGo(t *testing.T) {
	graph := build(t, repo(t, files{
		"go.mod":                    "module example.com/app\n",
		"main.go":                   "package main\n\nimport (\n\t\"example.com/app/store\"\n\t_ \"example.com/app/driver\"\n\tkit \"example.com/app/tools\"\n\t\"example.com/other\"\n)\n\nfunc main() { store.Open(); kit.Run(); helper(); other.X() }\n",
		"helper.go":                 "package main\n\n// store.Closed in a comment isn't used.\nfunc helper() { _ = \"openAll\" }\n",
		"store/store.go":            "// Package store keeps things. More words.\npackage store\n\nfunc Open() { openAll() }\n",
		"store/all.go":              "package store\n\nfunc openAll() {}\n\nfunc Closed() {}\n",
		"store/all_test.go":         "package store\n\nfunc TestX() { Open() }\n",
		"driver/d.go":               "package driver\n",
		"tools/kit.go":              "package kit\n\nfunc Run() {}\n",
		"nested/go.mod":             "module example.com/app/nested\n",
		"nested/in/in.go":           "package in\n\nimport . \"example.com/app/store\"\n\nfunc x() { Closed() }\n",
		"broken.go":                 "package main\n\nimport \"example.com/app/store\"\n\nfunc broken( { store.Closed()\n",
		"vendor/example.com/v/v.go": "package v\n",
	}), "HEAD")
	expect(t, "links", linksOf(graph),
		"main.go > store/store.go", "main.go > driver/d.go", "main.go > tools/kit.go", "main.go > helper.go",
		"store/store.go > store/all.go", "nested/in/in.go > store/all.go", "broken.go > store/all.go")
	if graph.Docs["store"] != "keeps things" {
		t.Errorf("docs = %v", graph.Docs)
	}
}

func TestTypeScript(t *testing.T) {
	graph := build(t, repo(t, files{
		"package.json":                  `{"name": "root", "description": "All of it. Really."}`,
		"packages/ui/package.json":      `{"name": "@acme/ui"}`,
		"packages/ui/src/index.ts":      "export * from './button'\n",
		"packages/ui/src/button.tsx":    "export const a = 1\n",
		"packages/ui/src/theme/a.ts":    "export {}\n",
		"app/tsconfig.json":             "{\n  // paths\n  \"compilerOptions\": { \"paths\": { \"@/*\": [\"./src/*\"], \"exact\": [\"./src/exact.ts\"], }, },\n}\n",
		"app/src/main.ts":               "import { a } from '@acme/ui'\nimport t from \"@acme/ui/theme/a\"\nimport type { X } from '@/lib/x'\nimport './side.js'\nconst l = () => import('./lazy')\nconst c = require('./common')\nimport e from 'exact'\nimport {\n  many,\n  lines,\n} from './lines'\nimport react from 'react'\n",
		"app/src/lib/x.ts":              "export {}\n",
		"app/src/side.ts":               "// import './lazy'\n/* import './common' */\nconst s = `import './lazy'`\nconst url = 'https://example.com' // not a comment's start\nimport './exact'\n",
		"app/src/lazy/index.tsx":        "export {}\n",
		"app/src/common.cjs":            "module.exports = {}\n",
		"app/src/exact.ts":              "export {}\n",
		"app/src/lines.ts":              "export {}\n",
		"app/src/lines.test.ts":         "import './lines'\n",
		"other/src/main.ts":             "import '@/lib/x'\n",
		"app/src/types.d.ts":            "import './lines'\n",
		"app/dist/bundle.js":            "import './x'\n",
		"packages/ui/src/Widget.svelte": "<script></script>\n",
	}), "HEAD")
	expect(t, "links", linksOf(graph),
		"packages/ui/src/index.ts > packages/ui/src/button.tsx",
		"app/src/main.ts > packages/ui/src/index.ts", "app/src/main.ts > packages/ui/src/theme/a.ts",
		"app/src/main.ts > app/src/lib/x.ts", "app/src/main.ts > app/src/side.ts", "app/src/main.ts > app/src/lazy/index.tsx",
		"app/src/main.ts > app/src/common.cjs", "app/src/main.ts > app/src/exact.ts", "app/src/main.ts > app/src/lines.ts",
		"app/src/side.ts > app/src/exact.ts")
	if graph.Docs[""] != "All of it" || graph.Unread != 1 {
		t.Errorf("docs = %v, %d unread", graph.Docs, graph.Unread)
	}
}

func TestPython(t *testing.T) {
	graph := build(t, repo(t, files{
		"shop/__init__.py":       "from . import cart\n",
		"shop/cart.py":           "from .money import (\n    Price,  # the price\n    total,\n)\nimport shop.util.dates as dates, os\n\"\"\"\nimport shop.money\n\"\"\"\n",
		"shop/money.py":          "from shop.util import \\\n    dates\n",
		"shop/util/__init__.py":  "",
		"shop/util/dates.py":     "from ..money import Price\nfrom ... import nothing\n",
		"shop/lazy.py":           "def f():\n    from shop import cart\n",
		"src/lib/__init__.py":    "",
		"src/lib/a.py":           "import lib.b\n",
		"src/lib/b.py":           "",
		"scripts/run.py":         "import helper\nimport shop\n",
		"scripts/helper.py":      "",
		"tools/x/__init__.py":    "",
		"tools/helper.py":        "",
		"shop/uses_helper.py":    "import helper\n",
		"tests/test_cart.py":     "import shop.cart\n",
		"shop/util/not_python.c": "",
	}), "HEAD")
	expect(t, "links", linksOf(graph),
		"shop/__init__.py > shop/cart.py", "shop/cart.py > shop/money.py", "shop/cart.py > shop/util/dates.py",
		"shop/money.py > shop/util/dates.py", "shop/util/dates.py > shop/money.py", "shop/lazy.py > shop/cart.py",
		"src/lib/a.py > src/lib/b.py", "scripts/run.py > scripts/helper.py", "scripts/run.py > shop/__init__.py")
}

func TestCSharp(t *testing.T) {
	graph := build(t, repo(t, files{
		"App/App.csproj":    "",
		"App/Global.cs":     "global using Acme.Shared;\n",
		"App/Main.cs":       "using Acme.Store;\nnamespace Acme.App;\n\n[Cached]\nclass Program {\n  // Hidden in a comment\n  string s = \"Hidden\"; string v = @\"Hidden \"\" \\\"; char q = '\"';\n  Cart cart; Money m; Shared x; int n = cart.Hidden; Settings t;\n}\n",
		"App/Settings.cs":   "namespace Acme.App { class Settings { class Hidden { } } }\n",
		"Store/Cart.cs":     "namespace Acme.Store;\npublic partial class Cart { }\n",
		"Store/Cart.Sum.cs": "namespace Acme.Store;\npublic partial class Cart { }\n",
		"Store/Cached.cs":   "namespace Acme.Store;\npublic class CachedAttribute { }\n",
		"Store/Money.cs":    "namespace Acme.Elsewhere;\npublic class Money { }\npublic class Hidden { }\n",
		"Shared/Shared.cs":  "namespace Acme.Shared;\npublic record Shared(int A);\n",
	}), "HEAD")
	expect(t, "links", linksOf(graph),
		"App/Main.cs > App/Settings.cs", "App/Main.cs > Store/Cart.cs", "App/Main.cs > Store/Cart.Sum.cs",
		"App/Main.cs > Store/Cached.cs", "App/Main.cs > Shared/Shared.cs")
}

func TestSwift(t *testing.T) {
	graph := build(t, repo(t, files{
		"Modules/Core/Package.swift":             "import PackageDescription\n",
		"Modules/Core/Sources/Core/Feed.swift":   "public struct Feed { struct Nested {} }\n",
		"Modules/Core/Sources/Core/Parse.swift":  "func parse() -> Feed { Feed() }\n",
		"Modules/Other/Package.swift":            "import PackageDescription\n",
		"Modules/Other/Sources/Other/Feed.swift": "public class Unseen {}\n",
		"Mac/App.swift":                          "import Core\n\nclass App {\n  let feed: Feed // Unseen\n  let text = \"\"\"\n  Window\n  \"\"\"\n  let u: Unseen\n  let n: Other.Window\n}\n",
		"Mac/Window.swift":                       "class Window { let app: App }\n",
	}), "HEAD")
	expect(t, "links", linksOf(graph),
		"Modules/Core/Sources/Core/Parse.swift > Modules/Core/Sources/Core/Feed.swift",
		"Mac/App.swift > Modules/Core/Sources/Core/Feed.swift", "Mac/Window.swift > Mac/App.swift")
}

func TestALanguageThatIsntRead(t *testing.T) {
	graph := build(t, repo(t, files{"src/main.rs": "mod a;\n", "src/a.rs": "", "README.md": "# Name\n\n[![badge](x)](y)\n\nA thing that **does** `this`, see [it](x).\nAnd more.\n"}), "HEAD")
	if len(graph.Links) != 0 || len(graph.Languages) != 0 || graph.Unread != 2 || len(graph.Files) != 3 {
		t.Errorf("graph = %+v", graph)
	}
	if graph.Docs[""] != "A thing that does this, see it. And more" && graph.Docs[""] != "A thing that does this, see it" {
		t.Errorf("docs = %q", graph.Docs[""])
	}
}

func describe(level Level) []string {
	var all []string
	for _, n := range level.Nodes {
		all = append(all, strings.TrimSpace(fmt.Sprintf("%s %s %d %s", n.Side, n.Path, len(n.Files), n.State)))
	}
	for _, l := range level.Links {
		all = append(all, strings.TrimSpace(fmt.Sprintf("%s > %s %d %s", level.Nodes[l.From].Path, level.Nodes[l.To].Path, l.Count, l.State)))
	}
	slices.Sort(all)
	return all
}

var layered = files{
	"go.mod":               "module a\n",
	"main.go":              "package main\n\nimport \"a/app\"\n\nfunc main() { app.Run() }\n",
	"app/run.go":           "package app\n\nimport (\n\t\"a/core/deep/store\"\n\t\"a/core/text\"\n)\n\nfunc Run() { store.Open(); text.Cut(); start() }\n",
	"app/start.go":         "package app\n\nimport \"a/core/text\"\n\nfunc start() { text.Cut() }\n",
	"core/deep/store/s.go": "package store\n\nimport \"a/core/text\"\n\nfunc Open() { text.Cut() }\n",
	"core/text/t.go":       "package text\n\nfunc Cut() {}\n",
	"docs/only/here/a.md":  "x\n",
}

func TestLevels(t *testing.T) {
	view := NewView(build(t, repo(t, layered), "HEAD"), nil)
	expect(t, "the whole", describe(view.Level("")),
		"card app 2", "card core 2", "card docs/only/here 1", "card go.mod 1", "card main.go 1",
		"main.go > app 1", "app > core 3")
	expect(t, "a folder", describe(view.Level("core")),
		"card core/deep/store 1", "card core/text 1", "core/deep/store > core/text 1",
		// Few enough to name.
		"used-by app/run.go 1", "used-by app/start.go 1",
		"app/run.go > core/deep/store 1", "app/run.go > core/text 1", "app/start.go > core/text 1")
	// Its only folder isn't a step, and neither is the one above.
	if level := view.Level("core/deep"); level.Path != "core/deep/store" || fmt.Sprint(level.Crumbs) != "[{ } {core core} {core/deep/store deep/store}]" {
		t.Errorf("core/deep is %s, %v", level.Path, level.Crumbs)
	}
	expect(t, "a file", describe(view.Level("app/run.go")),
		"card app/run.go 1", "used-by main.go 1", "uses app/start.go 1", "uses core/deep/store/s.go 1", "uses core/text/t.go 1",
		"main.go > app/run.go 1", "app/run.go > app/start.go 1", "app/run.go > core/deep/store/s.go 1", "app/run.go > core/text/t.go 1")
	if level := view.Level("app/run.go"); level.Folder || len(level.Files) != 1 {
		t.Errorf("a file = %+v", level)
	}
	if level := view.Level("nowhere"); len(level.Nodes) != 0 {
		t.Errorf("a folder that isn't there = %+v", level)
	}
	// A name that starts another isn't in it.
	if level := view.Level("app"); len(level.Files) != 2 {
		t.Errorf("app = %v", level.Files)
	}
}

func TestALevelComparedWithAnotherPoint(t *testing.T) {
	dir := repo(t, layered)
	runGit(t, dir, "tag", "then")
	commit(t, dir, files{
		"app/start.go":   "package app\n\nimport \"a/extra\"\n\nfunc start() { extra.X() }\n",
		"extra/x.go":     "package extra\n\nfunc X() {}\n",
		"core/text/u.go": "package text\n\nfunc Cut() {}\n",
	}, "core/deep/store/s.go", "core/text/t.go", "docs/only/here/a.md")
	commit(t, dir, files{"app/run.go": "package app\n\nimport \"a/core/text\"\n\nfunc Run() { text.Cut(); start() }\n"})
	view := NewView(build(t, dir, "HEAD"), build(t, dir, "then"))
	expect(t, "the whole", describe(view.Level("")),
		"card app 2", "card core 1", "card docs/only/here 0 removed", "card extra 1 added", "card go.mod 1", "card main.go 1",
		"main.go > app 1", "app > core 3", "app > extra 1 added")
	expect(t, "a folder that lost one", describe(view.Level("core")),
		"card core/deep/store 0 removed", "card core/text 1", "core/deep/store > core/text 1 removed",
		"used-by app/run.go 1", "used-by app/start.go 1",
		"app/run.go > core/deep/store 1 removed", "app/run.go > core/text 1", "app/start.go > core/text 1 removed")
	// A renamed file is one gone and one new, in a folder that's the same.
	expect(t, "a rename", describe(view.Level("core/text")),
		"card core/text/t.go 0 removed", "card core/text/u.go 1 added",
		"used-by app/run.go 1", "used-by app/start.go 1", "used-by core/deep/store/s.go 0 removed",
		"app/run.go > core/text/t.go 1 removed", "app/run.go > core/text/u.go 1 added", "app/start.go > core/text/t.go 1 removed",
		"core/deep/store/s.go > core/text/t.go 1 removed")
}

func TestABusyLevel(t *testing.T) {
	content := files{"go.mod": "module a\n", "hub/h.go": "package hub\n\nfunc H() {}\n"}
	for i := range 40 {
		uses := ""
		for k := range i % 12 {
			uses += fmt.Sprintf("a/lib/p%02d ", (i+k+1)%40)
		}
		var imports, calls strings.Builder
		for _, p := range append(strings.Fields(uses), "a/hub") {
			fmt.Fprintf(&imports, "\t%q\n", p)
			name := p[strings.LastIndex(p, "/")+1:]
			fmt.Fprintf(&calls, "%s.%s(); ", name, map[bool]string{true: "H", false: "F"}[name == "hub"])
		}
		content[fmt.Sprintf("lib/p%02d/f.go", i)] = fmt.Sprintf("package p%02d\n\nimport (\n%s)\n\nfunc F() { %s }\n", i, imports.String(), calls.String())
	}
	level := NewView(build(t, repo(t, content), "HEAD"), nil).Level("lib")
	main := 0
	for _, l := range level.Links {
		if l.Main {
			main++
		}
	}
	if len(level.Nodes) != mostCards || len(level.Rest) != 40-mostCards || main != mostMain || len(level.Links) <= mostMain {
		t.Errorf("%d cards, %d listed, %d of %d lines", len(level.Nodes), len(level.Rest), main, len(level.Links))
	}
	if len(level.WidelyUses) != 1 || level.WidelyUses[0] != (Wide{Path: "hub/h.go", Label: "h.go", Cards: 40}) || level.Cards != 40 {
		t.Errorf("widely uses = %+v of %d", level.WidelyUses, level.Cards)
	}
	for _, n := range level.Nodes {
		if n.Slot < 1 || n.Slot > 8 {
			t.Errorf("%s has color %d", n.Path, n.Slot)
		}
	}
}

func TestWhatAFileSaysIsKeptByItsContent(t *testing.T) {
	dir := repo(t, layered)
	build(t, dir, "HEAD")
	kept.Lock()
	before := len(kept.facts)
	kept.Unlock()
	commit(t, dir, files{"app/start.go": "package app\n\nfunc start() { /* only this */ }\n"})
	build(t, dir, "HEAD")
	build(t, dir, "HEAD~1")
	kept.Lock()
	defer kept.Unlock()
	if read := len(kept.facts) - before; read != 1 {
		t.Errorf("read %d files again, want the one that changed", read)
	}
}
