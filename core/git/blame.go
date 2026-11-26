package git

import (
	"bufio"
	"context"
	"fmt"
	"github.com/gspeager/current-client/core/gitexec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type BlameLine struct {
	SHA         string
	AuthorName  string
	AuthorEmail string
	Date        time.Time
	Summary     string
	LineNo      int
	Content     string
	// The commit before SHA that had the file, and its path there; empty for
	// the commit that added it.
	Previous     string
	PreviousPath string
}

func Blame(ctx context.Context, repoPath, path string) ([]BlameLine, error) {
	return BlameAt(ctx, repoPath, path, "")
}

// BlameAt blames path as of rev, or the working tree when rev is empty.
func BlameAt(ctx context.Context, repoPath, path, rev string) ([]BlameLine, error) {
	if strings.HasPrefix(rev, "-") {
		return nil, &gitexec.AppError{Message: "That isn't a commit."}
	}
	args := []string{"blame", "--porcelain", "-w"}
	if rev != "" {
		args = append(args, rev)
	}
	result, err := runResult(ctx, repoPath, append(args, "--", path)...)
	if err != nil {
		return nil, err
	}
	return parseBlame(result.Stdout)
}

var blameHeaderPattern = regexp.MustCompile(`^([0-9a-f]{40}) \d+ (\d+)`)

// parseBlame caches metadata by SHA because porcelain output only includes a
// commit's author/summary lines the first time that commit appears.
func parseBlame(output string) ([]BlameLine, error) {
	type commitMeta struct {
		authorName  string
		authorEmail string
		authorTime  int64
		summary     string
		previous    string
		prevPath    string
	}
	metaBySHA := make(map[string]*commitMeta)

	var lines []BlameLine
	var curSHA string
	var curLineNo int

	scanner := bufio.NewScanner(strings.NewReader(output))
	scanner.Buffer(make([]byte, 0, 64*1024), 10<<20)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "\t") {
			meta := metaBySHA[curSHA]
			if meta == nil {
				return nil, fmt.Errorf("blame content line before any header for %q", curSHA)
			}
			lines = append(lines, BlameLine{
				SHA:          curSHA,
				AuthorName:   meta.authorName,
				AuthorEmail:  meta.authorEmail,
				Date:         time.Unix(meta.authorTime, 0),
				Summary:      meta.summary,
				LineNo:       curLineNo,
				Content:      line[1:],
				Previous:     meta.previous,
				PreviousPath: meta.prevPath,
			})
			continue
		}
		if m := blameHeaderPattern.FindStringSubmatch(line); m != nil {
			curSHA = m[1]
			lineNo, err := strconv.Atoi(m[2])
			if err != nil {
				return nil, fmt.Errorf("malformed blame line number %q: %w", m[2], err)
			}
			curLineNo = lineNo
			if _, ok := metaBySHA[curSHA]; !ok {
				metaBySHA[curSHA] = &commitMeta{}
			}
			continue
		}
		meta := metaBySHA[curSHA]
		if meta == nil {
			continue
		}
		switch {
		case strings.HasPrefix(line, "author "):
			meta.authorName = strings.TrimPrefix(line, "author ")
		case strings.HasPrefix(line, "author-mail "):
			meta.authorEmail = strings.Trim(strings.TrimPrefix(line, "author-mail "), "<>")
		case strings.HasPrefix(line, "author-time "):
			ts, err := strconv.ParseInt(strings.TrimPrefix(line, "author-time "), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("malformed blame author-time %q: %w", line, err)
			}
			meta.authorTime = ts
		case strings.HasPrefix(line, "summary "):
			meta.summary = strings.TrimPrefix(line, "summary ")
		case strings.HasPrefix(line, "previous "):
			meta.previous, meta.prevPath, _ = strings.Cut(strings.TrimPrefix(line, "previous "), " ")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}
