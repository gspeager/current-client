// Package board reads every worktree of a repository side by side: what each has
// uncommitted, and which of them would conflict.
package board

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
	"github.com/gspeager/current-client/core/overlap"
	"github.com/gspeager/current-client/core/snapshot"
)

const (
	// 190 pairs, and a status each time the board is read.
	mostColumns = 20
	parallel    = 4
)

// Column is a worktree. Problem is why the rest couldn't be read.
type Column struct {
	git.Worktree
	// What differs from HEAD, untracked files included.
	Files []git.TreeChange
	// How many untracked files are too large to count.
	Skipped int
	// Against the board's base.
	Ahead   int
	Behind  int
	Problem string
}

type Board struct {
	// What the columns are ahead of and behind, or "".
	Base    string
	Columns []Column
	// Pairs of columns whose work, uncommitted included, would conflict.
	Collisions []overlap.GridConflict
	// The pairs there was no time for. Reading again goes on with them.
	Unchecked [][2]int
	// Why collisions can't be predicted, if they can't: git.ErrMergeTreeUnsupported
	// or overlap.ErrPartialClone.
	Unpredicted error
	// How many worktrees there was no room for.
	More int
}

// Reader reads a board again and again, doing over only what changed. The zero
// value is ready to use.
type Reader struct {
	mu     sync.Mutex
	repo   string
	stamp  string
	board  Board
	states map[string]state
}

// state is a worktree as it was when its column was read.
type state struct {
	stamp    string
	head     string
	statuses []git.Status
	err      error

	baseTip string
	tree    string
	column  Column
}

// Read costs a status for each worktree when nothing changed. Nothing it reads
// takes a lock, so it's never in the way of a command in a worktree.
func (r *Reader) Read(ctx context.Context, repoPath string) (Board, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.repo != repoPath {
		r.repo, r.stamp, r.board, r.states = repoPath, "", Board{}, nil
	}
	worktrees, err := git.ListWorktrees(ctx, repoPath)
	if err != nil {
		return Board{}, err
	}
	_, base, err := git.DefaultBase(ctx, repoPath)
	if err != nil {
		return Board{}, err
	}
	baseTip := ""
	if base != "" {
		if baseTip, err = git.ResolveCommit(ctx, repoPath, base); err != nil {
			return Board{}, err
		}
	}
	shown, more := pick(worktrees)
	states := make([]state, len(shown))
	inParallel(len(shown), func(i int) { states[i] = readState(ctx, shown[i]) })

	stamp := base + "\x00" + baseTip + "\x00" + fmt.Sprint(more)
	for i, s := range states {
		stamp += fmt.Sprintf("\x00%v\x00%s\x00%v", shown[i], s.stamp, s.err)
	}
	// Pairs there was no time for are worth another go.
	if stamp == r.stamp && len(r.board.Unchecked) == 0 {
		return r.board, nil
	}

	env, discard := git.ScratchObjects(ctx, repoPath)
	defer discard()
	board := Board{Base: base, More: more, Columns: make([]Column, len(shown))}
	tips := make([]string, len(shown))
	inParallel(len(shown), func(i int) {
		states[i].column, tips[i] = r.column(ctx, env, shown[i], &states[i], baseTip)
		board.Columns[i] = states[i].column
	})
	if err := board.predict(ctx, repoPath, env, tips); err != nil {
		return Board{}, err
	}
	r.stamp, r.board, r.states = stamp, board, map[string]state{}
	for i, w := range shown {
		r.states[w.Path] = states[i]
	}
	return board, nil
}

// pick keeps the current worktree when there are more than fit.
func pick(worktrees []git.Worktree) (shown []git.Worktree, more int) {
	if len(worktrees) <= mostColumns {
		return worktrees, 0
	}
	shown = append(shown, worktrees[:mostColumns]...)
	for _, w := range worktrees[mostColumns:] {
		if w.Current {
			shown[mostColumns-1] = w
		}
	}
	return shown, len(worktrees) - mostColumns
}

func inParallel(count int, do func(i int)) {
	slots := make(chan struct{}, parallel)
	var wg sync.WaitGroup
	for i := range count {
		slots <- struct{}{}
		wg.Go(func() {
			defer func() { <-slots }()
			do(i)
		})
	}
	wg.Wait()
}

