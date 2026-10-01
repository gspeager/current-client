package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type ReflogEntry struct {
	SHA      string
	Selector string // "HEAD@{0}"
	Action   string // "reset: moving to <sha>"
	Date     time.Time
	Subject  string
}

const reflogFormat = "%H\x1f%gd\x1f%gs\x1f%at\x1f%s"

func Reflog(ctx context.Context, repoPath string, limit int) ([]ReflogEntry, error) {
	args := []string{"log", "-g", "-z", "--pretty=format:" + reflogFormat}
	if limit > 0 {
		args = append(args, "-n", strconv.Itoa(limit))
	}
	args = append(args, "HEAD")
	result, err := runResult(ctx, repoPath, args...)
	if err != nil {
		if hasNoCommits(ctx, repoPath) {
			return nil, nil
		}
		return nil, err
	}
	return parseReflog(result.Stdout)
}

func parseReflog(output string) ([]ReflogEntry, error) {
	var entries []ReflogEntry
	for _, record := range strings.Split(output, "\x00") {
		if record == "" {
			continue
		}
		fields := strings.Split(record, "\x1f")
		if len(fields) != 5 {
			return nil, fmt.Errorf("malformed reflog entry: got %d fields, want 5", len(fields))
		}
		ts, err := strconv.ParseInt(fields[3], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("malformed reflog timestamp %q: %w", fields[3], err)
		}
		entries = append(entries, ReflogEntry{
			SHA:      fields[0],
			Selector: fields[1],
			Action:   fields[2],
			Date:     time.Unix(ts, 0),
			Subject:  fields[4],
		})
	}
	return entries, nil
}
