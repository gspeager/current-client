package structure

import (
	"cmp"
	"path"
	"slices"
	"sort"
	"strings"
)

const (
	// More cards than this can't be followed, and the rest are listed.
	mostCards = 14
	mostSide  = 6
	// Lines drawn until all of them are asked for.
	mostMain = 20
	// A side's groups are opened while there are no more than this.
	mostGroups = 7
)

// View is a graph, and another point it's compared with.
type View struct {
	paths []string
	// Whether each is there now, and then.
	now, then []bool
	links     []viewLink
	compared  bool
	docs      map[string]string
}

type viewLink struct {
	from, to  int32
	now, then bool
}

// NewView compares with then when it's given.
func NewView(now, then *Graph) *View {
	v := &View{compared: then != nil, docs: now.Docs}
	if then == nil {
		v.paths = now.Files
		v.now = filled(len(v.paths))
		v.links = make([]viewLink, len(now.Links))
		for i, l := range now.Links {
			v.links[i] = viewLink{from: l[0], to: l[1], now: true}
		}
		return v
	}
	v.paths = slices.Concat(now.Files, then.Files)
	slices.Sort(v.paths)
	v.paths = slices.Compact(v.paths)
	v.now, v.then = make([]bool, len(v.paths)), make([]bool, len(v.paths))
	at := make(map[string]int32, len(v.paths))
	for i, p := range v.paths {
		at[p] = int32(i)
	}
	links := map[[2]int32]int{}
	place := func(g *Graph, present []bool, mark func(l *viewLink)) {
		for _, p := range g.Files {
			present[at[p]] = true
		}
		for _, l := range g.Links {
			key := [2]int32{at[g.Files[l[0]]], at[g.Files[l[1]]]}
			i, known := links[key]
			if !known {
				i = len(v.links)
				links[key] = i
				v.links = append(v.links, viewLink{from: key[0], to: key[1]})
			}
			mark(&v.links[i])
		}
	}
	place(now, v.now, func(l *viewLink) { l.now = true })
	place(then, v.then, func(l *viewLink) { l.then = true })
	return v
}

func filled(n int) []bool {
	all := make([]bool, n)
	for i := range all {
		all[i] = true
	}
	return all
}

const (
	Card   = "card"
	UsedBy = "used-by"
	Uses   = "uses"

	Added   = "added"
	Removed = "removed"
)

// Node is a card, or a place outside the level that uses it or is used by it.
type Node struct {
	Path   string
	Label  string
	Side   string
	Folder bool
	// The files there now.
	Files []string
	// Added or removed since the point compared with.
	State string
	Doc   string
	// Which of the colors a card's lines have, from 1.
	Slot int
}

// Link is between nodes by index. Main ones are drawn when there are too many to
// follow.
type Link struct {
	From, To int
	Count    int
	Main     bool
	State    string
}

// Wide is a place so many of a level's cards share that it's said in words.
type Wide struct {
	Path   string
	Label  string
	Folder bool
	Cards  int
}

type Crumb struct {
	Path  string
	Label string
}

type Level struct {
	Path   string
	Crumbs []Crumb
	Doc    string
	Folder bool
	// Every file in it now.
	Files []string
	Nodes []Node
	// Cards there was no room for, the most connected first.
	Rest  []Node
	Links []Link
	// Cards in all, which Wide.Cards is of.
	Cards                    int
	WidelyUsedBy, WidelyUses []Wide
	MoreUsedBy, MoreUses     int
}

func (v *View) within(dir string) (from, to int) {
	if dir == "" {
		return 0, len(v.paths)
	}
	from = sort.SearchStrings(v.paths, dir)
	if from < len(v.paths) && v.paths[from] == dir {
		return from, from + 1
	}
	from = sort.SearchStrings(v.paths, dir+"/")
	// "0" comes after "/".
	return from, sort.SearchStrings(v.paths, dir+"0")
}

func (v *View) isFile(p string) bool {
	_, found := slices.BinarySearch(v.paths, p)
	return found
}

func under(p, dir string) (rest string, ok bool) {
	if dir == "" {
		return p, true
	}
	return strings.CutPrefix(p, dir+"/")
}

// child is what directly in dir holds p.
func child(p, dir string) string {
	rest, ok := under(p, dir)
	if !ok {
		return p
	}
	if first, _, nested := strings.Cut(rest, "/"); nested {
		return path.Join(dir, first)
	}
	return p
}

// step goes past folders that hold only one folder.
func (v *View) step(dir string) string {
	for !v.isFile(dir) {
		from, to := v.within(dir)
		if from >= to {
			return dir
		}
		only := child(v.paths[from], dir)
		// Sorted, so when the first and the last are in one, all are.
		if child(v.paths[to-1], dir) != only || v.isFile(only) {
			return dir
		}
		dir = only
	}
	return dir
}

func (v *View) label(p, dir string) string {
	name, _ := under(p, dir)
	if v.isFile(p) {
		return path.Base(p)
	}
	return name + "/"
}

