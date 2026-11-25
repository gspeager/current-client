package structure

import (
	"path"
	"regexp"
	"slices"
	"strings"
)

func init() {
	register(&language{name: "Python", read: readPython, link: linkPython}, ".py", ".pyi")
}

var (
	pyTriple = regexp.MustCompile(`(?s)""".*?"""|'''.*?'''`)
	pyImport = regexp.MustCompile(`(?m)^[ \t]*import[ \t]+([^\n#;]+)`)
	pyFrom   = regexp.MustCompile(`(?m)^[ \t]*from[ \t]+([.\w]+)[ \t]+import[ \t]*(\([^)]*\)|(?:[^\n#\\]|\\\r?\n)+)`)
	pyNoise  = regexp.MustCompile(`#[^\n]*|[()\\]`)
)

func readPython(src string) *facts {
	f := &facts{}
	code := pyTriple.ReplaceAllString(src, `""`)
	for _, match := range pyImport.FindAllStringSubmatch(code, -1) {
		for module := range strings.SplitSeq(match[1], ",") {
			if name := beforeAs(module); name != "" {
				f.imports = append(f.imports, imported{path: intern(name)})
			}
		}
	}
	for _, match := range pyFrom.FindAllStringSubmatch(code, -1) {
		base := strings.TrimLeft(match[1], ".")
		imp := imported{path: intern(base), up: len(match[1]) - len(base)}
		for name := range strings.SplitSeq(pyNoise.ReplaceAllString(match[2], ""), ",") {
			if name = beforeAs(name); name != "" {
				imp.names = append(imp.names, intern(name))
			}
		}
		f.imports = append(f.imports, imp)
	}
	return f
}

func beforeAs(name string) string {
	fields := strings.Fields(name)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func linkPython(l *linker, files []int32) {
	isPackage := map[string]bool{}
	for _, i := range files {
		if path.Base(l.files[i]) == "__init__.py" {
			isPackage[path.Dir(l.files[i])] = true
		}
	}
	// Where a name can start: the top, and whatever holds a package and isn't one.
	roots := []string{"."}
	for dir := range isPackage {
		if parent := path.Dir(dir); parent != "." && !isPackage[parent] && !slices.Contains(roots, parent) {
			roots = append(roots, parent)
		}
	}
	slices.Sort(roots[1:])
	at := func(p string) (int32, bool) {
		return l.find(p+".py", p+"/__init__.py", p+".pyi", p+"/__init__.pyi")
	}
	module := func(from, name string) (int32, bool) {
		p := strings.ReplaceAll(name, ".", "/")
		// A script's own folder comes first.
		if dir := path.Dir(from); !isPackage[dir] {
			if i, ok := at(path.Join(dir, p)); ok {
				return i, true
			}
		}
		first, _, _ := strings.Cut(p, "/")
		for _, root := range roots {
			// Elsewhere only a package is named, not a script that lies beside one.
			if root != "." && !strings.HasPrefix(from, root+"/") && !isPackage[path.Join(root, first)] {
				continue
			}
			if i, ok := at(path.Join(root, p)); ok {
				return i, true
			}
		}
		return 0, false
	}
	for _, i := range files {
		file := l.files[i]
		for _, imp := range l.facts[i].imports {
			if imp.up == 0 && len(imp.names) == 0 {
				if to, ok := module(file, imp.path); ok {
					l.add(i, to)
				}
				continue
			}
			for _, name := range imp.names {
				var to int32
				var ok bool
				if imp.up > 0 {
					dir := path.Dir(file)
					for range imp.up - 1 {
						dir = path.Dir(dir)
					}
					dir = path.Join(dir, strings.ReplaceAll(imp.path, ".", "/"))
					if to, ok = at(path.Join(dir, name)); !ok {
						to, ok = at(dir)
					}
				} else if to, ok = module(file, imp.path+"."+name); !ok {
					to, ok = module(file, imp.path)
				}
				if ok {
					l.add(i, to)
				}
			}
		}
	}
}
