package gitexec

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
)

type Command struct {
	Dir   string
	Env   []string
	Args  []string
	Stdin io.Reader
}

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type Executor struct {
	binary string
}

var defaultBinary = struct {
	mu   sync.RWMutex
	path string
}{path: "git"}

// SetDefaultBinary sets the binary for executors created with an empty path;
// an empty value resets it to "git" on PATH.
func SetDefaultBinary(path string) {
	defaultBinary.mu.Lock()
	defer defaultBinary.mu.Unlock()
	if path == "" {
		path = "git"
	}
	defaultBinary.path = path
}

func NewExecutor(binary string) *Executor {
	if binary == "" {
		defaultBinary.mu.RLock()
		binary = defaultBinary.path
		defaultBinary.mu.RUnlock()
	}
	return &Executor{binary: binary}
}

func (e *Executor) Run(ctx context.Context, cmd Command) (Result, error) {
	args := cmd.Args
	var credEnv []string
	if cred, ok := credentialFrom(ctx); ok {
		var credArgs []string
		credArgs, credEnv = credentialArgs(cred)
		args = append(credArgs, args...)
	}
	c := exec.CommandContext(ctx, e.binary, args...)
	c.Dir = cmd.Dir
	c.Stdin = cmd.Stdin
	detachFromTerminal(c)
	// Git must fail instead of waiting on a terminal prompt nobody can answer.
	// GUI credential helpers (Git Credential Manager) still show their windows.
	env := append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	// An HTTP transfer that stalls below 1 KB/s for a minute is aborted rather
	// than left running until someone presses Cancel.
	env = append(env, "GIT_HTTP_LOW_SPEED_LIMIT=1000", "GIT_HTTP_LOW_SPEED_TIME=60")
	env = append(env, cmd.Env...)
	env = append(env, credEnv...)
	// Last so it wins: output must stay English for the parsers.
	c.Env = append(env, "LC_ALL=C")

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	runErr := c.Run()
	result := Result{Stdout: stdout.String(), Stderr: stderr.String()}

	if ctx.Err() != nil {
		return result, ctx.Err()
	}

	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	}
	return result, runErr
}

// RunChecked also turns a non-zero exit code into an *AppError.
func (e *Executor) RunChecked(ctx context.Context, cmd Command) (Result, error) {
	result, err := e.Run(ctx, cmd)
	if err != nil {
		return Result{}, WrapRunError(err)
	}
	if result.ExitCode != 0 {
		return Result{}, WrapResult(result)
	}
	return result, nil
}
