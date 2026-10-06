package gitexec

import (
	"context"
	"errors"
	"strings"
)

type AppError struct {
	Message string
	Detail  string
}

func (e *AppError) Error() string {
	return e.Message
}

type outputStream int

const (
	stderrStream outputStream = iota
	stdoutStream
)

// The frontend offers to sign in when it sees one of these.
const (
	credentialsNeededMessage = "This remote needs credentials. Set up a credential helper, such as Git Credential Manager, for it."
	authFailedMessage        = "Authentication failed. Check the credentials Git uses for this remote."
)

// First match wins.
var knownErrorPatterns = []struct {
	stream     outputStream
	substrings []string
	message    string
}{
	{stderrStream, []string{"not a git repository"}, "Not a Git repository."},
	{stderrStream, []string{"unknown revision or path not in the working tree", "bad revision"}, "That reference doesn't exist."},
	// Network failures, most specific first: an SSH passphrase failure also
	// ends in "Permission denied (publickey)".
	{stderrStream, []string{"terminal prompts disabled", "could not read Username", "could not read Password"}, credentialsNeededMessage},
	{stderrStream, []string{"read_passphrase", "incorrect passphrase"}, "The SSH key needs its passphrase. Load the key into an SSH agent first."},
	{stderrStream, []string{"Host key verification failed"}, "This remote's SSH host key isn't trusted yet. Connect to it once from a terminal to add it to known_hosts."},
	{stderrStream, []string{"Permission denied (publickey"}, "The remote rejected the SSH key. Check that the key is loaded in an SSH agent and added to the remote."},
	{stderrStream, []string{"Authentication failed"}, authFailedMessage},
	{stderrStream, []string{"SSL certificate problem", "server certificate verification failed"}, "The remote's HTTPS certificate isn't trusted. Point Git's http.sslCAInfo at the certificate for this server."},
	{stderrStream, []string{"Repository not found", "fatal: repository '", "does not appear to be a git repository"}, "The remote repository doesn't exist, or access to it is denied."},
	{stderrStream, []string{"The requested URL returned error: 403"}, "Access to this remote was denied."},
	{stderrStream, []string{"Could not resolve host"}, "Can't find the remote host. Check the remote's address and the network connection."},
	{stderrStream, []string{"Failed to connect", "Connection refused", "Connection timed out", "Operation timed out", "Network is unreachable"}, "Can't reach the remote host. Check the network connection."},
	{stderrStream, []string{"Operation too slow"}, "The remote stopped responding."},
	{stderrStream, []string{"already exists and is not an empty directory"}, "Destination folder already has files in it."},
	{stderrStream, []string{"Please tell me who you are", "unable to auto-detect email address"}, "Set a name and email for commits first."},
	{stderrStream, []string{"does not have any commits yet"}, "This branch has no commits yet."},
	{stderrStream, []string{"Aborting commit due to empty commit message"}, "Commit message can't be empty."},
	// `git commit` with nothing staged reports this on stdout, not stderr.
	{stdoutStream, []string{"nothing to commit"}, "Nothing is staged to commit."},
	{stderrStream, []string{"is not a valid branch name"}, "That isn't a valid branch name. Branch names can't contain spaces or any of ~ ^ : ? * [ \\."},
	{stderrStream, []string{"a branch named"}, "A branch with that name already exists."},
	{stderrStream, []string{"is not fully merged"}, "This branch has unmerged changes."},
	{stderrStream, []string{"contains modified or untracked files"}, "This worktree has changes."},
	// Git 2.42 changed "is already checked out at" to "is already used by worktree at".
	{stderrStream, []string{"is already checked out at", "is already used by worktree at"}, "That branch is checked out in another worktree."},
	// Checkout, merge, pull, cherry-pick and stash pop all refuse this way.
	{stderrStream, []string{"Your local changes to the following files would be overwritten"}, "Uncommitted changes would be overwritten. Commit or stash them first."},
	{stderrStream, []string{"untracked working tree files would be overwritten"}, "Untracked files would be overwritten. Move or delete them first."},
	{stderrStream, []string{"no tracking information"}, "This branch has no upstream to pull from."},
	{stderrStream, []string{"has no upstream branch"}, "This branch has no upstream configured."},
	{stderrStream, []string{"(stale info)"}, "Someone else pushed since your last fetch. Fetch and try again."},
	{stderrStream, []string{"(non-fast-forward)", "(fetch first)"}, "The remote has changes you don't have locally. Pull first."},
}

func WrapResult(result Result) *AppError {
	stderr := strings.TrimSpace(result.Stderr)
	stdout := strings.TrimSpace(result.Stdout)

	for _, pattern := range knownErrorPatterns {
		text := stderr
		if pattern.stream == stdoutStream {
			text = stdout
		}
		for _, substr := range pattern.substrings {
			if strings.Contains(text, substr) {
				return &AppError{Message: pattern.message, Detail: text}
			}
		}
	}
	return &AppError{Message: "Git command failed.", Detail: stderr}
}

func WrapRunError(err error) *AppError {
	if errors.Is(err, context.Canceled) {
		return &AppError{Message: "The operation was cancelled.", Detail: err.Error()}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &AppError{Message: "The operation timed out.", Detail: err.Error()}
	}
	return &AppError{Message: "Could not run git.", Detail: err.Error()}
}
