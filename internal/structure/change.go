package structure

import (
	"cmp"
	"path"
	"slices"
	"strings"

	"github.com/gspeager/current-client/core/filekind"
)

// Changed is a file a change adds, modifies, renames or deletes.
type Changed struct {
	Path     string
	OrigPath string
	// A, M, D, R or T, with a rename's similarity.
	Status  string
	Added   int
	Removed int
}

// Area is a folder whose own files, not its subfolders', the change touches.
type Area struct {
	Path string
	// Added or Removed when the folder is new or gone.
	State   string
	Files   []string
	Added   int
	Removed int
}

// AreaLink is between two folders, the first using the second, and how many of
// their files' links the change adds or removes.
type AreaLink struct {
	From, To string
	Count    int
	State    string
}

// Reason is a folder or a file to read first or skim, and why.
type Reason struct {
	Path   string
	Folder bool
	// "used" by Count areas or files, a file kind such as "generated", "renamed",
	// or "few" lines, Count of them.
	Why   string
	Count int
}

// Shape is what a change does to the structure.
type Shape struct {
	Areas     []Area
	Links     []AreaLink
	ReadFirst []Reason
	Skim      []Reason
}

const (
	// Areas listed to read first, and files listed apart from them, the most used
	// first, so the list stays a starting point rather than the whole change.
	mostReadFirst = 5
	// Files that use a changed file before it's listed to read first; fewer is an
	// ordinary dependency, not one much of the code rests on.
	widelyUsed = 10
	// Lines added and removed, all in all, below which an area is skimmed.
	fewLines = 5
)

// CompareChange is a change's areas, the links between folders it adds or removes,
// what to read first and what to skim. A graph is nil where its point can't be read,
// and then there are no links.
func CompareChange(now, then *Graph, changes []Changed) Shape {
	var shape Shape
	areas := map[string]*Area{}
	area := func(file string) *Area {
		dir := folderOf(file)
		if areas[dir] == nil {
			areas[dir] = &Area{Path: dir}
		}
		return areas[dir]
	}
	for _, c := range changes {
		a := area(c.Path)
		a.Files = append(a.Files, c.Path)
		a.Added += c.Added
		a.Removed += c.Removed
		if c.OrigPath != "" && folderOf(c.OrigPath) != a.Path {
			area(c.OrigPath).Files = append(area(c.OrigPath).Files, c.OrigPath)
		}
	}
	for _, a := range areas {
		if now != nil && then != nil {
			a.State = state(now.Folders[a.Path], then.Folders[a.Path])
		}
		shape.Areas = append(shape.Areas, *a)
	}
	slices.SortFunc(shape.Areas, func(a, b Area) int { return cmp.Compare(a.Path, b.Path) })

	if now != nil && then != nil {
		shape.Links = changedLinks(folderLinks(now), folderLinks(then))
		shape.ReadFirst = readFirst(now, areas, changes)
	}
	shape.Skim = skim(shape.Areas, changes)
	return shape
}

func folderOf(file string) string {
	if dir := path.Dir(file); dir != "." {
		return dir
	}
	return ""
}

func state(now, then bool) string {
	switch {
	case now && !then:
		return Added
	case then && !now:
		return Removed
	}
	return ""
}

func folderLinks(g *Graph) map[[2]string]int {
	links := map[[2]string]int{}
	for _, l := range g.Links {
		from, to := folderOf(g.Files[l[0]]), folderOf(g.Files[l[1]])
		if from != to {
			links[[2]string{from, to}]++
		}
	}
	return links
}

func changedLinks(now, then map[[2]string]int) []AreaLink {
	var links []AreaLink
	for key, n := range now {
		if then[key] == 0 {
			links = append(links, AreaLink{From: key[0], To: key[1], Count: n, State: Added})
		}
	}
	for key, n := range then {
		if now[key] == 0 {
			links = append(links, AreaLink{From: key[0], To: key[1], Count: n, State: Removed})
		}
	}
	slices.SortFunc(links, func(a, b AreaLink) int {
		return cmp.Or(b.Count-a.Count, cmp.Compare(a.From, b.From), cmp.Compare(a.To, b.To))
	})
	return links
}

// readFirst is the change's areas that its other areas use, and its files that at
// least widelyUsed files use.
func readFirst(now *Graph, areas map[string]*Area, changes []Changed) []Reason {
	users := map[string]map[string]bool{}
	byFile := map[string]map[string]bool{}
	for _, l := range now.Links {
		from, to := now.Files[l[0]], now.Files[l[1]]
		if byFile[to] == nil {
			byFile[to] = map[string]bool{}
		}
		byFile[to][from] = true
		user, used := folderOf(from), folderOf(to)
		if user != used && areas[user] != nil && areas[used] != nil {
			if users[used] == nil {
				users[used] = map[string]bool{}
			}
			users[used][user] = true
		}
	}
	var folders, files []Reason
	for used, by := range users {
		folders = append(folders, Reason{Path: used, Folder: true, Why: "used", Count: len(by)})
	}
	for _, c := range changes {
		if n := len(byFile[c.Path]); n >= widelyUsed && !strings.HasPrefix(c.Status, "D") {
			files = append(files, Reason{Path: c.Path, Why: "used", Count: n})
		}
	}
	most := func(list []Reason) []Reason {
		slices.SortFunc(list, func(a, b Reason) int { return cmp.Or(b.Count-a.Count, cmp.Compare(a.Path, b.Path)) })
		return list[:min(len(list), mostReadFirst)]
	}
	return append(most(folders), most(files)...)
}

func skim(areas []Area, changes []Changed) []Reason {
	var reasons []Reason
	for _, c := range changes {
		switch kind := filekind.Of(c.Path); {
		case kind == filekind.Lockfile || kind == filekind.Generated:
			reasons = append(reasons, Reason{Path: c.Path, Why: string(kind)})
		case strings.HasPrefix(c.Status, "R") && c.Added+c.Removed == 0:
			reasons = append(reasons, Reason{Path: c.Path, Why: "renamed"})
		}
	}
	for _, a := range areas {
		if a.State == "" && a.Added+a.Removed > 0 && a.Added+a.Removed < fewLines {
			reasons = append(reasons, Reason{Path: a.Path, Folder: true, Why: "few", Count: a.Added + a.Removed})
		}
	}
	return reasons
}
