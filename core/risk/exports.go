package risk

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/gspeager/current-client/core/filekind"
	"github.com/gspeager/current-client/core/git"
)

// place is where a name is exported.
type place struct {
	path string
	line int
}

// exportFindings compares what each package exports before and after. Go's is
// every exported name outside main and internal packages. A JavaScript or
// TypeScript package's is what its index exports, beside package.json or in src.
// Only changed files are read: the others export the same on both sides.
func exportFindings(ctx context.Context, c change, files []git.ChangedFile, sizes map[string]int64) ([]Finding, error) {
	var wanted []git.RevPath
	var read []string
	entries, err := c.scriptEntries(ctx, files)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if strings.Contains(file.Path, "\n") || sizes[file.Path] >= largeFileBytes || !(isGoPackageFile(file.Path) || entries[file.Path]) {
			continue
		}
		read = append(read, file.Path)
		wanted = append(wanted, git.RevPath{Rev: c.from, Path: file.Path}, git.RevPath{Rev: c.to, Path: file.Path})
	}
	contents, err := git.FilesAtWith(ctx, c.repoPath, c.env, wanted)
	if err != nil {
		return nil, err
	}
	before, after := map[string]map[string]place{}, map[string]map[string]place{}
	for _, file := range read {
		for _, side := range []struct {
			rev     string
			exports map[string]map[string]place
		}{{c.from, before}, {c.to, after}} {
			content, present := contents[git.RevPath{Rev: side.rev, Path: file}]
			if !present {
				continue
			}
			pkg := path.Dir(file)
			if side.exports[pkg] == nil {
				side.exports[pkg] = map[string]place{}
			}
			names := scriptExports
			if path.Ext(file) == ".go" {
				names = goExports
			}
			for name, line := range names(content) {
				side.exports[pkg][name] = place{file, line}
			}
		}
	}
	packages := slices.Sorted(maps.Keys(mergedKeys(before, after)))
	var findings []Finding
	for _, pkg := range packages {
		name := pkg
		if pkg == "." {
			name = "The top folder"
		}
		if gone := missingFrom(before[pkg], after[pkg]); len(gone) > 0 {
			at := before[pkg][gone[0]]
			findings = append(findings, Finding{Rule: RuleExports, Path: at.path, OldLine: at.line, Message: fmt.Sprintf("%s no longer exports %s.", name, list(gone))})
		}
		if added := missingFrom(after[pkg], before[pkg]); len(added) > 0 {
			at := after[pkg][added[0]]
			findings = append(findings, Finding{Rule: RuleExports, Path: at.path, Line: at.line, Message: fmt.Sprintf("%s now exports %s.", name, list(added))})
		}
	}
	return findings, nil
}

func mergedKeys(a, b map[string]map[string]place) map[string]bool {
	keys := map[string]bool{}
	for key := range a {
		keys[key] = true
	}
	for key := range b {
		keys[key] = true
	}
	return keys
}

// missingFrom names what's in a and not in b, in the order of a's lines.
func missingFrom(a, b map[string]place) []string {
	var names []string
	for name := range a {
		if _, there := b[name]; !there {
			names = append(names, name)
		}
	}
	slices.SortFunc(names, func(x, y string) int {
		if order := strings.Compare(a[x].path, a[y].path); order != 0 {
			return order
		}
		if a[x].line != a[y].line {
			return a[x].line - a[y].line
		}
		return strings.Compare(x, y)
	})
	return names
}

const mostNamed = 4

func list(names []string) string {
	if len(names) <= mostNamed+1 {
		return strings.Join(names, ", ")
	}
	return fmt.Sprintf("%s and %d more", strings.Join(names[:mostNamed], ", "), len(names)-mostNamed)
}

var internalFolder = regexp.MustCompile(`(^|/)(internal|testdata|_[^/]*|\.[^/]*)/`)

func isGoPackageFile(file string) bool {
	return path.Ext(file) == ".go" && filekind.Of(file) == filekind.Code && !internalFolder.MatchString(file)
}

