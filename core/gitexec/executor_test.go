package gitexec

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func writeStub(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()

	var path, content string
	if runtime.GOOS == "windows" {
		path = filepath.Join(dir, "stub.bat")
		content = "@echo off\r\n" + body + "\r\n"
	} else {
		path = filepath.Join(dir, "stub.sh")
		content = "#!/bin/sh\n" + body + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func TestExecutorRunSuccess(t *testing.T) {
	e := NewExecutor("")
	result, err := e.Run(context.Background(), Command{Args: []string{"--version"}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0", result.ExitCode)
	}
	if !strings.HasPrefix(strings.TrimSpace(result.Stdout), "git version") {
		t.Fatalf("unexpected stdout: %q", result.Stdout)
	}
}

func TestExecutorRunFailingCommandReportsExitCodeNotError(t *testing.T) {
	e := NewExecutor("")
	result, err := e.Run(context.Background(), Command{Args: []string{"not-a-real-subcommand"}})
	if err != nil {
		t.Fatalf("Run should report a bad subcommand via ExitCode, not error: %v", err)
	}
	if result.ExitCode == 0 {
		t.Fatal("expected a non-zero exit code for an invalid subcommand")
	}
	if result.Stderr == "" {
		t.Fatal("expected stderr output for an invalid subcommand")
	}
}

func TestExecutorRunRespectsWorkingDirectory(t *testing.T) {
	// Resolved, since the child reports its real cwd: macOS's temp dir is under
	// /var, a symlink to /private/var.
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var script string
	if runtime.GOOS == "windows" {
		script = "cd"
	} else {
		script = "pwd"
	}
	stub := writeStub(t, script)

	e := NewExecutor(stub)
	result, err := e.Run(context.Background(), Command{Dir: dir})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := filepath.Clean(strings.TrimSpace(result.Stdout))
	want := filepath.Clean(dir)
	if !strings.EqualFold(got, want) {
		t.Fatalf("cwd = %q, want %q", got, want)
	}
}

func TestExecutorRunPassesEnv(t *testing.T) {
	var script string
	if runtime.GOOS == "windows" {
		script = "echo %GIT_CLIENT_TEST_VAR%"
	} else {
		script = `echo "$GIT_CLIENT_TEST_VAR"`
	}
	stub := writeStub(t, script)

	e := NewExecutor(stub)
	result, err := e.Run(context.Background(), Command{Env: []string{"GIT_CLIENT_TEST_VAR=hello"}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := strings.TrimSpace(result.Stdout); got != "hello" {
		t.Fatalf("env var = %q, want %q", got, "hello")
	}
}

func TestExecutorRunForcesStableLocale(t *testing.T) {
	var script string
	if runtime.GOOS == "windows" {
		script = "echo %LC_ALL%"
	} else {
		script = `echo "$LC_ALL"`
	}
	stub := writeStub(t, script)

	e := NewExecutor(stub)
	result, err := e.Run(context.Background(), Command{Env: []string{"LC_ALL=fr_FR.UTF-8"}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := strings.TrimSpace(result.Stdout); got != "C" {
		t.Fatalf("LC_ALL = %q, want %q", got, "C")
	}
}

func TestExecutorRunPassesStdin(t *testing.T) {
	var script string
	if runtime.GOOS == "windows" {
		script = `findstr "^"`
	} else {
		script = "cat"
	}
	stub := writeStub(t, script)

	e := NewExecutor(stub)
	result, err := e.Run(context.Background(), Command{Stdin: strings.NewReader("hello from stdin")})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := strings.TrimSpace(result.Stdout); got != "hello from stdin" {
		t.Fatalf("stdout = %q, want %q", got, "hello from stdin")
	}
}

func TestNewExecutorUsesOverriddenDefaultBinary(t *testing.T) {
	t.Cleanup(func() { SetDefaultBinary("") })

	stub := writeStub(t, "echo overridden")

	SetDefaultBinary(stub)
	result, err := NewExecutor("").Run(context.Background(), Command{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := strings.TrimSpace(result.Stdout); got != "overridden" {
		t.Fatalf("stdout = %q, want %q (expected the overridden binary to run)", got, "overridden")
	}
}

func TestSetDefaultBinaryEmptyResetsToGit(t *testing.T) {
	t.Cleanup(func() { SetDefaultBinary("") })

	SetDefaultBinary(writeStub(t, "echo overridden"))
	SetDefaultBinary("")

	result, err := NewExecutor("").Run(context.Background(), Command{Args: []string{"--version"}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(result.Stdout), "git version") {
		t.Fatalf("expected a reset to plain git, got stdout: %q", result.Stdout)
	}
}

func TestExecutorRunCancellationTerminatesProcess(t *testing.T) {
	// Not a .bat stub: killing cmd.exe wouldn't kill its child process, since
	// Kill() only targets the process Go itself spawned.
	var binary string
	var args []string
	if runtime.GOOS == "windows" {
		binary, args = "ping", []string{"-n", "31", "127.0.0.1"}
	} else {
		binary, args = "sleep", []string{"30"}
	}

	ctx, cancel := context.WithCancel(context.Background())
	e := NewExecutor(binary)

	done := make(chan error, 1)
	go func() {
		_, err := e.Run(ctx, Command{Args: args})
		done <- err
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return within 5s of cancellation; process was not terminated")
	}
}

func TestExecutorFailsInsteadOfPromptingForCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Basic realm="test"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// credential.helper= clears any helper configured on this machine, leaving
	// git's own terminal prompt as the only way to ask.
	result, err := NewExecutor("").Run(ctx, Command{
		Dir:  t.TempDir(),
		Args: []string{"-c", "credential.helper=", "ls-remote", server.URL + "/repo.git"},
	})
	if err != nil {
		t.Fatalf("Run: %v (git waited on a prompt until the timeout)", err)
	}
	if result.ExitCode == 0 {
		t.Fatal("expected ls-remote to fail without credentials")
	}
	if !strings.Contains(result.Stderr, "terminal prompts disabled") {
		t.Fatalf("stderr = %q, want git to report terminal prompts disabled", result.Stderr)
	}
}

func TestExecutorAbortsStalledHTTPTransfer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := NewExecutor("").RunChecked(ctx, Command{
		Dir:  t.TempDir(),
		Env:  []string{"GIT_HTTP_LOW_SPEED_TIME=2"},
		Args: []string{"ls-remote", server.URL + "/repo.git"},
	})
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Message != "The remote stopped responding." {
		t.Fatalf("err = %v, want the stalled transfer aborted as \"The remote stopped responding.\"", err)
	}
}
