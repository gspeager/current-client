package tasks

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gspeager/current-client/internal/terminal"
)

// The callbacks run on the run's goroutines.
type Options struct {
	Env      []string
	OnOutput func(data []byte)
	OnDone   func(result Result)
}

type Result struct {
	ExitCode int
	Duration time.Duration
	Stopped  bool
}

type run struct {
	cmd     *exec.Cmd
	stopped atomic.Bool
	done    chan struct{}
}

type Runner struct {
	mu   sync.Mutex
	runs map[string]*run
}

func NewRunner() *Runner {
	return &Runner{runs: map[string]*run{}}
}

func shellCommand(command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd.exe", "/C", command)
	}
	// A login shell, as the terminal uses: an app opened from the desktop has a
	// bare PATH, without Homebrew, nvm and the like.
	shell, _ := terminal.Shell("", "")
	return exec.Command(shell, "-lc", command)
}

// Start runs command in dir, with its output and errors together, as a terminal
// would show them. Nothing is kept here: output only goes to OnOutput. The caller
// picks the id, so it knows the run before the first of its output can arrive.
func (r *Runner) Start(id, dir, command string, options Options) error {
	cmd := shellCommand(command)
	cmd.Dir = dir
	cmd.Env = options.Env
	reader, writer := io.Pipe()
	cmd.Stdout = writer
	cmd.Stderr = writer
	started := time.Now()
	if err := cmd.Start(); err != nil {
		return err
	}
	current := &run{cmd: cmd, done: make(chan struct{})}
	r.mu.Lock()
	r.runs[id] = current
	r.mu.Unlock()

	read := make(chan struct{})
	go func() {
		defer close(read)
		chunk := make([]byte, 32<<10)
		for {
			n, err := reader.Read(chunk)
			if n > 0 && options.OnOutput != nil {
				options.OnOutput(bytes.Clone(chunk[:n]))
			}
			if err != nil {
				return
			}
		}
	}()
	go func() {
		err := cmd.Wait()
		writer.Close()
		<-read
		r.mu.Lock()
		delete(r.runs, id)
		r.mu.Unlock()
		close(current.done)
		if options.OnDone != nil {
			options.OnDone(Result{ExitCode: exitCode(err), Duration: time.Since(started), Stopped: current.stopped.Load()})
		}
	}()
	return nil
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode()
	}
	return 1
}

// Stop ends the run and everything it started, and returns once it has.
func (r *Runner) Stop(id string) {
	r.mu.Lock()
	current, ok := r.runs[id]
	r.mu.Unlock()
	if !ok {
		return
	}
	current.stopped.Store(true)
	terminal.KillTree(current.cmd.Process.Pid)
	select {
	case <-current.done:
	case <-time.After(3 * time.Second):
	}
}

func (r *Runner) StopAll() {
	r.mu.Lock()
	ids := make([]string, 0, len(r.runs))
	for id := range r.runs {
		ids = append(ids, id)
	}
	r.mu.Unlock()
	for _, id := range ids {
		r.Stop(id)
	}
}

func (r *Runner) Running() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.runs) > 0
}
