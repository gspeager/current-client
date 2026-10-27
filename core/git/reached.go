package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

// reachSlack is how far a commit's date may run ahead of an ancestor's
// before Reached misses it: the walk stops a day before the oldest commit
// asked about, which covers ordinary clock skew between machines.
const reachSlack = 24 * 60 * 60

// Reached says which of commits are in tip's history, keyed as given. A
// commit or a tip the repository doesn't have is left out, since Git can't
// tell: the caller shows it as unknown. Three processes however many
// commits: one to find them, one for their dates, one walk from the tip.
func Reached(ctx context.Context, repoPath, tip string, commits []string) (map[string]bool, error) {
	reached := map[string]bool{}
	ids, err := commitIDs(ctx, repoPath, append([]string{tip}, commits...))
	if err != nil || ids[tip] == "" || len(commits) == 0 {
		return reached, err
	}
	var present []string
	for _, c := range commits {
		if ids[c] != "" {
			present = append(present, ids[c])
		}
	}
	if len(present) == 0 {
		return reached, nil
	}
	dates, err := runResult(ctx, repoPath, append([]string{"log", "--no-walk=unsorted", "--format=%ct"}, present...)...)
	if err != nil {
		return nil, err
	}
	oldest := int64(-1)
	for _, line := range strings.Fields(dates.Stdout) {
		if t, err := strconv.ParseInt(line, 10, 64); err == nil && (oldest < 0 || t < oldest) {
			oldest = t
		}
	}
	walk, err := runResult(ctx, repoPath, "rev-list", "--since="+strconv.FormatInt(oldest-reachSlack, 10), ids[tip])
	if err != nil {
		return nil, err
	}
	history := map[string]bool{}
	for _, id := range strings.Fields(walk.Stdout) {
		history[id] = true
	}
	for _, c := range commits {
		if id := ids[c]; id != "" {
			reached[c] = history[id]
		}
	}
	return reached, nil
}

// commitIDs resolves each rev to its commit's full id with one cat-file;
// one that isn't a commit here maps to "".
func commitIDs(ctx context.Context, repoPath string, revs []string) (map[string]string, error) {
	var input strings.Builder
	for _, rev := range revs {
		if strings.ContainsAny(rev, "\n ") {
			return nil, fmt.Errorf("can't look up %q", rev)
		}
		input.WriteString(rev + "^{commit}\n")
	}
	result, err := gitexec.NewExecutor("").RunChecked(ctx, gitexec.Command{
		Dir:   repoPath,
		Args:  []string{"cat-file", "--batch-check=%(objectname)"},
		Stdin: strings.NewReader(input.String()),
	})
	if err != nil {
		return nil, err
	}
	ids := map[string]string{}
	lines := strings.Split(strings.TrimRight(result.Stdout, "\n"), "\n")
	for i, rev := range revs {
		if i < len(lines) && !strings.Contains(lines[i], " ") {
			ids[rev] = lines[i]
		}
	}
	return ids, nil
}
