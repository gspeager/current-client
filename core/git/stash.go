package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gspeager/current-client/core/gitexec"
)

type Stash struct {
	Index   int
	Message string
	Date    time.Time
}

var stashIndexPattern = regexp.MustCompile(`^stash@\{(\d+)\}$`)

func ListStashes(ctx context.Context, repoPath string) ([]Stash, error) {
	result, err := runResult(ctx, repoPath, "stash", "list", "--format=%gd%x09%s%x09%at")
	if err != nil {
		return nil, err
	}
	return parseStashList(result.Stdout)
}

func parseStashList(output string) ([]Stash, error) {
	var stashes []Stash
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "\t", 3)
		if len(fields) != 3 {
			continue
		}
		m := stashIndexPattern.FindStringSubmatch(fields[0])
		if m == nil {
			continue
		}
		index, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		ts, err := strconv.ParseInt(fields[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("malformed stash timestamp %q: %w", fields[2], err)
		}
		stashes = append(stashes, Stash{Index: index, Message: fields[1], Date: time.Unix(ts, 0)})
	}
	return stashes, nil
}

func StashSave(ctx context.Context, repoPath, message string, includeUntracked bool) error {
	return StashPush(ctx, repoPath, StashOptions{Message: message, IncludeUntracked: includeUntracked})
}

type StashOptions struct {
	Message          string
	IncludeUntracked bool
	// KeepIndex leaves staged changes in place, stashing only the rest.
	KeepIndex bool
	// Paths limits the stash to these files; empty means every change.
	Paths []string
}

func StashPush(ctx context.Context, repoPath string, opts StashOptions) error {
	args := []string{"stash", "push"}
	if opts.IncludeUntracked {
		args = append(args, "-u")
	}
	if opts.KeepIndex {
		args = append(args, "--keep-index")
	}
	if opts.Message != "" {
		args = append(args, "-m", opts.Message)
	}
	if len(opts.Paths) == 0 {
		_, err := runResult(ctx, repoPath, args...)
		return err
	}
	return stashPaths(ctx, repoPath, append(append(args, "--"), opts.Paths...), opts)
}

// `git stash push -- <paths>` still saves the whole index, so files staged but
// not chosen end up in the stash too, and can stop it popping later. Stashing
// against a temporary index holding only HEAD plus the chosen paths' staged
// changes leaves them out; the real index then drops those paths' staged
// changes, as the stash did in the working tree. If the temporary index can't
// be built (no commits yet, unresolved conflicts), the plain stash runs and
// reports what's wrong.
func stashPaths(ctx context.Context, repoPath string, args []string, opts StashOptions) error {
	tmp, err := os.MkdirTemp("", "current-client-stash-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	withIndex := func(args ...string) error {
		_, err := gitexec.NewExecutor("").RunChecked(ctx, gitexec.Command{
			Dir:  repoPath,
			Args: args,
			Env:  []string{"GIT_INDEX_FILE=" + filepath.Join(tmp, "index")},
		})
		return err
	}

	staged, err := runResult(ctx, repoPath, "write-tree")
	if err != nil || withIndex("read-tree", "HEAD") != nil ||
		withIndex(append([]string{"reset", "-q", strings.TrimSpace(staged.Stdout), "--"}, opts.Paths...)...) != nil {
		_, err := runResult(ctx, repoPath, args...)
		return err
	}
	if err := withIndex(args...); err != nil {
		return err
	}
	if opts.KeepIndex {
		return nil
	}
	_, err = runResult(ctx, repoPath, append([]string{"reset", "-q", "--"}, opts.Paths...)...)
	return err
}

func StashApply(ctx context.Context, repoPath string, index int) error {
	_, err := runResult(ctx, repoPath, "stash", "apply", stashRef(index))
	return err
}

func StashPop(ctx context.Context, repoPath string, index int) error {
	_, err := runResult(ctx, repoPath, "stash", "pop", stashRef(index))
	return err
}

func StashDrop(ctx context.Context, repoPath string, index int) error {
	_, err := runResult(ctx, repoPath, "stash", "drop", stashRef(index))
	return err
}

func stashRef(index int) string {
	return fmt.Sprintf("stash@{%d}", index)
}

// StashChangedFiles lists a stash's tracked changes against the commit it was
// made on, then the untracked files it saved with status "?".
func StashChangedFiles(ctx context.Context, repoPath string, index int) ([]ChangedFile, error) {
	ref := stashRef(index)
	files, err := RefRangeChangedFiles(ctx, repoPath, ref+"^1", ref)
	if err != nil {
		return nil, err
	}
	untrackedRef, err := stashUntrackedRef(ctx, repoPath, index)
	if err != nil || untrackedRef == "" {
		return files, err
	}
	untracked, err := ChangedFiles(ctx, repoPath, untrackedRef)
	if err != nil {
		return nil, err
	}
	for _, f := range untracked {
		f.Status = "?"
		files = append(files, f)
	}
	return files, nil
}

func StashNumstat(ctx context.Context, repoPath string, index int) ([]NumstatEntry, error) {
	ref := stashRef(index)
	stats, err := RefRangeNumstat(ctx, repoPath, ref+"^1", ref)
	if err != nil {
		return nil, err
	}
	untrackedRef, err := stashUntrackedRef(ctx, repoPath, index)
	if err != nil || untrackedRef == "" {
		return stats, err
	}
	untracked, err := CommitNumstat(ctx, repoPath, untrackedRef)
	if err != nil {
		return nil, err
	}
	return append(stats, untracked...), nil
}

// stashUntrackedRef names the parentless commit holding the untracked files a
// stash saved (`stash push -u`), or "" when it saved none.
func stashUntrackedRef(ctx context.Context, repoPath string, index int) (string, error) {
	ref := stashRef(index)
	result, err := runResult(ctx, repoPath, "rev-list", "--parents", "-n", "1", ref)
	if err != nil {
		return "", err
	}
	// The stash commit itself, then its parents: HEAD, the index, and the untracked files.
	if len(strings.Fields(result.Stdout)) < 4 {
		return "", nil
	}
	return ref + "^3", nil
}
