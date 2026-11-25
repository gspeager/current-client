package app

import (
	"context"
	"sync"

	"github.com/gspeager/current-client/core/board"
	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
	"github.com/gspeager/current-client/internal/structure"
)

// StructureInfo is a level of the code's structure. Compared is whether it's compared
// with another point, whose target is a branch's. Partial is true where that point
// would have to be fetched, so it isn't.
type StructureInfo struct {
	Path   string           `json:"path"`
	Crumbs []StructureCrumb `json:"crumbs"`
	Doc    string           `json:"doc"`
	Folder bool             `json:"folder"`
	Count  StructureCount   `json:"count"`
	Nodes  []StructureNode  `json:"nodes"`
	// Cards there was no room for.
	Rest         []StructureNode `json:"rest"`
	Links        []StructureLink `json:"links"`
	Cards        int             `json:"cards"`
	WidelyUsedBy []StructureWide `json:"widelyUsedBy"`
	WidelyUses   []StructureWide `json:"widelyUses"`
	MoreUsedBy   int             `json:"moreUsedBy"`
	MoreUses     int             `json:"moreUses"`
	Languages    []string        `json:"languages"`
	// Files of code in a language that isn't read.
	Unread   int    `json:"unread"`
	Compared bool   `json:"compared"`
	Target   string `json:"target"`
	Partial  bool   `json:"partial"`
}

type StructureCrumb struct {
	Path  string `json:"path"`
	Label string `json:"label"`
}

type StructureCount struct {
	Files   int `json:"files"`
	Lines   int `json:"lines"`
	Changed int `json:"changed"`
}

// StructureNode.Side is "card", "used-by" or "uses". State is "added" or "removed"
// since the point compared with.
type StructureNode struct {
	Path   string         `json:"path"`
	Label  string         `json:"label"`
	Side   string         `json:"side"`
	Folder bool           `json:"folder"`
	Count  StructureCount `json:"count"`
	State  string         `json:"state"`
	Doc    string         `json:"doc"`
	Slot   int            `json:"slot"`
}

// StructureLink is between nodes by index, the first using the second.
type StructureLink struct {
	From  int    `json:"from"`
	To    int    `json:"to"`
	Count int    `json:"count"`
	Main  bool   `json:"main"`
	State string `json:"state"`
}

type StructureWide struct {
	Path   string `json:"path"`
	Label  string `json:"label"`
	Folder bool   `json:"folder"`
	Cards  int    `json:"cards"`
}

// structureView is the last one read, as a level after another is of the same code.
type structureView struct {
	key     string
	view    *structure.View
	graph   *structure.Graph
	lines   map[string]int
	changed map[string]bool
}

var lastStructure = struct {
	sync.Mutex
	structureView
}{}

var (
	errSparsePartial = &gitexec.AppError{Message: "Not available in a sparse partial clone, where reading the code would fetch it."}
	errNoBase        = &gitexec.AppError{Message: "This repository has no default branch to compare with."}
)

// Structure is what's inside a folder or a file, "" for the whole repository, with what
// uses it and what it uses, compared with another point when one is given.
func (s *ExploreService) Structure(repoPath, path string, comparison *ExploreComparison) (StructureInfo, error) {
	ctx := context.Background()
	info := StructureInfo{Crumbs: []StructureCrumb{}, Nodes: []StructureNode{}, Rest: []StructureNode{}, Links: []StructureLink{}}
	if noCommits(ctx, repoPath) {
		return info, nil
	}
	partial := git.PartialClone(ctx, repoPath)
	if partial && git.SparseCheckout(ctx, repoPath) {
		return info, errSparsePartial
	}
	if partial && comparison != nil && comparison.Kind != "working" {
		info.Partial, comparison = true, nil
	}
	lastStructure.Lock()
	defer lastStructure.Unlock()
	read, target, err := readStructure(ctx, repoPath, comparison, partial)
	if err != nil {
		return info, err
	}
	lastStructure.structureView = read
	info.Compared, info.Target = comparison != nil, target
	info.Languages, info.Unread = read.graph.Languages, read.graph.Unread
	info.fill(read, read.view.Level(path))
	return info, nil
}

func readStructure(ctx context.Context, repoPath string, comparison *ExploreComparison, partial bool) (read structureView, target string, err error) {
	commit, err := git.ResolveCommit(ctx, repoPath, "HEAD")
	if err != nil {
		return read, "", err
	}
	tree, then := commit, ""
	// Its objects are only needed until it's read.
	var env []string
	if comparison != nil {
		compared, branchTarget, err := comparedRange(ctx, repoPath, *comparison)
		if err != nil {
			return read, "", err
		}
		if comparison.Kind == "branch" && branchTarget == "" {
			return read, "", errNoBase
		}
		tree, commit, then, target = compared.Head, compared.Head, compared.Base, branchTarget
		if compared.Uncommitted {
			var discard func()
			env, discard = git.ScratchWrites(ctx, repoPath)
			defer discard()
			if tree, err = board.Tree(ctx, repoPath, env); err != nil {
				return read, "", err
			}
		}
	}
	key := repoPath + "\x00" + tree + "\x00" + then
	if lastStructure.key == key {
		return lastStructure.structureView, target, nil
	}
	now, err := structure.Build(ctx, repoPath, env, tree)
	if err != nil {
		return read, "", err
	}
	read = structureView{key: key, graph: now, lines: map[string]int{}, changed: map[string]bool{}}
	var before *structure.Graph
	if then != "" {
		if before, err = structure.Build(ctx, repoPath, nil, then); err != nil {
			return read, "", err
		}
		changes, err := git.TreeChanges(ctx, repoPath, env, then, tree)
		if err != nil {
			return read, "", err
		}
		for _, change := range changes {
			read.changed[change.Path] = true
		}
	}
	read.view = structure.NewView(now, before)
	if partial {
		read.lines = now.Lines
		return read, target, nil
	}
	counts, err := git.LineCounts(ctx, repoPath, commit)
	if err != nil {
		return read, "", err
	}
	for _, f := range counts {
		read.lines[f.Path] = f.Lines
	}
	return read, target, nil
}

func (read structureView) count(files []string) StructureCount {
	count := StructureCount{Files: len(files)}
	for _, f := range files {
		count.Lines += read.lines[f]
		if read.changed[f] {
			count.Changed++
		}
	}
	return count
}

func (info *StructureInfo) fill(read structureView, level structure.Level) {
	node := func(n structure.Node) StructureNode {
		return StructureNode{
			Path: n.Path, Label: n.Label, Side: n.Side, Folder: n.Folder, Count: read.count(n.Files),
			State: n.State, Doc: n.Doc, Slot: n.Slot,
		}
	}
	wide := func(w structure.Wide) StructureWide { return StructureWide(w) }
	info.Path, info.Doc, info.Folder = level.Path, level.Doc, level.Folder
	info.Count = read.count(level.Files)
	info.Crumbs = mapSlice(level.Crumbs, func(c structure.Crumb) StructureCrumb { return StructureCrumb(c) })
	info.Nodes = mapSlice(level.Nodes, node)
	info.Rest = mapSlice(level.Rest, node)
	info.Links = mapSlice(level.Links, func(l structure.Link) StructureLink { return StructureLink(l) })
	info.Cards = level.Cards
	info.WidelyUsedBy, info.WidelyUses = mapSlice(level.WidelyUsedBy, wide), mapSlice(level.WidelyUses, wide)
	info.MoreUsedBy, info.MoreUses = level.MoreUsedBy, level.MoreUses
}
