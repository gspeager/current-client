package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gspeager/current-client/core/gitexec"
)

type CommitLines struct {
	SHA     string
	Time    time.Time
	Subject string
	// A renamed file is under its new name, counting only what changed.
	Files []NumstatEntry
}

// RangeCommitLines reads the commits from..to on to's first parents, newest
// first, with each file's lines. A merge counts what it brought, against its first
// parent, and the merged branch's own commits aren't read. from "" reads back to
// the first commit, and a non-zero since stops at commits made before it. It reads
// at most most, and more says the range goes further back.
func RangeCommitLines(ctx context.Context, repoPath, from, to string, since time.Time, most int) (commits []CommitLines, more bool, err error) {
	rev, err := revRange(ctx, repoPath, from, to)
	if err != nil {
		return nil, false, err
	}
	args := []string{"-c", "log.showRoot=true", "log", "--first-parent", "-m", "-M", "--numstat", "-z",
		"--no-ext-diff", "--no-textconv", "--no-show-signature", "--no-relative",
		"--max-count=" + strconv.Itoa(most+1), "--format=%x01%H%x1f%ct%x1f%s"}
	if !since.IsZero() {
		args = append(args, "--since="+strconv.FormatInt(since.Unix(), 10))
	}
	result, err := gitexec.RunIn(ctx, repoPath, append(args, rev, "--")...)
	if err != nil {
		return nil, false, err
	}
	for block := range strings.SplitSeq(result.Stdout, "\x01") {
		if block == "" {
			continue
		}
		if len(commits) == most {
			return commits, true, nil
		}
		header, stats, _ := strings.Cut(block, "\x00")
		fields := strings.SplitN(header, "\x1f", 3)
		if len(fields) != 3 {
			return nil, false, fmt.Errorf("malformed log entry: %q", header)
		}
		seconds, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return nil, false, err
		}
		files, err := parseNumstat(strings.TrimPrefix(stats, "\n"))
		if err != nil {
			return nil, false, err
		}
		commits = append(commits, CommitLines{SHA: fields[0], Time: time.Unix(seconds, 0), Subject: fields[2], Files: files})
	}
	return commits, false, nil
}
