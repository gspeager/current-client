package git

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
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
	args := []string{"stash", "push"}
	if includeUntracked {
		args = append(args, "-u")
	}
	if message != "" {
		args = append(args, "-m", message)
	}
	_, err := runResult(ctx, repoPath, args...)
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
