package app

import (
	"context"
	"errors"

	"github.com/gspeager/current-client/core/board"
	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/overlap"
	"github.com/gspeager/current-client/internal/policy"
)

type WorktreeService struct {
	board board.Reader
}

type WorktreeInfo struct {
	Path    string `json:"path"`
	Branch  string `json:"branch"`
	Head    string `json:"head"`
	Main    bool   `json:"main"`
	Current bool   `json:"current"`
	Locked  bool   `json:"locked"`
	Missing bool   `json:"missing"`
	Bare    bool   `json:"bare"`
}

func (s *WorktreeService) List(repoPath string) ([]WorktreeInfo, error) {
	worktrees, err := git.ListWorktrees(context.Background(), repoPath)
	if err != nil {
		return nil, err
	}
	return mapSlice(worktrees, func(w git.Worktree) WorktreeInfo { return WorktreeInfo(w) }), nil
}

// Add checks out branch in a new worktree at path, creating the branch from
// HEAD first when create is set.
func (s *WorktreeService) Add(repoPath, path, branch string, create bool) error {
	return git.AddWorktree(context.Background(), repoPath, path, branch, create, "")
}

func (s *WorktreeService) Remove(repoPath, path string, force bool) error {
	ctx := context.Background()
	if err := git.RemoveWorktree(ctx, repoPath, path, force); err != nil {
		return err
	}
	forgetMarks(ctx, repoPath)
	return nil
}

// BoardInfo is every worktree side by side. Unpredicted is "", "git" where Git is
// too old to predict collisions, or "partial" in a partial clone, where it would fetch.
type BoardInfo struct {
	Base        string             `json:"base"`
	Columns     []BoardColumn      `json:"columns"`
	Collisions  []GridConflictInfo `json:"collisions"`
	Unchecked   [][2]int           `json:"unchecked"`
	Unpredicted string             `json:"unpredicted"`
	More        int                `json:"more"`
}

type BoardColumn struct {
	Worktree WorktreeInfo `json:"worktree"`
	Changed  int          `json:"changed"`
	// The first of what's changed, with the ones that collide first.
	Files   []BoardFile `json:"files"`
	Added   int         `json:"added"`
	Removed int         `json:"removed"`
	Skipped int         `json:"skipped"`
	Ahead   int         `json:"ahead"`
	Behind  int         `json:"behind"`
	Problem string      `json:"problem"`
	// Whether something is running in its terminal.
	Running bool `json:"running"`
}

type BoardFile struct {
	Path   string `json:"path"`
	Status string `json:"status"`
}

func (s *WorktreeService) Board(repoPath string) (BoardInfo, error) {
	read, err := s.board.Read(context.Background(), repoPath)
	if err != nil {
		return BoardInfo{}, err
	}
	info := BoardInfo{
		Base:       read.Base,
		Collisions: mapSlice(read.Collisions, func(c overlap.GridConflict) GridConflictInfo { return GridConflictInfo(c) }),
		Unchecked:  read.Unchecked,
		More:       read.More,
	}
	colliding := make([]map[string]bool, len(read.Columns))
	for _, c := range read.Collisions {
		for _, i := range []int{c.A, c.B} {
			if colliding[i] == nil {
				colliding[i] = map[string]bool{}
			}
			for _, file := range c.Files {
				colliding[i][file] = true
			}
		}
	}
	for i, c := range read.Columns {
		info.Columns = append(info.Columns, boardColumn(c, colliding[i]))
	}
	switch {
	case errors.Is(read.Unpredicted, git.ErrMergeTreeUnsupported):
		info.Unpredicted = "git"
	case errors.Is(read.Unpredicted, overlap.ErrPartialClone):
		info.Unpredicted = "partial"
	}
	return info, nil
}

// What a column has room for. It's all sent again on every read.
const mostBoardFiles = 100

func boardColumn(c board.Column, colliding map[string]bool) BoardColumn {
	column := BoardColumn{
		Worktree: WorktreeInfo(c.Worktree),
		Skipped:  c.Skipped,
		Ahead:    c.Ahead,
		Behind:   c.Behind,
		Problem:  c.Problem,
		Running:  !policy.LocalCommandsOff() && terminals.AnyBusyWhere(func(path string) bool { return git.SamePath(path, c.Path) }),
		Changed:  len(c.Files),
		Files:    []BoardFile{},
	}
	var rest []BoardFile
	for _, f := range c.Files {
		column.Added += f.Added
		column.Removed += f.Removed
		if file := (BoardFile{Path: f.Path, Status: f.Status[:1]}); colliding[f.Path] {
			column.Files = append(column.Files, file)
		} else {
			rest = append(rest, file)
		}
	}
	if room := mostBoardFiles - len(column.Files); room > 0 {
		column.Files = append(column.Files, rest[:min(room, len(rest))]...)
	}
	return column
}
