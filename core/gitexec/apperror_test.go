package gitexec

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestWrapResultKnownFailures(t *testing.T) {
	tests := []struct {
		name    string
		stderr  string
		wantMsg string
	}{
		{
			name:    "not a repository",
			stderr:  "fatal: not a git repository (or any of the parent directories): .git",
			wantMsg: "Not a Git repository.",
		},
		{
			name:    "bad ref",
			stderr:  "fatal: ambiguous argument 'badref': unknown revision or path not in the working tree.",
			wantMsg: "That reference doesn't exist.",
		},
		{
			name:    "credentials needed with prompts disabled",
			stderr:  "fatal: could not read Username for 'http://127.0.0.1:50964': terminal prompts disabled",
			wantMsg: "This remote needs credentials. Set up a credential helper, such as Git Credential Manager, for it.",
		},
		{
			name:    "ssh key passphrase without a terminal",
			stderr:  "read_passphrase: can't open /dev/tty: No such device or address\ngit@git.example.com: Permission denied (publickey).\nfatal: Could not read from remote repository.",
			wantMsg: "The SSH key needs its passphrase. Load the key into an SSH agent first.",
		},
		{
			name:    "unknown ssh host key",
			stderr:  "Host key verification failed.\nfatal: Could not read from remote repository.\n\nPlease make sure you have the correct access rights\nand the repository exists.",
			wantMsg: "This remote's SSH host key isn't trusted yet. Connect to it once from a terminal to add it to known_hosts.",
		},
		{
			name:    "ssh key rejected",
			stderr:  "git@git.example.com: Permission denied (publickey,password).\nfatal: Could not read from remote repository.",
			wantMsg: "The remote rejected the SSH key. Check that the key is loaded in an SSH agent and added to the remote.",
		},
		{
			name:    "https auth failure",
			stderr:  "remote: Support for password authentication was removed.\nfatal: Authentication failed for 'https://github.com/example/repo.git/'",
			wantMsg: "Authentication failed. Check the credentials Git uses for this remote.",
		},
		{
			name:    "untrusted https certificate",
			stderr:  "fatal: unable to access 'https://git.example.local/team/app.git/': SSL certificate problem: self-signed certificate in certificate chain",
			wantMsg: "The remote's HTTPS certificate isn't trusted. Point Git's http.sslCAInfo at the certificate for this server.",
		},
		{
			name:    "https repository not found",
			stderr:  "remote: Repository not found.\nfatal: repository 'https://github.com/example/missing.git/' not found",
			wantMsg: "The remote repository doesn't exist, or access to it is denied.",
		},
		{
			name:    "self-hosted repository not found",
			stderr:  "fatal: repository 'https://git.example.local/team/missing.git/' not found",
			wantMsg: "The remote repository doesn't exist, or access to it is denied.",
		},
		{
			name:    "path remote is not a repository",
			stderr:  "fatal: '/srv/git/missing.git' does not appear to be a git repository\nfatal: Could not read from remote repository.",
			wantMsg: "The remote repository doesn't exist, or access to it is denied.",
		},
		{
			name:    "access denied",
			stderr:  "remote: Write access to repository not granted.\nfatal: unable to access 'https://github.com/example/repo.git/': The requested URL returned error: 403",
			wantMsg: "Access to this remote was denied.",
		},
		{
			name:    "unknown host",
			stderr:  "fatal: unable to access 'https://github.com/example/repo.git/': Could not resolve host: github.com",
			wantMsg: "Can't find the remote host. Check the remote's address and the network connection.",
		},
		{
			name:    "unknown ssh host",
			stderr:  "ssh: Could not resolve hostname git.example.local: Name or service not known\nfatal: Could not read from remote repository.",
			wantMsg: "Can't find the remote host. Check the remote's address and the network connection.",
		},
		{
			name:    "connection refused",
			stderr:  "fatal: unable to access 'https://git.example.local/app.git/': Failed to connect to git.example.local port 443 after 2 ms: Connection refused",
			wantMsg: "Can't reach the remote host. Check the network connection.",
		},
		{
			name:    "ssh connection timed out",
			stderr:  "ssh: connect to host git.example.local port 22: Connection timed out\nfatal: Could not read from remote repository.",
			wantMsg: "Can't reach the remote host. Check the network connection.",
		},
		{
			name:    "stalled transfer",
			stderr:  "error: RPC failed; curl 28 Operation too slow. Less than 1000 bytes/sec transferred the last 60 seconds",
			wantMsg: "The remote stopped responding.",
		},
		{
			name:    "destination not empty",
			stderr:  "fatal: destination path 'C:\\projects\\homepage' already exists and is not an empty directory.",
			wantMsg: "Destination folder already has files in it.",
		},
		{
			name:    "no identity",
			stderr:  "Author identity unknown\n\n*** Please tell me who you are.\n\nRun\n\n  git config --global user.email \"you@example.com\"\n\nfatal: unable to auto-detect email address (got 'user@PC.(none)')",
			wantMsg: "Set a name and email for commits first.",
		},
		{
			name:    "no commits yet",
			stderr:  "fatal: your current branch 'main' does not have any commits yet",
			wantMsg: "This branch has no commits yet.",
		},
		{
			name:    "empty commit message",
			stderr:  "Aborting commit due to empty commit message.",
			wantMsg: "Commit message can't be empty.",
		},
		{
			name:    "unmerged branch delete",
			stderr:  "error: the branch 'feature' is not fully merged.\nIf you are sure you want to delete it, run 'git branch -D feature'.",
			wantMsg: "This branch has unmerged changes.",
		},
		{
			name:    "push no upstream",
			stderr:  "The current branch main has no upstream branch.\nTo push the current branch and set the remote as upstream, use\n\n    git push --set-upstream origin main",
			wantMsg: "This branch has no upstream configured.",
		},
		{
			name:    "pull no upstream",
			stderr:  "There is no tracking information for the current branch.\nPlease specify which branch you want to merge with.",
			wantMsg: "This branch has no upstream to pull from.",
		},
		{
			name:    "force push stale lease",
			stderr:  "To remote.git\n ! [rejected]        main -> main (stale info)\nerror: failed to push some refs to 'remote.git'",
			wantMsg: "Someone else pushed since your last fetch. Fetch and try again.",
		},
		{
			name:    "push rejected, remote has new work",
			stderr:  "To remote.git\n ! [rejected]        main -> main (fetch first)\nerror: failed to push some refs to 'remote.git'",
			wantMsg: "The remote has changes you don't have locally. Pull first.",
		},
		{
			name:    "push non-fast-forward",
			stderr:  "To remote.git\n ! [rejected]        main -> main (non-fast-forward)\nerror: failed to push some refs to 'remote.git'",
			wantMsg: "The remote has changes you don't have locally. Pull first.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WrapResult(Result{Stderr: tt.stderr})
			if err.Message != tt.wantMsg {
				t.Fatalf("Message = %q, want %q", err.Message, tt.wantMsg)
			}
			if err.Detail == "" {
				t.Fatal("expected raw detail to be preserved")
			}
			if err.Error() != err.Message {
				t.Fatalf("Error() = %q, want it to equal Message %q", err.Error(), err.Message)
			}
		})
	}
}

