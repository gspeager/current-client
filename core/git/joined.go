package git

import (
	"context"
	"strconv"
	"strings"
	"time"
)

// Joined is where a commit arrived on a branch's own line of history, the
// line its first parents trace. At is the commit on that line that brought
// it there: the commit itself when it was made on the line or fast-forwarded
// onto it (OnLine), else the merge commit. At is "" when the branch doesn't
// include the commit.
type Joined struct {
	At     string
	OnLine bool
	Date   time.Time
}

// JoinedInto says for each commit when into came to include it, from one
// walk of into's history however many commits are asked about. A commit on
// an unmerged or unrelated branch isn't included, so it has no At; a squash
// or rebase merge brings the changes without the commit, which MergedInto
// finds instead.
func JoinedInto(ctx context.Context, repoPath, into string, commits []string) ([]Joined, error) {
	joined := make([]Joined, len(commits))
	if len(commits) == 0 {
		return joined, nil
	}
	resolved, err := runResult(ctx, repoPath, append([]string{"rev-parse"}, commits...)...)
	if err != nil {
		return nil, err
	}
	shas := strings.Fields(resolved.Stdout)
	result, err := runResult(ctx, repoPath, "log", "--topo-order", "--format=%H %P %ct", into, "--")
	if err != nil {
		return nil, err
	}
	type commit struct {
		parents []string
		date    int64
	}
	graph := map[string]commit{}
	var tip string
	for _, line := range strings.Split(result.Stdout, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		date, _ := strconv.ParseInt(fields[len(fields)-1], 10, 64)
		graph[fields[0]] = commit{parents: fields[1 : len(fields)-1], date: date}
		if tip == "" {
			tip = fields[0]
		}
	}
	var line []string
	for sha := tip; sha != ""; {
		line = append(line, sha)
		if parents := graph[sha].parents; len(parents) > 0 {
			sha = parents[0]
		} else {
			sha = ""
		}
	}
	// Every commit belongs to the oldest line commit that reaches it: the
	// first one to include it.
	at := map[string]string{}
	var stack []string
	for i := len(line) - 1; i >= 0; i-- {
		stack = append(stack[:0], line[i])
		for len(stack) > 0 {
			sha := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if _, seen := at[sha]; seen {
				continue
			}
			at[sha] = line[i]
			stack = append(stack, graph[sha].parents...)
		}
	}
	for i, sha := range shas {
		if l, ok := at[sha]; ok {
			joined[i] = Joined{At: l, OnLine: l == sha, Date: time.Unix(graph[l].date, 0)}
		}
	}
	return joined, nil
}