// readState tells one edit of a changed file from the next by its size and time.
func readState(ctx context.Context, w git.Worktree) state {
	if w.Missing {
		return state{}
	}
	result, err := gitexec.RunIn(ctx, w.Path,
		"--no-optional-locks", "status", "--porcelain=v2", "--branch", "-z", "--untracked-files=all")
	if err != nil {
		return state{err: err}
	}
	statuses, err := git.ParseStatus(result.Stdout)
	if err != nil {
		return state{err: err}
	}
	s := state{statuses: statuses}
	h := fnv.New64a()
	for _, record := range strings.Split(result.Stdout, "\x00") {
		// The upstream's counts aren't shown.
		if strings.HasPrefix(record, "# branch.ab") {
			continue
		}
		if head, ok := strings.CutPrefix(record, "# branch.oid "); ok && head != "(initial)" {
			s.head = head
		}
		fmt.Fprintf(h, "%s\x00", record)
	}
	for _, status := range statuses {
		if info, err := os.Lstat(filepath.Join(w.Path, status.Path)); err == nil {
			fmt.Fprintf(h, "%d\x00%d\x00%s\x00", info.Size(), info.ModTime().UnixNano(), info.Mode())
		}
	}
	s.stamp = fmt.Sprintf("%x", h.Sum64())
	return s
}

// column is the worktree's column, and a commit that holds its work, or "" when it
// has none to merge.
func (r *Reader) column(ctx context.Context, env []string, w git.Worktree, s *state, baseTip string) (column Column, tip string) {
	column = Column{Worktree: w}
	if w.Missing {
		return column, ""
	}
	fail := func(err error) (Column, string) {
		column.Problem = err.Error()
		return column, ""
	}
	if s.err != nil {
		return fail(s.err)
	}
	last, known := r.states[w.Path]
	s.baseTip = baseTip
	if known && last.head == s.head && last.baseTip == baseTip {
		column.Ahead, column.Behind = last.column.Ahead, last.column.Behind
	} else if s.head != "" && baseTip != "" {
		var err error
		if column.Ahead, column.Behind, err = git.AheadBehind(ctx, w.Path, s.head, baseTip); err != nil {
			return fail(err)
		}
	}
	if len(s.statuses) == 0 {
		return column, s.head
	}
	tree, skipped, err := snapshot.WriteTree(ctx, w.Path, env, s.statuses)
	if err != nil {
		return fail(err)
	}
	s.tree, column.Skipped = tree, skipped
	if known && last.tree == tree && last.head == s.head {
		column.Files = last.column.Files
	} else if column.Files, err = changes(ctx, w.Path, env, s.head, tree); err != nil {
		return fail(err)
	}
	if s.head == "" {
		return column, ""
	}
	if tip, err = commit(ctx, w.Path, env, tree, s.head); err != nil {
		return fail(err)
	}
	return column, tip
}

func changes(ctx context.Context, path string, env []string, head, tree string) ([]git.TreeChange, error) {
	if head == "" {
		empty, err := run(ctx, path, env, "hash-object", "-t", "tree", "-w", "--stdin")
		if err != nil {
			return nil, err
		}
		head = empty
	}
	return git.TreeChanges(ctx, path, env, head, tree)
}

// The same work is the same commit, so what it conflicts with is known already.
var identity = []string{
	"GIT_AUTHOR_NAME=Current Client", "GIT_AUTHOR_EMAIL=board@localhost", "GIT_AUTHOR_DATE=1700000000 +0000",
	"GIT_COMMITTER_NAME=Current Client", "GIT_COMMITTER_EMAIL=board@localhost", "GIT_COMMITTER_DATE=1700000000 +0000",
}

func commit(ctx context.Context, path string, env []string, tree, head string) (string, error) {
	return run(ctx, path, append(append([]string{}, env...), identity...), "commit-tree", tree, "-p", head, "-m", "work")
}

func run(ctx context.Context, path string, env []string, args ...string) (string, error) {
	result, err := gitexec.NewExecutor("").RunChecked(ctx, gitexec.Command{Dir: path, Env: env, Args: args})
	return strings.TrimSpace(result.Stdout), err
}

func (b *Board) predict(ctx context.Context, repoPath string, env, tips []string) error {
	if git.PartialClone(ctx, repoPath) {
		b.Unpredicted = overlap.ErrPartialClone
		return nil
	}
	var at []int
	var commits []string
	for i, tip := range tips {
		if tip != "" {
			at = append(at, i)
			commits = append(commits, tip)
		}
	}
	conflicts, unchecked, err := overlap.Between(ctx, repoPath, env, commits)
	if errors.Is(err, git.ErrMergeTreeUnsupported) {
		b.Unpredicted = git.ErrMergeTreeUnsupported
		return nil
	}
	if err != nil {
		return err
	}
	for _, c := range conflicts {
		b.Collisions = append(b.Collisions, overlap.GridConflict{A: at[c.A], B: at[c.B], Files: c.Files})
	}
	for _, pair := range unchecked {
		b.Unchecked = append(b.Unchecked, [2]int{at[pair[0]], at[pair[1]]})
	}
	return nil
}
