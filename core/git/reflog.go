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
	// Date is when the entry was recorded: when the checkout, reset or commit happened, not
	// the date of the commit it moved to.
	Date    time.Time
	Subject string
}

// With --date=unix, %gd is the entry's own time ("HEAD@{1728000000}"); there's no other
// placeholder for it. The numbered selector is rebuilt from the entry's position, since
// git log -g lists the reflog in order from HEAD@{0}.
const reflogFormat = "%H\x1f%gd\x1f%gs\x1f%s"

func Reflog(ctx context.Context, repoPath string, limit int) ([]ReflogEntry, error) {
	args := []string{"log", "-g", "-z", "--date=unix", "--pretty=format:" + reflogFormat}
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
		if len(fields) != 4 {
			return nil, fmt.Errorf("malformed reflog entry: got %d fields, want 4", len(fields))
		}
		dated := fields[1]
		open, end := strings.LastIndex(dated, "@{"), strings.LastIndex(dated, "}")
		if open < 0 || end < open {
			return nil, fmt.Errorf("malformed reflog selector %q", dated)
		}
		ts, err := strconv.ParseInt(dated[open+2:end], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("malformed reflog timestamp %q: %w", dated, err)
		}
		entries = append(entries, ReflogEntry{
			SHA:      fields[0],
			Selector: dated[:open] + "@{" + strconv.Itoa(len(entries)) + "}",
			Action:   fields[2],
			Date:     time.Unix(ts, 0),
			Subject:  fields[3],
		})
	}
	return entries, nil
}
