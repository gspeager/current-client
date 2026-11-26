package gitexec

import (
	"context"
	"strings"
)

// Run runs a command with the default binary, returning a non-zero exit in the result.
func Run(ctx context.Context, cmd Command) (Result, error) {
	return NewExecutor("").Run(ctx, cmd)
}

// RunChecked runs a command with the default binary, failing on a non-zero exit.
func RunChecked(ctx context.Context, cmd Command) (Result, error) {
	return NewExecutor("").RunChecked(ctx, cmd)
}

// OutputIn is a checked command's stdout in dir, trimmed; env may be nil.
func OutputIn(ctx context.Context, dir string, env []string, args ...string) (string, error) {
	result, err := RunChecked(ctx, Command{Dir: dir, Env: env, Args: args})
	return strings.TrimSpace(result.Stdout), err
}

// Identity is the author and committer for commits the app makes for its own
// use, so they work before the user has set a name and email.
func Identity(email string) []string {
	return []string{
		"GIT_AUTHOR_NAME=Current Client", "GIT_AUTHOR_EMAIL=" + email,
		"GIT_COMMITTER_NAME=Current Client", "GIT_COMMITTER_EMAIL=" + email,
	}
}
