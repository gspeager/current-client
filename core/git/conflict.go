package git

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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
	// The message git prepared for the commit that finishes the operation.
	Message string
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
	return ConflictState{Operation: op, ConflictedPaths: paths, Message: preparedMessage(dir, op)}, nil
}

// preparedMessage is what git would open in the editor on continue: MERGE_MSG
// for a merge, cherry-pick or revert, the rebase's own file for a rebase.
func preparedMessage(dir string, op ConflictOperation) string {
	var path string
	switch op {
	case ConflictMerge, ConflictCherryPick, ConflictRevert:
		path = filepath.Join(dir, "MERGE_MSG")
	case ConflictRebase:
		path = filepath.Join(dir, "rebase-merge", "message")
	default:
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(stripCommentLines(string(data)))
}

func stripCommentLines(message string) string {
	var kept []string
	for _, line := range strings.Split(message, "\n") {
		if !strings.HasPrefix(line, "#") {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

// ContinueWithMessage finishes the operation with message as the commit's
// message, by replacing what git prepared before continuing.
func ContinueWithMessage(ctx context.Context, repoPath string, op ConflictOperation, message string) error {
	if strings.TrimSpace(message) != "" {
		dir, err := GitDir(ctx, repoPath)
		if err != nil {
			return err
		}
		var path string
		switch op {
		case ConflictMerge, ConflictCherryPick, ConflictRevert:
			path = filepath.Join(dir, "MERGE_MSG")
		case ConflictRebase:
			path = filepath.Join(dir, "rebase-merge", "message")
		}
		if path != "" {
			if err := os.WriteFile(path, []byte(message+"\n"), 0o644); err != nil {
				return err
			}
		}
	}
	return ContinueConflictOperation(ctx, repoPath, op)
}

// runStoppable runs a command that can stop on conflicts. A stop that leaves
// the operation in progress isn't a failure: the conflict banner finishes it.
func runStoppable(ctx context.Context, cmd gitexec.Command) (gitexec.Result, error) {
	result, err := gitexec.NewExecutor("").Run(ctx, cmd)
	if err != nil {
		return gitexec.Result{}, gitexec.WrapRunError(err)
	}
	if result.ExitCode == 0 {
		return result, nil
	}
	if strings.Contains(result.Stdout, "CONFLICT (") {
		if state, err := DetectConflictState(ctx, cmd.Dir); err == nil && state.Operation != ConflictNone {
			return result, nil
		}
	}
	return gitexec.Result{}, gitexec.WrapResult(result)
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

	var names ConflictSideNames
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
	// Git keeps ours as stage 2 of a conflicted file and theirs as stage 3.
	var stage string
	switch side {
	case Ours:
		stage = "2"
	case Theirs:
		stage = "3"
	default:
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
// working-tree file, or 0 when it has none. It reads as it goes, so a large
// binary file isn't held in memory.
func ConflictMarkerLine(repoPath, path string) (int, error) {
	f, err := os.Open(filepath.Join(repoPath, path))
	if err != nil {
		return 0, err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	marker := []byte("<<<<<<< ")
	line, atLineStart := 1, true
	for {
		chunk, err := r.ReadSlice('\n')
		if atLineStart && bytes.HasPrefix(chunk, marker) {
			return line, nil
		}
		// A line longer than the buffer arrives in several chunks.
		atLineStart = err == nil
		if atLineStart {
			line++
		}
		if errors.Is(err, io.EOF) {
			return 0, nil
		}
		if err != nil && !errors.Is(err, bufio.ErrBufferFull) {
			return 0, err
		}
	}
}

// ContinueConflictOperation sets GIT_EDITOR=true so git accepts its prepared
// message instead of hanging on an editor that can never open.
func ContinueConflictOperation(ctx context.Context, repoPath string, op ConflictOperation) error {
	args, err := conflictOpArgs(op, "--continue")
	if err != nil {
		return err
	}
	_, err = runStoppable(ctx, gitexec.Command{
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
