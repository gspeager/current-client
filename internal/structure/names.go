package structure

import (
	"path"
	"regexp"
	"slices"
	"strings"
)

// C# and Swift don't name files, so a type's name is matched to where it's defined,
// when the file can see it there: by using, by import or by its own namespace.

func init() {
	register(&language{name: "C#", read: readCSharp, link: linkCSharp}, ".cs")
	register(&language{name: "Swift", read: readSwift, link: linkSwift}, ".swift")
}

var (
	typeDefined = regexp.MustCompile(`\b(?:class|interface|protocol|struct|enum|actor|typealias|delegate\s+[\w.<>\[\],?\s]+?|record(?:\s+(?:struct|class))?)\s+([A-Z]\w+)`)
	typeUsed    = regexp.MustCompile(`(?:^|[^\w.])([A-Z]\w+)\b`)
	csNamespace = regexp.MustCompile(`(?m)^\s*namespace\s+([\w.]+)\s*([;{])?`)
	csUsing     = regexp.MustCompile(`(?m)^\s*(global\s+)?using\s+(?:static\s+)?(?:\w+\s*=\s*)?([\w.]+)\s*;`)
	swiftImport = regexp.MustCompile(`(?m)^\s*(?:@\w+(?:\(\w+\))?\s+)*(?:public\s+|internal\s+|package\s+)?import\s+(?:struct |class |enum |protocol |func )?(\w+)`)
)

// withoutCComments also empties strings.
func withoutCComments(src string, verbatim bool) string {
	var out strings.Builder
	out.Grow(len(src))
	for i := 0; i < len(src); {
		rest := src[i:]
		end := -1
		switch {
		case strings.HasPrefix(rest, "//"):
			end = strings.IndexByte(rest, '\n')
		case strings.HasPrefix(rest, "/*"):
			if end = strings.Index(rest[2:], "*/"); end >= 0 {
				end += 4
			}
		case strings.HasPrefix(rest, `"""`):
			if end = strings.Index(rest[3:], `"""`); end >= 0 {
				end += 6
			}
		case verbatim && (strings.HasPrefix(rest, `@"`) || strings.HasPrefix(rest, `$@"`) || strings.HasPrefix(rest, `@$"`)):
			end = strings.IndexByte(rest, '"') + 1
			for end < len(rest) && (rest[end] != '"' || strings.HasPrefix(rest[end:], `""`)) {
				if rest[end] == '"' {
					end++
				}
				end++
			}
			end++
		case rest[0] == '"':
			end = 1
			for end < len(rest) && rest[end] != '"' && rest[end] != '\n' {
				if rest[end] == '\\' {
					end++
				}
				end++
			}
			end++
		case verbatim && rest[0] == '\'' && len(rest) > 2:
			// A character, such as '"'.
			if rest[1] == '\\' && len(rest) > 3 && rest[3] == '\'' {
				end = 4
			} else if rest[2] == '\'' {
				end = 3
			}
			if end < 0 {
				end = 1
			}
		default:
			out.WriteByte(rest[0])
			i++
			continue
		}
		if end < 0 || end > len(rest) {
			end = len(rest)
		}
		out.WriteByte(' ')
		i += end
	}
	return out.String()
}

// topTypes are the types defined no deeper than base braces.
func topTypes(code string, base int) map[string]bool {
	found := map[string]bool{}
	depth, at := 0, 0
	for _, match := range typeDefined.FindAllStringSubmatchIndex(code, -1) {
		depth += strings.Count(code[at:match[0]], "{") - strings.Count(code[at:match[0]], "}")
		at = match[0]
		if depth <= base {
			found[code[match[2]:match[3]]] = true
		}
	}
	return found
}

func usedTypes(code string, own map[string]bool) []string {
	used := map[string]bool{}
	for _, match := range typeUsed.FindAllStringSubmatch(code, -1) {
		if !own[match[1]] {
			used[match[1]] = true
		}
	}
	return sorted(used)
}

