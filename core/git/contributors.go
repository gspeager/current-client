package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type ContributorStat struct {
	Name    string
	Email   string
	Commits int
}

// Contributors passes HEAD explicitly: without a revision, shortlog reads
// from stdin and returns nothing.
func Contributors(ctx context.Context, repoPath, since, until string) ([]ContributorStat, error) {
	args := []string{"shortlog", "-sn", "-e"}
	if since != "" {
		args = append(args, "--since="+since)
	}
	if until != "" {
		args = append(args, "--until="+until)
	}
	args = append(args, "HEAD")
	result, err := runResult(ctx, repoPath, args...)
	if err != nil {
		if hasNoCommits(ctx, repoPath) {
			return nil, nil
		}
		return nil, err
	}
	return parseShortlog(result.Stdout)
}

func parseShortlog(output string) ([]ContributorStat, error) {
	var stats []ContributorStat
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(strings.TrimLeft(line, " \t"), "\t", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("malformed shortlog entry: %q", line)
		}
		count, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("malformed shortlog count %q: %w", parts[0], err)
		}
		nameEmail := parts[1]
		open := strings.LastIndex(nameEmail, "<")
		end := strings.LastIndex(nameEmail, ">")
		if open == -1 || end == -1 || end < open {
			return nil, fmt.Errorf("malformed shortlog author %q", nameEmail)
		}
		stats = append(stats, ContributorStat{
			Name:    strings.TrimSpace(nameEmail[:open]),
			Email:   nameEmail[open+1 : end],
			Commits: count,
		})
	}
	return stats, nil
}