func goExports(content []byte) map[string]int {
	names := map[string]int{}
	fileSet := token.NewFileSet()
	// What parsed before an error is still worth comparing.
	file, _ := parser.ParseFile(fileSet, "", content, parser.SkipObjectResolution)
	if file == nil || file.Name == nil || file.Name.Name == "main" {
		return names
	}
	add := func(name string, ident *ast.Ident) {
		names[name] = fileSet.Position(ident.Pos()).Line
	}
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			if !decl.Name.IsExported() {
				continue
			}
			if decl.Recv == nil {
				add(decl.Name.Name, decl.Name)
			} else if receiver := receiverName(decl.Recv); ast.IsExported(receiver) {
				add(receiver+"."+decl.Name.Name, decl.Name)
			}
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					if spec.Name.IsExported() {
						add(spec.Name.Name, spec.Name)
					}
				case *ast.ValueSpec:
					for _, name := range spec.Names {
						if name.IsExported() {
							add(name.Name, name)
						}
					}
				}
			}
		}
	}
	return names
}

func receiverName(receiver *ast.FieldList) string {
	if len(receiver.List) == 0 {
		return ""
	}
	for expr := receiver.List[0].Type; ; {
		switch typed := expr.(type) {
		case *ast.StarExpr:
			expr = typed.X
		case *ast.IndexExpr:
			expr = typed.X
		case *ast.IndexListExpr:
			expr = typed.X
		case *ast.ParenExpr:
			expr = typed.X
		case *ast.Ident:
			return typed.Name
		default:
			return ""
		}
	}
}

var scriptEntry = regexp.MustCompile(`^(.*/)?(src/)?index\.(d\.ts|[cm]?[jt]sx?)$`)

// scriptEntries is the changed files that are a package's index.
func (c change) scriptEntries(ctx context.Context, files []git.ChangedFile) (map[string]bool, error) {
	manifests := map[string][]string{}
	var asked []string
	for _, file := range files {
		if !scriptEntry.MatchString(file.Path) || filekind.Of(file.Path) != filekind.Code {
			continue
		}
		folder := path.Dir(file.Path)
		for _, dir := range []string{folder, path.Dir(folder)} {
			if dir != folder && path.Base(folder) != "src" {
				continue
			}
			manifest := path.Join(dir, "package.json")
			if manifests[manifest] == nil {
				asked = append(asked, manifest)
			}
			manifests[manifest] = append(manifests[manifest], file.Path)
		}
	}
	entries := map[string]bool{}
	if len(asked) == 0 {
		return entries, nil
	}
	sizes, err := c.sizes(ctx, asked)
	if err != nil {
		return nil, err
	}
	for manifest := range sizes {
		for _, file := range manifests[manifest] {
			entries[file] = true
		}
	}
	return entries, nil
}

var (
	scriptDeclared = regexp.MustCompile(`^\s*export\s+(?:declare\s+)?(default\b|(?:abstract\s+)?(?:async\s+)?(?:function\s*\*?|class|const\s+enum|const|let|var|interface|type|enum|namespace)\s+([\w$]+))`)
	scriptList     = regexp.MustCompile(`(?s)\bexport\s+(?:type\s+)?\{([^}]*)\}`)
	scriptAll      = regexp.MustCompile(`^\s*export\s+(?:type\s+)?\*\s*(?:as\s+([\w$]+)\s+)?from\s+(['"][^'"]+['"])`)
)

func scriptExports(content []byte) map[string]int {
	names := map[string]int{}
	text := string(content)
	for i, line := range strings.Split(text, "\n") {
		if match := scriptDeclared.FindStringSubmatch(line); match != nil {
			name := match[2]
			if name == "" {
				name = "default"
			}
			names[name] = i + 1
		} else if match := scriptAll.FindStringSubmatch(line); match != nil {
			name := match[1]
			if name == "" {
				name = "everything from " + match[2]
			}
			names[name] = i + 1
		}
	}
	for _, at := range scriptList.FindAllStringSubmatchIndex(text, -1) {
		line := 1 + strings.Count(text[:at[0]], "\n")
		for _, item := range strings.Split(text[at[2]:at[3]], ",") {
			fields := strings.Fields(item)
			if len(fields) > 0 && !strings.HasPrefix(fields[0], "//") {
				names[fields[len(fields)-1]] = line
			}
		}
	}
	return names
}
