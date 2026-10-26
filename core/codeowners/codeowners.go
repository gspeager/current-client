package codeowners

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Style string

const (
	GitHub Style = "github"
	GitLab Style = "gitlab"
)

// Section is a GitLab CODEOWNERS section; GitHub files have one unnamed
// section. Approvals is how many of its owners must approve, 0 for the
// host's default.
type Section struct {
	Name      string
	Optional  bool
	Approvals int
	Default   []string
}

type Owner struct {
	Name      string
	Section   string
	Optional  bool
	Approvals int
}

type rule struct {
	re      *regexp.Regexp
	owners  []string
	section *Section
}

type File struct {
	Style    Style
	Sections []*Section
	rules    []rule
}

// Locations are the paths each host reads, in the order it tries them; a
// repository's file is looked for at its own host's paths first, then the
// other's.
var Locations = map[Style][]string{
	GitHub: {".github/CODEOWNERS", "CODEOWNERS", "docs/CODEOWNERS"},
	GitLab: {".gitlab/CODEOWNERS", "CODEOWNERS", "docs/CODEOWNERS"},
}

// Load reads the first CODEOWNERS file found under root. A repository with
// none gives a nil File and an empty path.
func Load(root string, style Style) (*File, string, error) {
	seen := map[string]bool{}
	for _, s := range []Style{style, other(style)} {
		for _, rel := range Locations[s] {
			if seen[rel] {
				continue
			}
			seen[rel] = true
			content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return nil, "", err
			}
			return Parse(string(content), style), rel, nil
		}
	}
	return nil, "", nil
}

func other(style Style) Style {
	if style == GitLab {
		return GitHub
	}
	return GitLab
}

var sectionHeader = regexp.MustCompile(`^(\^?)\[([^\]]+)\](?:\[(\d+)\])?\s*(.*)$`)

// Parse keeps the file's rules in order. With GitLab's rules, `[Name]`,
// `^[Name]` (optional) and `[Name][2]` (approvals) open sections, with
// default owners after the header; a section named twice is one section.
func Parse(content string, style Style) *File {
	f := &File{Style: style}
	var current *Section
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if style == GitLab {
			if m := sectionHeader.FindStringSubmatch(line); m != nil {
				current = f.section(m[2])
				current.Optional = current.Optional || m[1] == "^"
				if m[3] != "" {
					current.Approvals = atoi(m[3])
				}
				if m[4] != "" {
					current.Default = fields(m[4])
				}
				continue
			}
		}
		parts := fields(strings.TrimPrefix(line, `\`))
		if len(parts) == 0 {
			continue
		}
		owners := parts[1:]
		if len(owners) == 0 && current != nil {
			owners = current.Default
		}
		f.rules = append(f.rules, rule{re: compile(parts[0]), owners: owners, section: current})
	}
	return f
}

func (f *File) section(name string) *Section {
	for _, s := range f.Sections {
		if strings.EqualFold(s.Name, name) {
			return s
		}
	}
	s := &Section{Name: name}
	f.Sections = append(f.Sections, s)
	return s
}

// Match lists who owns path. In each section the last matching rule wins
// and a rule without owners clears the match; a file can be owned in every
// GitLab section at once.
func (f *File) Match(path string) []Owner {
	if f == nil {
		return nil
	}
	path = strings.TrimPrefix(filepath.ToSlash(path), "/")
	type hit struct {
		owners  []string
		section *Section
	}
	var hits []hit
	bySection := map[*Section]int{}
	for _, r := range f.rules {
		if !r.re.MatchString(path) {
			continue
		}
		if i, ok := bySection[r.section]; ok {
			hits[i].owners = r.owners
			continue
		}
		bySection[r.section] = len(hits)
		hits = append(hits, hit{owners: r.owners, section: r.section})
	}
	var out []Owner
	for _, h := range hits {
		for _, name := range h.owners {
			o := Owner{Name: name}
			if h.section != nil {
				o.Section, o.Optional, o.Approvals = h.section.Name, h.section.Optional, h.section.Approvals
			}
			out = append(out, o)
		}
	}
	return out
}

// Owners is Match's names, each once, in order.
func (f *File) Owners(path string) []string {
	var names []string
	seen := map[string]bool{}
	for _, o := range f.Match(path) {
		if !seen[o.Name] {
			seen[o.Name] = true
			names = append(names, o.Name)
		}
	}
	return names
}

// fields splits on whitespace, keeping GitLab's `\ ` escaped spaces.
func fields(s string) []string {
	const mark = "\x00"
	parts := strings.Fields(strings.ReplaceAll(s, `\ `, mark))
	for i := range parts {
		parts[i] = strings.ReplaceAll(parts[i], mark, " ")
	}
	return parts
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}

// compile turns a gitignore-style pattern into a regexp over a slash path.
// A pattern with a slash inside is anchored at the root, one without matches
// at any depth; a trailing slash means a directory. A match on a directory
// covers everything beneath, except that a pattern ending in `*` matches
// only direct children, as both hosts document for `docs/*`.
func compile(pattern string) *regexp.Regexp {
	dirOnly := strings.HasSuffix(pattern, "/")
	p := strings.TrimSuffix(pattern, "/")
	anchored := strings.HasPrefix(p, "/") || strings.Contains(p, "/")
	p = strings.TrimPrefix(p, "/")
	var b strings.Builder
	if anchored {
		b.WriteString("^")
	} else {
		b.WriteString("^(?:.*/)?")
	}
	for i := 0; i < len(p); i++ {
		switch {
		case strings.HasPrefix(p[i:], "**/"):
			b.WriteString("(?:.*/)?")
			i += 2
		case strings.HasPrefix(p[i:], "**"):
			b.WriteString(".*")
			i++
		case p[i] == '*':
			b.WriteString("[^/]*")
		case p[i] == '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(p[i : i+1]))
		}
	}
	switch {
	case dirOnly:
		b.WriteString("/.*$")
	case strings.HasSuffix(p, "*"):
		b.WriteString("$")
	default:
		b.WriteString("(?:/.*)?$")
	}
	return regexp.MustCompile(b.String())
}
