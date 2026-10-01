package git

import (
	"context"
	"strconv"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

type CommitOptions struct {
	Amend      bool
	AllowEmpty bool
	Sign       bool
}

// Commit always passes an explicit signing flag, overriding commit.gpgsign.
func Commit(ctx context.Context, repoPath, message string, opts CommitOptions) error {
	args := []string{"commit", "-m", message}
	if opts.Amend {
		args = append(args, "--amend")
	}
	if opts.AllowEmpty {
		args = append(args, "--allow-empty")
	}
	if opts.Sign {
		args = append(args, "-S")
	} else {
		args = append(args, "--no-gpg-sign")
	}
	_, err := runResult(ctx, repoPath, args...)
	return err
}

func GPGSignDefault(ctx context.Context, repoPath string) bool {
	return configValue(ctx, repoPath, "commit.gpgsign") == "true"
}

func LastCommitMessage(ctx context.Context, repoPath string) (string, error) {
	result, err := runResult(ctx, repoPath, "log", "-1", "--pretty=%B")
	if err != nil {
		return "", err
	}
	return strings.TrimRight(result.Stdout, "\n"), nil
}

func UndoLastCommit(ctx context.Context, repoPath string) error {
	_, err := runResult(ctx, repoPath, "reset", "--soft", "HEAD~1")
	return err
}

type Author struct {
	Name  string
	Email string
}

const (
	recentAuthorScanLimit   = 500
	recentAuthorResultLimit = 20
)

// RecentAuthors returns distinct authors, most recently active first.
func RecentAuthors(ctx context.Context, repoPath string) ([]Author, error) {
	result, err := runResult(ctx, repoPath, "log", "-n", strconv.Itoa(recentAuthorScanLimit), "--format=%aN%x1f%aE")
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var authors []Author
	for _, line := range strings.Split(result.Stdout, "\n") {
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "\x1f", 2)
		if len(fields) != 2 || seen[line] {
			continue
		}
		seen[line] = true
		authors = append(authors, Author{Name: fields[0], Email: fields[1]})
		if len(authors) >= recentAuthorResultLimit {
			break
		}
	}
	return authors, nil
}

// IsLastCommitPushed treats a branch without an upstream as unpushed; that's
// the common case, so the upstream lookup's failure isn't surfaced.
func IsLastCommitPushed(ctx context.Context, repoPath string) (bool, error) {
	upstream, err := gitexec.NewExecutor("").Run(ctx, gitexec.Command{
		Dir:  repoPath,
		Args: []string{"rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"},
	})
	if err != nil || upstream.ExitCode != 0 {
		return false, nil
	}

	result, err := runResult(ctx, repoPath, "rev-list", "--count", "@{u}..HEAD")
	if err != nil {
		return false, err
	}
	ahead, err := strconv.Atoi(strings.TrimSpace(result.Stdout))
	if err != nil {
		return false, err
	}
	return ahead == 0, nil
}