func readCSharp(src string) *facts {
	code := withoutCComments(src, true)
	f := &facts{}
	base := 0
	if match := csNamespace.FindStringSubmatch(code); match != nil {
		f.scope = intern(match[1])
		if match[2] != ";" {
			base = 1
		}
	}
	sees := map[string]bool{}
	for _, match := range csUsing.FindAllStringSubmatch(code, -1) {
		sees[match[2]] = true
		if match[1] != "" {
			f.imports = append(f.imports, imported{path: intern(match[2])})
		}
	}
	own := topTypes(code, base)
	f.defines, f.sees = sorted(own), sorted(sees)
	f.uses = usedTypes(csUsing.ReplaceAllString(code, ""), own)
	return f
}

func linkCSharp(l *linker, files []int32) {
	defined := map[string][]int32{}
	// A global using is seen by its whole project.
	global := map[string][]string{}
	projectOf := l.nearest(func(p string) bool { return strings.HasSuffix(p, ".csproj") })
	for _, i := range files {
		for _, name := range l.facts[i].defines {
			defined[name] = append(defined[name], i)
		}
		for _, imp := range l.facts[i].imports {
			project := projectOf(l.files[i])
			global[project] = append(global[project], imp.path)
		}
	}
	for _, i := range files {
		f := l.facts[i]
		globals := global[projectOf(l.files[i])]
		sees := func(to int32) bool {
			scope := l.facts[to].scope
			if scope == "" || scope == f.scope || strings.HasPrefix(f.scope, scope+".") ||
				slices.Contains(f.sees, scope) || slices.Contains(globals, scope) {
				return true
			}
			// using static, or an alias for the type itself.
			return slices.ContainsFunc(l.facts[to].defines, func(name string) bool { return slices.Contains(f.sees, scope+"."+name) })
		}
		l.linkSeen(i, defined, sees)
	}
}

// linkSeen matches each name to the one or two files that define it where this one
// can see it. Two are a partial type.
func (l *linker) linkSeen(from int32, defined map[string][]int32, sees func(to int32) bool) {
	for _, name := range l.facts[from].uses {
		// An attribute is named without its ending.
		if len(defined[name]) == 0 {
			name += "Attribute"
		}
		var seen []int32
		for _, to := range defined[name] {
			if to != from && sees(to) {
				seen = append(seen, to)
			}
		}
		if len(seen) <= 2 {
			for _, to := range seen {
				l.add(from, to)
			}
		}
	}
}

// nearest answers with the closest folder above a file that holds one that matches,
// or "" for none.
func (l *linker) nearest(matches func(p string) bool) func(file string) string {
	holders := map[string]bool{}
	for _, p := range l.files {
		if matches(p) {
			holders[path.Dir(p)] = true
		}
	}
	return func(file string) string {
		for dir := path.Dir(file); ; dir = path.Dir(dir) {
			if holders[dir] {
				return dir
			}
			if dir == "." {
				return ""
			}
		}
	}
}

func readSwift(src string) *facts {
	code := withoutCComments(src, false)
	sees := map[string]bool{}
	for _, match := range swiftImport.FindAllStringSubmatch(code, -1) {
		sees[match[1]] = true
	}
	own := topTypes(code, 0)
	return &facts{defines: sorted(own), sees: sorted(sees), uses: usedTypes(swiftImport.ReplaceAllString(code, ""), own)}
}

var swiftTarget = regexp.MustCompile(`(?:^|/)(?:Sources|Source|Tests)/([^/]+)/`)

func linkSwift(l *linker, files []int32) {
	packageOf := l.nearest(func(p string) bool { return path.Base(p) == "Package.swift" })
	defined := map[string][]int32{}
	// A package's target is a module. What's in no package is taken as one, as an
	// app's own targets are.
	module := make(map[int32]string, len(files))
	for _, i := range files {
		p := l.files[i]
		if path.Base(p) == "Package.swift" {
			module[i] = "Package.swift in " + path.Dir(p)
			continue
		}
		switch match := swiftTarget.FindStringSubmatch(p); {
		case match != nil && packageOf(p) != "":
			module[i] = match[1]
		case packageOf(p) != "":
			module[i] = path.Base(packageOf(p))
		}
		for _, name := range l.facts[i].defines {
			defined[name] = append(defined[name], i)
		}
	}
	for _, i := range files {
		f := l.facts[i]
		l.linkSeen(i, defined, func(to int32) bool {
			return module[to] == module[i] || (module[to] != "" && slices.Contains(f.sees, module[to]))
		})
	}
}
