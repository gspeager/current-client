package gitexec

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAFailedPushNamesItsCommandWithoutSecrets(t *testing.T) {
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("HOME", t.TempDir())
	remote, repo := filepath.Join(t.TempDir(), "remote.git"), t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "--bare", remote},
		{"-C", repo, "init", "-q"},
		{"-C", repo, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "c"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatal(string(out))
		}
	}
	// The remote refuses, and its message carries both secrets.
	hook := filepath.Join(remote, "hooks", "pre-receive")
	script := "#!/bin/sh\necho 'from https://me:hunter22@example.invalid/r.git with http.extraheader=Authorization: Bearer sekrit123' >&2\nexit 1\n"
	if err := os.WriteFile(hook, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	url := "https://me:hunter22@example.invalid/r.git"
	_, err := RunIn(context.Background(), repo,
		"-c", "http.extraheader=Authorization: Bearer sekrit123", "-c", "url."+remote+".insteadOf="+url, "push", url, "HEAD:main")
	var failed *AppError
	if !errors.As(err, &failed) {
		t.Fatalf("not an AppError: %v", err)
	}
	if failed.ExitCode != 1 || !strings.Contains(failed.Detail, "pre-receive hook declined") {
		t.Fatalf("the failure: %+v", failed)
	}
	for _, text := range []string{failed.Detail, failed.Message, failed.Command} {
		if strings.Contains(text, "hunter22") || strings.Contains(text, "sekrit123") {
			t.Fatalf("a secret shows: %q", text)
		}
	}
	want := "git -c 'http.extraheader=" + RedactedSecret + "' -c 'url." + remote + ".insteadOf=https://" + RedactedSecret +
		"@example.invalid/r.git' push 'https://" + RedactedSecret + "@example.invalid/r.git' HEAD:main"
	if failed.Command != want {
		t.Fatalf("the command isn't the one that ran: %q", failed.Command)
	}
}

func TestAnAppErrorOfTheAppsOwnStaysItself(t *testing.T) {
	if got := withCommand(ErrCommandLineToolsMissing, []string{"status"}, 1); got == ErrCommandLineToolsMissing || ErrCommandLineToolsMissing.Command != "" {
		t.Fatal("adding the command changed the shared error")
	}
}