func (v *View) crumbs(dir string) []Crumb {
	crumbs := []Crumb{{}}
	last := ""
	parts := strings.Split(dir, "/")
	for i := range parts {
		if p := strings.Join(parts[:i+1], "/"); p != "" && (v.step(p) == p || p == dir) {
			crumbs = append(crumbs, Crumb{Path: p, Label: strings.TrimSuffix(v.label(p, last), "/")})
			last = p
		}
	}
	return crumbs
}

// count is how many links there are now and were then.
type count struct{ now, then int }

func (c *count) add(l viewLink) {
	if l.now {
		c.now++
	}
	if l.then {
		c.then++
	}
}

func (c count) most() int { return max(c.now, c.then) }

func (v *View) state(now, then int) string {
	switch {
	case !v.compared || (now > 0 && then > 0):
		return ""
	case now > 0:
		return Added
	}
	return Removed
}

// Level is what's inside a folder, or a file by itself, with what uses it and what it
// uses. A path that isn't there has no cards.
func (v *View) Level(dir string) Level {
	dir = v.step(dir)
	from, to := v.within(dir)
	level := Level{Path: dir, Crumbs: v.crumbs(dir), Doc: v.docs[dir], Folder: !v.isFile(dir)}
	cards, owner := v.cards(dir, from, to)
	level.Cards = len(cards)
	for i := from; i < to; i++ {
		if v.now[i] {
			level.Files = append(level.Files, v.paths[i])
		}
	}

	inside := map[[2]int]*count{}
	left, right := map[int32]map[int]*count{}, map[int32]map[int]*count{}
	in := func(i int32) bool { return int(i) >= from && int(i) < to }
	tally := func(side map[int32]map[int]*count, file int32, card int, l viewLink) {
		if side[file] == nil {
			side[file] = map[int]*count{}
		}
		if side[file][card] == nil {
			side[file][card] = &count{}
		}
		side[file][card].add(l)
	}
	for _, l := range v.links {
		a, b := in(l.from), in(l.to)
		switch {
		case a && b:
			key := [2]int{owner[int(l.from)-from], owner[int(l.to)-from]}
			if key[0] == key[1] {
				continue
			}
			if inside[key] == nil {
				inside[key] = &count{}
			}
			inside[key].add(l)
		case b:
			tally(left, l.from, owner[int(l.to)-from], l)
		case a:
			tally(right, l.to, owner[int(l.from)-from], l)
		}
	}

	usedBy, uses := v.group(left, dir), v.group(right, dir)
	level.WidelyUsedBy, usedBy = widely(usedBy, len(cards))
	level.WidelyUses, uses = widely(uses, len(cards))

	weight := make([]int, len(cards))
	for key, c := range inside {
		weight[key[0]] += c.most()
		weight[key[1]] += c.most()
	}
	for _, side := range [][]sideGroup{usedBy, uses} {
		for _, g := range side {
			for card, c := range g.cards {
				weight[card] += c.most()
			}
		}
	}
	order := make([]int, len(cards))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int {
		return cmp.Or(cmp.Compare(weight[b], weight[a]), cmp.Compare(len(cards[b].Files), len(cards[a].Files)))
	})
	at := map[int]int{}
	for rank, i := range order {
		card := cards[i]
		if rank >= mostCards {
			level.Rest = append(level.Rest, card)
			continue
		}
		card.Slot = rank%8 + 1
		at[i] = len(level.Nodes)
		level.Nodes = append(level.Nodes, card)
	}
	for key, c := range inside {
		a, shownA := at[key[0]]
		b, shownB := at[key[1]]
		if shownA && shownB {
			level.Links = append(level.Links, Link{From: a, To: b, Count: c.most(), State: v.state(c.now, c.then)})
		}
	}
	level.MoreUsedBy = v.addSide(&level, usedBy, UsedBy, at)
	level.MoreUses = v.addSide(&level, uses, Uses, at)
	slices.SortFunc(level.Links, func(a, b Link) int { return cmp.Or(a.From-b.From, a.To-b.To) })
	chooseMain(level.Links)
	return level
}

func (v *View) cards(dir string, from, to int) (cards []Node, owner []int) {
	owner = make([]int, to-from)
	if v.isFile(dir) {
		if from < to {
			cards = []Node{v.node(dir, path.Dir(dir), from, to)}
		}
		return cards, owner
	}
	for i := from; i < to; {
		p := child(v.paths[i], dir)
		end := i + 1
		if p != v.paths[i] {
			_, end = v.within(p)
		}
		for k := i; k < end; k++ {
			owner[k-from] = len(cards)
		}
		cards = append(cards, v.node(v.step(p), dir, i, end))
		i = end
	}
	return cards, owner
}

