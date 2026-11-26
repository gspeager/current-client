package structure

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"strconv"
	"strings"
)

func init() {
	register(&language{name: "Go", read: readGo, link: linkGo}, ".go")
}

// readGo keeps what Go's parser read of a file it couldn't parse in full.
func readGo(src string) *facts {
	file, _ := parser.ParseFile(token.NewFileSet(), "", src, parser.ParseComments)
	if file == nil || file.Name == nil {
		return &facts{}
	}
	f := &facts{scope: intern(file.Name.Name)}
	if file.Doc != nil {
		f.doc = goPackageDoc(file.Doc.Text())
	}
	defines, uses, selected := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			if decl.Recv == nil {
				defines[decl.Name.Name] = true
			}
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					defines[spec.Name.Name] = true
				case *ast.ValueSpec:
					for _, name := range spec.Names {
						defines[name.Name] = true
					}
				}
			}
		}
	}
	// file.Unresolved is the identifiers the file doesn't declare itself.
	unresolved := map[*ast.Ident]bool{}
	for _, ident := range file.Unresolved { //nolint:staticcheck
		unresolved[ident] = true
		uses[ident.Name] = true
	}
	ast.Inspect(file, func(node ast.Node) bool {
		if selector, ok := node.(*ast.SelectorExpr); ok {
			if ident, ok := selector.X.(*ast.Ident); ok && unresolved[ident] {
				selected[ident.Name+"."+selector.Sel.Name] = true
			}
		}
		return true
	})
	f.defines, f.uses, f.selected = sorted(defines), sorted(uses), sorted(selected)
	for _, spec := range file.Imports {
		p, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		imp := imported{path: intern(p)}
		if spec.Name != nil {
			imp.alias = intern(spec.Name.Name)
		}
		f.imports = append(f.imports, imp)
	}
	return f
}

func goPackageDoc(text string) string {
	_, rest, ok := strings.Cut(strings.Join(strings.Fields(text), " "), " ")
	if !ok || !strings.HasPrefix(text, "Package ") {
		return ""
	}
	// After "Package name".
	_, rest, _ = strings.Cut(rest, " ")
	return firstSentence(rest)
}

func linkGo(l *linker, files []int32) {
	modules := map[string]string{}
	for p, f := range l.config {
		if path.Base(p) == "go.mod" && f.module != "" {
			modules[f.module] = path.Dir(p)
		}
	}
	type pkg struct {
		files []int32
		names map[string][]int32
		scope string
	}
	packages := map[string]*pkg{}
	for _, i := range files {
		dir := path.Dir(l.files[i])
		p := packages[dir]
		if p == nil {
			p = &pkg{names: map[string][]int32{}}
			packages[dir] = p
		}
		p.files = append(p.files, i)
		if p.scope == "" || p.scope == "main" {
			p.scope = l.facts[i].scope
		}
		for _, name := range l.facts[i].defines {
			p.names[name] = append(p.names[name], i)
		}
	}
	folderOf := func(importPath string) (string, bool) {
		for prefix := importPath; prefix != "." && prefix != "/"; prefix = path.Dir(prefix) {
			if root, ok := modules[prefix]; ok {
				dir := path.Join(root, strings.TrimPrefix(strings.TrimPrefix(importPath, prefix), "/"))
				return dir, packages[dir] != nil
			}
		}
		return "", false
	}
	for _, i := range files {
		f := l.facts[i]
		own := packages[path.Dir(l.files[i])]
		for _, name := range f.uses {
			linkNamed(l, i, own.names[name])
		}
		for _, imp := range f.imports {
			dir, ok := folderOf(imp.path)
			if !ok {
				continue
			}
			target := packages[dir]
			if imp.alias == "." {
				for _, name := range f.uses {
					linkNamed(l, i, target.names[name])
				}
				continue
			}
			local := imp.alias
			if local == "" {
				local = target.scope
			}
			found := false
			for _, qualified := range f.selected {
				if name, ok := strings.CutPrefix(qualified, local+"."); ok && len(target.names[name]) > 0 {
					linkNamed(l, i, target.names[name])
					found = true
				}
			}
			// No name used from it is known, as for an import for its side effects:
			// the link goes to the package's first file.
			if !found {
				l.add(i, target.files[0])
			}
		}
	}
}

// Files in one package that may define the same name, as one per platform behind
// build tags. A name defined in more is too common to say which is meant, so it
// links to none.
const mostDefining = 4

func linkNamed(l *linker, from int32, defining []int32) {
	if len(defining) > mostDefining {
		return
	}
	for _, to := range defining {
		l.add(from, to)
	}
}
