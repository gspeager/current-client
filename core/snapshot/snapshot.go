// Package snapshot saves work a destructive action would lose, under
// refs/current-client/snapshots, which is never pushed, shown as a branch or
// counted as history. A tree snapshot holds the working tree's tracked and
// untracked changes. A tip snapshot holds a branch's commits, for an action that
// loses commits rather than edits.
package snapshot

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
)

const (
	refPrefix = "refs/current-client/snapshots/"

	KindTree = "tree"
	KindTip  = "tip"

	DefaultLimit = 50

	// An untracked file this large is left out, so a build output or a data file
	// doesn't land in the object store on every checkout.
	largestUntracked = 10 << 20
	// A folder of unignored output shouldn't either, and listing what's left out
	// is limited by the command line's length.
	mostUntrackedBytes = 100 << 20
	mostLeftOut        = 500
)

// ErrNotFound means the snapshot has been pruned.
var ErrNotFound = errors.New("this snapshot is gone")

type Snapshot struct {
	ID     string
	Kind   string
	SHA    string
	Parent string
	At     time.Time
	Action string
	Branch string
	// Skipped counts untracked files too large to keep.
	Skipped int
}

func run(ctx context.Context, repoPath string, env []string, args ...string) (string, error) {
	result, err := gitexec.NewExecutor("").RunChecked(ctx, gitexec.Command{Dir: repoPath, Args: args, Env: env})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

var identity = []string{
	"GIT_AUTHOR_NAME=Current Client", "GIT_AUTHOR_EMAIL=snapshot@localhost",
	"GIT_COMMITTER_NAME=Current Client", "GIT_COMMITTER_EMAIL=snapshot@localhost",
}

func currentBranch(ctx context.Context, repoPath string) string {
	branch, _ := run(ctx, repoPath, nil, "symbolic-ref", "--short", "-q", "HEAD")
	return branch
}

// Take saves the working tree before action, unless there's nothing to save or
// it's what the newest snapshot already holds. It reads through a copy of the
// index, so the real one is untouched and unchanged files aren't hashed again.
func Take(ctx context.Context, repoPath, action string, limit int) (bool, error) {
	statuses, err := git.GetStatus(ctx, repoPath)
	if err != nil || len(statuses) == 0 {
		return false, err
	}
	dir, err := os.MkdirTemp("", "current-client-snapshot")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(dir)
	index := filepath.Join(dir, "index")
	if real, err := run(ctx, repoPath, nil, "rev-parse", "--git-path", "index"); err == nil {
		if !filepath.IsAbs(real) {
			real = filepath.Join(repoPath, real)
		}
		if data, err := os.ReadFile(real); err == nil {
			if err := os.WriteFile(index, data, 0o600); err != nil {
				return false, err
			}
		}
	}
	env := []string{"GIT_INDEX_FILE=" + index}

	args := []string{"add", "-A", "--", "."}
	skipped := 0
	var kept int64
	for _, status := range statuses {
		if status.WorktreeStatus != '?' {
			continue
		}
		info, err := os.Stat(filepath.Join(repoPath, status.Path))
		if err != nil {
			continue
		}
		if info.Size() >= largestUntracked || kept+info.Size() > mostUntrackedBytes {
			args = append(args, ":(exclude,literal)"+status.Path)
			skipped++
			if skipped > mostLeftOut {
				return false, fmt.Errorf("too many large untracked files to snapshot")
			}
			continue
		}
		kept += info.Size()
	}
	if _, err := run(ctx, repoPath, env, args...); err != nil {
		return false, err
	}
	tree, err := run(ctx, repoPath, env, "write-tree")
	if err != nil {
		return false, err
	}
	head, headErr := git.ResolveCommit(ctx, repoPath, "HEAD")
	if headErr == nil {
		if headTree, err := run(ctx, repoPath, nil, "rev-parse", head+"^{tree}"); err == nil && headTree == tree {
			return false, nil
		}
	}
	existing, err := List(ctx, repoPath)
	if err != nil {
		return false, err
	}
	if len(existing) > 0 && existing[0].Kind == KindTree {
		if newest, err := run(ctx, repoPath, nil, "rev-parse", existing[0].SHA+"^{tree}"); err == nil && newest == tree {
			return false, nil
		}
	}
	commit := []string{"commit-tree", tree, "-m", action, "-m", metadata(KindTree, currentBranch(ctx, repoPath), skipped)}
	if headErr == nil {
		commit = append(commit, "-p", head)
	}
	sha, err := run(ctx, repoPath, identity, commit...)
	if err != nil {
		return false, err
	}
	return true, save(ctx, repoPath, sha, limit)
}

// TakeTip keeps branch's commits at tip reachable. Its snapshot adds nothing but a
// commit on top, which holds what the action and branch were.
func TakeTip(ctx context.Context, repoPath, action, branch, tip string, limit int) error {
	tree, err := run(ctx, repoPath, nil, "rev-parse", tip+"^{tree}")
	if err != nil {
		return err
	}
	sha, err := run(ctx, repoPath, identity, "commit-tree", tree, "-p", tip, "-m", action, "-m", metadata(KindTip, branch, 0))
	if err != nil {
		return err
	}
	return save(ctx, repoPath, sha, limit)
}

func metadata(kind, branch string, skipped int) string {
	return fmt.Sprintf("kind: %s\nbranch: %s\nskipped: %d", kind, branch, skipped)
}

func save(ctx context.Context, repoPath, sha string, limit int) error {
	id := strconv.FormatInt(time.Now().UnixNano(), 10)
	if _, err := run(ctx, repoPath, nil, "update-ref", refPrefix+id, sha); err != nil {
		return err
	}
	return Prune(ctx, repoPath, limit)
}

// Prune keeps the newest limit, or DefaultLimit when limit isn't positive.
func Prune(ctx context.Context, repoPath string, limit int) error {
	if limit <= 0 {
		limit = DefaultLimit
	}
	snapshots, err := List(ctx, repoPath)
	if err != nil || len(snapshots) <= limit {
		return err
	}
	return deleteEach(ctx, repoPath, snapshots[limit:])
}

func Delete(ctx context.Context, repoPath, id string) error {
	_, err := run(ctx, repoPath, nil, "update-ref", "-d", refPrefix+id)
	return err
}

func DeleteAll(ctx context.Context, repoPath string) error {
	snapshots, err := List(ctx, repoPath)
	if err != nil {
		return err
	}
	return deleteEach(ctx, repoPath, snapshots)
}

func deleteEach(ctx context.Context, repoPath string, snapshots []Snapshot) error {
	for _, snapshot := range snapshots {
		if err := Delete(ctx, repoPath, snapshot.ID); err != nil {
			return err
		}
	}
	return nil
}

// List returns them newest first.
func List(ctx context.Context, repoPath string) ([]Snapshot, error) {
	output, err := run(ctx, repoPath, nil, "for-each-ref", strings.TrimSuffix(refPrefix, "/"),
		"--format=%(refname)%00%(objectname)%00%(parent)%00%(committerdate:unix)%00%(subject)%00%(body)%01")
	if err != nil {
		return nil, err
	}
	var snapshots []Snapshot
	for _, record := range strings.Split(output, "\x01") {
		fields := strings.Split(strings.TrimSpace(record), "\x00")
		if len(fields) != 6 {
			continue
		}
		seconds, _ := strconv.ParseInt(fields[3], 10, 64)
		snapshot := Snapshot{
			ID:     strings.TrimPrefix(fields[0], refPrefix),
			SHA:    fields[1],
			Parent: fields[2],
			At:     time.Unix(seconds, 0),
			Action: fields[4],
		}
		for _, line := range strings.Split(fields[5], "\n") {
			key, value, _ := strings.Cut(line, ": ")
			switch key {
			case "kind":
				snapshot.Kind = value
			case "branch":
				snapshot.Branch = value
			case "skipped":
				snapshot.Skipped, _ = strconv.Atoi(value)
			}
		}
		snapshots = append(snapshots, snapshot)
	}
	sort.Slice(snapshots, func(i, j int) bool { return newerID(snapshots[i].ID, snapshots[j].ID) })
	return snapshots, nil
}

func newerID(a, b string) bool {
	if len(a) != len(b) {
		return len(a) > len(b)
	}
	return a > b
}

// Restore puts the snapshot's files back in the working tree, or just paths when
// given. It leaves the index alone, and first saves what it's about to overwrite.
func Restore(ctx context.Context, repoPath, id string, paths []string, limit int) error {
	sha, err := git.ResolveCommit(ctx, repoPath, refPrefix+id)
	if err != nil {
		return ErrNotFound
	}
	files, err := git.ChangedFiles(ctx, repoPath, sha)
	if err != nil {
		return err
	}
	if _, err := Take(ctx, repoPath, "restore", limit); err != nil {
		return err
	}
	for _, file := range files {
		if len(paths) > 0 && !slices.Contains(paths, file.Path) {
			continue
		}
		if file.OrigPath != "" {
			if err := remove(repoPath, file.OrigPath); err != nil {
				return err
			}
		}
		if file.Status == "D" {
			if err := remove(repoPath, file.Path); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(filepath.Join(repoPath, file.Path)), 0o755); err != nil {
			return err
		}
		if _, err := run(ctx, repoPath, nil, "restore", "--source="+sha, "--worktree", "--", ":(literal)"+file.Path); err != nil {
			return err
		}
	}
	return nil
}

func remove(repoPath, path string) error {
	err := os.Remove(filepath.Join(repoPath, path))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
