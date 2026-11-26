package structure

import (
	"path"
	"regexp"
	"slices"
	"strings"
)

func init() {
	register(&language{name: "TypeScript and JavaScript", read: readJavaScript, link: linkJavaScript},
		".ts", ".tsx", ".mts", ".cts", ".js", ".jsx", ".mjs", ".cjs")
}

var jsImport = regexp.MustCompile(`(?:^|[;\s}])(?:import|export)\b[^'";` + "`" + `]*?\bfrom\s*["']([^'"\n]+)["']` +
	`|(?:^|[;\s}])import\s*["']([^'"\n]+)["']` +
	`|\b(?:require|import)\s*\(\s*["']([^'"\n]+)["']`)

func readJavaScript(src string) *facts {
	f := &facts{}
	seen := map[string]bool{}
	for _, match := range jsImport.FindAllStringSubmatch(withoutJSComments(src), -1) {
		name := match[1] + match[2] + match[3]
		if !seen[name] {
			seen[name] = true
			f.imports = append(f.imports, imported{path: intern(name)})
		}
	}
	return f
}

// withoutJSComments also empties templates, which can hold anything that looks like
// code. A quote in a regular expression is taken for a string to the end of its line
// at most.
func withoutJSComments(src string) string {
	var out strings.Builder
	out.Grow(len(src))
	for i := 0; i < len(src); i++ {
		c := src[i]
		switch {
		case c == '/' && strings.HasPrefix(src[i:], "//"):
			end := strings.IndexByte(src[i:], '\n')
			if end < 0 {
				return out.String()
			}
			i += end - 1
		case c == '/' && strings.HasPrefix(src[i:], "/*"):
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				return out.String()
			}
			i += end + 3
			out.WriteByte(' ')
		case c == '"' || c == '\'':
			end := i + 1
			for end < len(src) && src[end] != c && src[end] != '\n' {
				if src[end] == '\\' {
					end++
				}
				end++
			}
			end = min(end, len(src)-1)
			out.WriteString(src[i : end+1])
			i = end
		case c == '`':
			end := i + 1
			for end < len(src) && src[end] != '`' {
				if src[end] == '\\' {
					end++
				}
				end++
			}
			out.WriteString("``")
			i = end
		default:
			out.WriteByte(c)
		}
	}
	return out.String()
}

// alias is a name, or the start of one, that stands for a folder or a file.
type alias struct {
	from string
	to   []string
	// The folder an importing file must be in for the alias to apply.
	within string
}

var jsEndings = []string{"", ".ts", ".tsx", ".mts", ".cts", ".js", ".jsx", ".mjs", ".cjs", "/index.ts", "/index.tsx", "/index.js", "/index.jsx", "/index.mjs"}

func (l *linker) jsFile(p string) (int32, bool) {
	p = path.Clean(p)
	for _, ending := range jsEndings {
		if i, ok := l.index[p+ending]; ok && languageOf(p+ending) != nil {
			return i, true
		}
	}
	// TypeScript is imported by what it's compiled to.
	for compiled, sources := range map[string][]string{".js": {".ts", ".tsx"}, ".jsx": {".tsx"}, ".mjs": {".mts"}, ".cjs": {".cts"}} {
		if stem, ok := strings.CutSuffix(p, compiled); ok {
			for _, source := range sources {
				if i, ok := l.index[stem+source]; ok {
					return i, true
				}
			}
		}
	}
	return 0, false
}

func linkJavaScript(l *linker, files []int32) {
	aliases := l.jsAliases()
	for _, i := range files {
		dir := path.Dir(l.files[i])
		for _, imp := range l.facts[i].imports {
			if strings.HasPrefix(imp.path, ".") {
				if to, ok := l.jsFile(path.Join(dir, imp.path)); ok {
					l.add(i, to)
				}
				continue
			}
			if to, ok := l.jsAliased(aliases, l.files[i], imp.path); ok {
				l.add(i, to)
			}
		}
	}
}

func (l *linker) jsAliased(aliases []alias, from, name string) (int32, bool) {
	for _, a := range aliases {
		if a.within != "" && !strings.HasPrefix(from, a.within+"/") {
			continue
		}
		rest, ok := strings.CutPrefix(name, a.from)
		// A base alias has an empty from, which every name starts with.
		if !ok || (rest != "" && a.from != "" && !strings.HasPrefix(rest, "/") && !strings.HasSuffix(a.from, "/")) {
			continue
		}
		for _, to := range a.to {
			if i, ok := l.jsFile(path.Join(to, rest)); ok {
				return i, true
			}
		}
	}
	return 0, false
}

// jsAliases are tsconfig paths and workspace packages, the closest first and then the
// longest name.
func (l *linker) jsAliases() []alias {
	var aliases []alias
	for p, f := range l.config {
		dir := path.Dir(p)
		within := dir
		if within == "." {
			within = ""
		}
		for _, a := range f.aliases {
			resolved := alias{from: a.from, within: within}
			for _, to := range a.to {
				resolved.to = append(resolved.to, path.Join(dir, f.base, to))
			}
			aliases = append(aliases, resolved)
		}
		if f.base != "" && tsconfigName.MatchString(path.Base(p)) {
			// Any name can start from the base.
			aliases = append(aliases, alias{to: []string{path.Join(dir, f.base)}, within: within})
		}
		if path.Base(p) == "package.json" && f.module != "" {
			targets := []string{path.Join(dir, f.base), path.Join(dir, "src"), dir}
			aliases = append(aliases, alias{from: f.module, to: targets})
		}
	}
	slices.SortFunc(aliases, func(a, b alias) int {
		if len(a.within) != len(b.within) {
			return len(b.within) - len(a.within)
		}
		if len(a.from) != len(b.from) {
			return len(b.from) - len(a.from)
		}
		return strings.Compare(a.from+a.within, b.from+b.within)
	})
	return aliases
}
