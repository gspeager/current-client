package git

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

// ErrMergeTreeUnsupported means Git is older than 2.38, which added
// merge-tree --write-tree.
var ErrMergeTreeUnsupported = errors.New("predicting conflicts needs Git 2.38 or newer")

// ErrMergeTreeStdinUnsupported means Git is older than 2.40, which added
// merge-tree --stdin; MergeConflicts still works one merge at a time.
var ErrMergeTreeStdinUnsupported = errors.New("merging in one batch needs Git 2.40 or newer")

// MergeConflicts merges two commits in memory, without touching the working
// tree or index, and returns the files that would conflict.
func MergeConflicts(ctx context.Context, repoPath, ours, theirs string) ([]string, error) {
	result, err := gitexec.NewExecutor("").Run(ctx, gitexec.Command{
		Dir:  repoPath,
		Args: []string{"merge-tree", "--write-tree", "--name-only", "--no-messages", "-z", ours, theirs},
	})
	if err != nil {
		return nil, gitexec.WrapRunError(err)
	}
	switch result.ExitCode {
	case 0:
		return nil, nil
	case 1:
		// "<tree>\0<file>\0<file>\0"
		fields := strings.Split(strings.TrimSuffix(result.Stdout, "\x00"), "\x00")
		return fields[1:], nil
	case 129:
		return nil, ErrMergeTreeUnsupported
	}
	return nil, gitexec.WrapResult(result)
}

// MergeConflictsMany merges ours with each of theirs in one process, as
// MergeConflicts does, and returns their conflicting files in the same order.
func MergeConflictsMany(ctx context.Context, repoPath, ours string, theirs []string) ([][]string, error) {
	var input strings.Builder
	for _, tip := range theirs {
		input.WriteString(ours + " " + tip + "\n")
	}
	result, err := gitexec.NewExecutor("").Run(ctx, gitexec.Command{
		Dir:   repoPath,
		Args:  []string{"merge-tree", "--write-tree", "--stdin", "--name-only", "--no-messages", "-z"},
		Stdin: strings.NewReader(input.String()),
	})
	if err != nil {
		return nil, gitexec.WrapRunError(err)
	}
	if result.ExitCode == 129 {
		return nil, ErrMergeTreeStdinUnsupported
	}
	if result.ExitCode != 0 {
		return nil, gitexec.WrapResult(result)
	}
	return parseMergeTreeStdin(result.Stdout, len(theirs))
}

// Each merge is "<1 clean, 0 conflicted>\0<tree>\0<file>\0…\0".
func parseMergeTreeStdin(output string, merges int) ([][]string, error) {
	fields := strings.Split(output, "\x00")
	conflicts := make([][]string, 0, merges)
	for i := 0; len(conflicts) < merges; {
		if i+1 >= len(fields) {
			return nil, fmt.Errorf("merge-tree answered %d of %d merges", len(conflicts), merges)
		}
		clean := fields[i] == "1"
		i += 2
		var files []string
		for ; i < len(fields) && fields[i] != ""; i++ {
			files = append(files, fields[i])
		}
		i++
		if clean {
			files = nil
		}
		conflicts = append(conflicts, files)
	}
	return conflicts, nil
}
