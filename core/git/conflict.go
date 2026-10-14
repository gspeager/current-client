package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

type ConflictOperation string

const (
	ConflictNone       ConflictOperation = ""
	ConflictMerge      ConflictOperation = "merge"
	ConflictCherryPick ConflictOperation = "cherry-pick"
	ConflictRevert     ConflictOperation = "revert"
	ConflictRebase     ConflictOperation = "rebase"
	ConflictAm         ConflictOperation = "am"
)

type ConflictState struct {
	Operation       ConflictOperation
	ConflictedPaths []string
}

// DetectConflictState reports an in-progress operation even once every
// conflict is resolved, since it still needs continuing or aborting.
func DetectConflictState(ctx context.Context, repoPath string) (ConflictState, error) {
	dir, err := GitDir(ctx, repoPath)
	if err != nil {
		return ConflictState{}, err
	}

	op := conflictOperation(dir)
	if op == ConflictNone {
		return ConflictState{}, nil
	}

	statuses, err := GetStatus(ctx, repoPath)
	if err != nil {
		return ConflictState{}, err
	}
	var paths []string
	for _, s := range statuses {
		if s.Conflicted {
			paths = append(paths, s.Path)
		}
	}
	return ConflictState{Operation: op, ConflictedPaths: paths}, nil
}

func conflictOperation(dir string) ConflictOperation {
	switch {
	case fileExists(filepath.Join(dir, "MERGE_HEAD")):
		return ConflictMerge
	case fileExists(filepath.Join(dir, "CHERRY_PICK_HEAD")):
		return ConflictCherryPick
	case fileExists(filepath.Join(dir, "REVERT_HEAD")):
		return ConflictRevert
	case dirExists(filepath.Join(dir, "rebase-merge")):
		return ConflictRebase
	case dirExists(filepath.Join(dir, "rebase-apply")):
		// Shared by `git am` and apply-backend rebases; only am leaves an
		// "applying" marker, and `rebase --continue` refuses to resume an am.
		if fileExists(filepath.Join(dir, "rebase-apply", "applying")) {
			return ConflictAm
		}
		return ConflictRebase
	}
	return ConflictNone
}

type ConflictSide string

const (
	Ours   ConflictSide = "ours"
	Theirs ConflictSide = "theirs"
)

// ConflictSideNames name the two sides as people know them: branches where
// there are some, otherwise a short SHA or what's being applied.
type ConflictSideNames struct {
	Ours   string
	Theirs string
}

// ConflictSides follows Git's meaning of ours and theirs, which a rebase
// swaps: ours is the branch being rebased onto, theirs the one being rebased.
func ConflictSides(ctx context.Context, repoPath string) (ConflictSideNames, error) {
	dir, err := GitDir(ctx, repoPath)
	if err != nil {
		return ConflictSideNames{}, err
	}
	op := conflictOperation(dir)
	if op == ConflictRebase {
		rebaseDir := filepath.Join(dir, "rebase-merge")
		if !dirExists(rebaseDir) {
			rebaseDir = filepath.Join(dir, "rebase-apply")
		}
		theirs := strings.TrimPrefix(readLine(filepath.Join(rebaseDir, "head-name")), "refs/heads/")
		if theirs == "" || theirs == "detached HEAD" {
			theirs = commitName(ctx, repoPath, readLine(filepath.Join(rebaseDir, "orig-head")))
		}
		return ConflictSideNames{Ours: commitName(ctx, repoPath, readLine(filepath.Join(rebaseDir, "onto"))), Theirs: theirs}, nil
	}

	names := ConflictSideNames{Ours: "HEAD"}
	if status, err := CurrentBranchStatus(ctx, repoPath); err == nil && status.DetachedAt == "" {
		names.Ours = status.Current
	} else {
		names.Ours = commitName(ctx, repoPath, "HEAD")
	}
	switch op {
	case ConflictMerge:
		names.Theirs = commitName(ctx, repoPath, "MERGE_HEAD")
	case ConflictCherryPick:
		names.Theirs = shortSHA(ctx, repoPath, "CHERRY_PICK_HEAD")
	case ConflictRevert:
		names.Theirs = "the revert"
	case ConflictAm:
		names.Theirs = "the patch"
	}
	return names, nil
}