func (v *View) node(p, dir string, from, to int) Node {
	if dir == "." {
		dir = ""
	}
	node := Node{Path: p, Label: v.label(p, dir), Side: Card, Folder: !v.isFile(p), Doc: v.docs[p]}
	then := 0
	for i := from; i < to; i++ {
		if v.now[i] {
			node.Files = append(node.Files, v.paths[i])
		}
		if v.compared && v.then[i] {
			then++
		}
	}
	node.State = v.state(len(node.Files), then)
	return node
}

// sideGroup is a place outside the level, and the cards it has to do with.
type sideGroup struct {
	path  string
	cards map[int]*count
	total int
}

// group gathers files outside dir by where they are: first by what stands beside dir
// or beside something above it, then opening the largest while there are few.
func (v *View) group(files map[int32]map[int]*count, dir string) []sideGroup {
	groups := map[string][]int32{}
	for file := range files {
		p := v.beside(v.paths[file], dir)
		groups[p] = append(groups[p], file)
	}
	for {
		largest := ""
		for p, members := range groups {
			if !v.isFile(p) && (largest == "" || len(members) > len(groups[largest]) || (len(members) == len(groups[largest]) && p < largest)) {
				largest = p
			}
		}
		if largest == "" || len(groups[largest])*10 < len(files)*3 {
			break
		}
		opened := map[string][]int32{}
		names := 0
		for _, file := range groups[largest] {
			p := v.step(child(v.paths[file], largest))
			if opened[p] == nil && v.isFile(p) {
				names++
			}
			opened[p] = append(opened[p], file)
		}
		if len(groups)-1+len(opened) > mostGroups || (names > 0 && len(opened) > 3) {
			break
		}
		delete(groups, largest)
		for p, members := range opened {
			groups[p] = members
		}
	}
	var list []sideGroup
	for p, members := range groups {
		g := sideGroup{path: p, cards: map[int]*count{}}
		for _, file := range members {
			for card, c := range files[file] {
				if g.cards[card] == nil {
					g.cards[card] = &count{}
				}
				g.cards[card].now += c.now
				g.cards[card].then += c.then
				g.total += c.most()
			}
		}
		list = append(list, g)
	}
	slices.SortFunc(list, func(a, b sideGroup) int { return cmp.Or(b.total-a.total, strings.Compare(a.path, b.path)) })
	return list
}

func (v *View) beside(file, dir string) string {
	parts, own := strings.Split(file, "/"), strings.Split(dir, "/")
	k := 0
	for k < len(parts)-1 && k < len(own) && parts[k] == own[k] {
		k++
	}
	return v.step(strings.Join(parts[:k+1], "/"))
}

// widely takes out what most cards have to do with.
func widely(groups []sideGroup, cards int) (wide []Wide, rest []sideGroup) {
	for _, g := range groups {
		if cards > 3 && len(g.cards) > 3 && len(g.cards)*10 >= cards*4 {
			wide = append(wide, Wide{Path: g.path, Cards: len(g.cards)})
		} else {
			rest = append(rest, g)
		}
	}
	return wide, rest
}

func (v *View) addSide(level *Level, groups []sideGroup, side string, at map[int]int) (more int) {
	shown := 0
	for _, g := range groups {
		var links []Link
		for card, c := range g.cards {
			if i, ok := at[card]; ok {
				links = append(links, Link{From: i, To: i, Count: c.most(), State: v.state(c.now, c.then)})
			}
		}
		if len(links) == 0 || shown == mostSide {
			more++
			continue
		}
		shown++
		from, to := v.within(g.path)
		node := v.node(g.path, path.Dir(path.Dir(g.path)), from, to)
		node.Side = side
		for _, l := range links {
			if side == UsedBy {
				l.From = len(level.Nodes)
			} else {
				l.To = len(level.Nodes)
			}
			level.Links = append(level.Links, l)
		}
		level.Nodes = append(level.Nodes, node)
	}
	for _, wide := range [][]Wide{level.WidelyUsedBy, level.WidelyUses} {
		for i := range wide {
			wide[i].Label = v.label(wide[i].Path, strings.TrimPrefix(path.Dir(path.Dir(wide[i].Path)), "."))
			wide[i].Folder = !v.isFile(wide[i].Path)
		}
	}
	return more
}

// chooseMain keeps what changed, each node's strongest link each way, and then the
// strongest of the rest, up to the limit.
func chooseMain(links []Link) {
	if len(links) <= mostMain {
		for i := range links {
			links[i].Main = true
		}
		return
	}
	order := make([]int, len(links))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return links[b].Count - links[a].Count })
	left := mostMain
	take := func(i int) {
		if left > 0 && !links[i].Main {
			links[i].Main = true
			left--
		}
	}
	for _, i := range order {
		if links[i].State != "" {
			take(i)
		}
	}
	into, out := map[int]bool{}, map[int]bool{}
	for _, i := range order {
		if l := links[i]; !into[l.To] || !out[l.From] {
			into[l.To], out[l.From] = true, true
			take(i)
		}
	}
	for _, i := range order {
		take(i)
	}
}
