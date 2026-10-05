package git

import (
	"context"
	"strconv"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

type GrepOptions struct {
	Pattern string
	// Rev is the commit-ish to search; "" searches the working tree,
	// including untracked files that aren't ignored.
	Rev        string
	IgnoreCase bool
	WholeWord  bool
	// Regexp treats Pattern as an extended regular expression rather than
	// fixed text.
	Regexp     bool
	MaxMatches int
}

type GrepMatch struct {
	Path string
	Line int
	Text string
}

// Grep searches file contents with git grep, skipping binary files. truncated
// reports that there were more than MaxMatches matches.
func Grep(ctx context.Context, repoPath string, opts GrepOptions) (matches []GrepMatch, truncated bool, err error) {
	args := []string{"grep", "-n", "-I", "-z", "--no-color", "--full-name"}
	if opts.IgnoreCase {
		args = append(args, "-i")
	}
	if opts.WholeWord {
		args = append(args, "-w")
	}
	if opts.Regexp {
		args = append(args, "-E")
	} else {
		args = append(args, "-F")
	}
	args = append(args, "-e", opts.Pattern)
	if opts.Rev == "" {
		args = append(args, "--untracked")
	} else {
		args = append(args, opts.Rev)
	}
	result, err := gitexec.NewExecutor("").Run(ctx, gitexec.Command{Dir: repoPath, Args: append(args, "--")})
	if err != nil {
		return nil, false, gitexec.WrapRunError(err)
	}
	switch result.ExitCode {
	case 0:
	case 1:
		return nil, false, nil
	default:
		detail := strings.TrimSpace(result.Stderr)
		return nil, false, &gitexec.AppError{Message: "Search failed: " + strings.TrimPrefix(detail, "fatal: "), Detail: detail}
	}
	matches, truncated = parseGrep(result.Stdout, opts.Rev, opts.MaxMatches)
	return matches, truncated, nil
}

// Each line is "path\0line\0text"; with a revision, the path starts "<rev>:".
func parseGrep(output, rev string, max int) ([]GrepMatch, bool) {
	var matches []GrepMatch
	for _, line := range strings.Split(output, "\n") {
		fields := strings.SplitN(line, "\x00", 3)
		if len(fields) != 3 {
			continue
		}
		if max > 0 && len(matches) == max {
			return matches, true
		}
		n, _ := strconv.Atoi(fields[1])
		path := fields[0]
		if rev != "" {
			path = strings.TrimPrefix(path, rev+":")
		}
		matches = append(matches, GrepMatch{Path: path, Line: n, Text: strings.TrimRight(fields[2], "\r")})
	}
	return matches, false
}