// commitName is a branch at rev, local first, or its short SHA.
func commitName(ctx context.Context, repoPath, rev string) string {
	result, err := runResult(ctx, repoPath, "for-each-ref", "--points-at="+rev, "--format=%(refname)%09%(refname:short)", "refs/heads", "refs/remotes")
	if err == nil {
		for _, line := range strings.Split(result.Stdout, "\n") {
			// origin/HEAD shortens to just "origin".
			if ref, name, ok := strings.Cut(line, "\t"); ok && !strings.HasSuffix(ref, "/HEAD") {
				return name
			}
		}
	}
	return shortSHA(ctx, repoPath, rev)
}

func shortSHA(ctx context.Context, repoPath, rev string) string {
	result, err := runResult(ctx, repoPath, "rev-parse", "--short", rev)
	if err != nil {
		return rev
	}
	return strings.TrimSpace(result.Stdout)
}

func readLine(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	line, _, _ := strings.Cut(string(data), "\n")
	return strings.TrimSpace(line)
}

// KeepConflictSide resolves a conflicted file with one side's whole version
// and stages it. If that side deleted the file, it's deleted.
func KeepConflictSide(ctx context.Context, repoPath, path string, side ConflictSide) error {
	stage := map[ConflictSide]string{Ours: "2", Theirs: "3"}[side]
	if stage == "" {
		return fmt.Errorf("unknown side %q", side)
	}
	stages, err := runResult(ctx, repoPath, "ls-files", "-u", "-z", "--", path)
	if err != nil {
		return err
	}
	kept := false
	for _, entry := range strings.Split(stages.Stdout, "\x00") {
		// "<mode> <sha> <stage>\t<path>"
		if fields := strings.Fields(strings.SplitN(entry, "\t", 2)[0]); len(fields) == 3 && fields[2] == stage {
			kept = true
		}
	}
	if !kept {
		_, err := runResult(ctx, repoPath, "rm", "--quiet", "--", path)
		return err
	}
	if _, err := runResult(ctx, repoPath, "checkout", "--"+string(side), "--", path); err != nil {
		return err
	}
	_, err = runResult(ctx, repoPath, "add", "--", path)
	return err
}

// ConflictMarkerLine is the 1-based line of the first "<<<<<<< " marker in a
// working-tree file, or 0 when it has none.
func ConflictMarkerLine(repoPath, path string) (int, error) {
	data, err := os.ReadFile(filepath.Join(repoPath, path))
	if err != nil {
		return 0, err
	}
	for i, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "<<<<<<< ") {
			return i + 1, nil
		}
	}
	return 0, nil
}

// ContinueConflictOperation sets GIT_EDITOR=true so git accepts its prepared
// message instead of hanging on an editor that can never open.
func ContinueConflictOperation(ctx context.Context, repoPath string, op ConflictOperation) error {
	args, err := conflictOpArgs(op, "--continue")
	if err != nil {
		return err
	}
	_, err = gitexec.NewExecutor("").RunChecked(ctx, gitexec.Command{
		Dir:  repoPath,
		Args: args,
		Env:  []string{"GIT_EDITOR=true"},
	})
	return err
}

func AbortConflictOperation(ctx context.Context, repoPath string, op ConflictOperation) error {
	args, err := conflictOpArgs(op, "--abort")
	if err != nil {
		return err
	}
	_, err = runResult(ctx, repoPath, args...)
	return err
}

func conflictOpArgs(op ConflictOperation, flag string) ([]string, error) {
	switch op {
	case ConflictMerge:
		return []string{"merge", flag}, nil
	case ConflictCherryPick:
		return []string{"cherry-pick", flag}, nil
	case ConflictRevert:
		return []string{"revert", flag}, nil
	case ConflictRebase:
		return []string{"rebase", flag}, nil
	case ConflictAm:
		return []string{"am", flag}, nil
	default:
		return nil, fmt.Errorf("no conflict operation in progress")
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