func TestWrapResultNothingToCommitReadsStdoutNotStderr(t *testing.T) {
	// Reported on stdout, unlike the other known failures.
	stdout := "On branch main\nnothing to commit, working tree clean"
	err := WrapResult(Result{Stdout: stdout})
	if err.Message != "Nothing is staged to commit." {
		t.Fatalf("Message = %q, want %q", err.Message, "Nothing is staged to commit.")
	}
	if err.Detail != stdout {
		t.Fatalf("Detail = %q, want %q", err.Detail, stdout)
	}
}

func TestWrapResultUnknownFailureFallsBackToGenericMessage(t *testing.T) {
	stderr := "some completely novel git error we've never seen before"
	err := WrapResult(Result{Stderr: stderr})
	if err.Message == "" || err.Message == stderr {
		t.Fatalf("expected a generic friendly message, got %q", err.Message)
	}
	if err.Detail != stderr {
		t.Fatalf("expected raw detail preserved for unknown errors, got %q", err.Detail)
	}
}

func TestWrapRunErrorClassifiesCancellation(t *testing.T) {
	err := WrapRunError(context.Canceled)
	if err.Message != "The operation was cancelled." {
		t.Fatalf("Message = %q", err.Message)
	}
}

func TestWrapRunErrorClassifiesTimeout(t *testing.T) {
	err := WrapRunError(context.DeadlineExceeded)
	if err.Message != "The operation timed out." {
		t.Fatalf("Message = %q", err.Message)
	}
}

func TestWrapRunErrorFallsBackForOtherFailures(t *testing.T) {
	raw := errors.New(`exec: "git": executable file not found in $PATH`)
	err := WrapRunError(raw)
	if err.Message != "Could not run git." {
		t.Fatalf("Message = %q", err.Message)
	}
	if err.Detail != raw.Error() {
		t.Fatalf("Detail = %q, want %q", err.Detail, raw.Error())
	}
}

func TestWrapRunErrorClassifiesWrappedCancellation(t *testing.T) {
	wrapped := fmt.Errorf("signal: killed: %w", context.Canceled)
	err := WrapRunError(wrapped)
	if err.Message != "The operation was cancelled." {
		t.Fatalf("Message = %q", err.Message)
	}
}
