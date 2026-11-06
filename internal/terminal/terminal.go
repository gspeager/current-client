// Package terminal runs shells on pseudo-terminals for the docked terminal panel.
// What passes through a session is never logged or written to disk, since it can
// hold credentials. The shell's own configuration is never edited.
package terminal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	pty "github.com/aymanbagabas/go-pty"
)

const (
	flushEvery = 8 * time.Millisecond
	// Past this much unacknowledged output, reading stops until the view catches
	// up, so something like `yes` can't flood the webview.
	mostUnacknowledged = 2 << 20
)

// Options starts a session. OnOutput and OnExit run on the session's goroutines.
type Options struct {
	Dir      string
	Name     string
	Args     []string
	Env      []string
	Cols     int
	Rows     int
	OnOutput func(data []byte)
	OnExit   func(code int)
}

type session struct {
	id      string
	repo    string
	console pty.Pty
	cmd     *pty.Cmd
	done    chan struct{}

	mu      sync.Mutex
	unread  sync.Cond
	buffer  []byte
	unacked int
	exited  bool
}

type Manager struct {
	mu       sync.Mutex
	sessions map[string]*session
	nextID   atomic.Int64
}

func NewManager() *Manager {
	return &Manager{sessions: map[string]*session{}}
}

// Start runs the command on a new pseudo-terminal and returns the session's ID.
func (m *Manager) Start(repo string, options Options) (string, error) {
	console, err := pty.New()
	if err != nil {
		return "", err
	}
	if options.Cols > 0 && options.Rows > 0 {
		if err := console.Resize(options.Cols, options.Rows); err != nil {
			console.Close()
			return "", err
		}
	}
	cmd := console.Command(options.Name, options.Args...)
	cmd.Dir = options.Dir
	cmd.Env = options.Env
	if err := cmd.Start(); err != nil {
		console.Close()
		return "", err
	}
	s := &session{
		id:      fmt.Sprintf("t%d", m.nextID.Add(1)),
		repo:    repo,
		console: console,
		cmd:     cmd,
		done:    make(chan struct{}),
	}
	s.unread.L = &s.mu
	m.mu.Lock()
	m.sessions[s.id] = s
	m.mu.Unlock()

	flushed := make(chan struct{})
	go s.flush(options.OnOutput, flushed)
	go s.read()
	go func() {
		code := 0
		if err := cmd.Wait(); err != nil {
			code = exitCode(err)
		}
		// Output a child still holds the terminal open for doesn't outlive the shell.
		s.mu.Lock()
		s.exited = true
		s.unread.Broadcast()
		s.mu.Unlock()
		m.mu.Lock()
		delete(m.sessions, s.id)
		m.mu.Unlock()
		close(s.done)
		<-flushed
		console.Close()
		if options.OnExit != nil {
			options.OnExit(code)
		}
	}()
	return s.id, nil
}

func exitCode(err error) int {
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}
	return 1
}

func (s *session) read() {
	chunk := make([]byte, 32<<10)
	for {
		s.mu.Lock()
		for s.unacked >= mostUnacknowledged && !s.exited {
			s.unread.Wait()
		}
		s.mu.Unlock()
		n, err := s.console.Read(chunk)
		if n > 0 {
			s.mu.Lock()
			s.buffer = append(s.buffer, chunk[:n]...)
			s.mu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

// flush batches output, so a burst reaches the view as one write rather than
// thousands of events.
func (s *session) flush(output func([]byte), finished chan struct{}) {
	defer close(finished)
	send := func() {
		s.mu.Lock()
		data := s.buffer
		s.buffer = nil
		s.unacked += len(data)
		s.mu.Unlock()
		if len(data) > 0 && output != nil {
			output(data)
		}
	}
	ticker := time.NewTicker(flushEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			send()
		case <-s.done:
			// A final moment for what the shell wrote as it exited.
			time.Sleep(2 * flushEvery)
			send()
			return
		}
	}
}

func (m *Manager) get(id string) (*session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, fmt.Errorf("this terminal has closed")
	}
	return s, nil
}

func (m *Manager) Write(id string, data []byte) error {
	s, err := m.get(id)
	if err != nil {
		return err
	}
	_, err = s.console.Write(data)
	return err
}

func (m *Manager) Resize(id string, cols, rows int) error {
	s, err := m.get(id)
	if err != nil {
		return err
	}
	return s.console.Resize(cols, rows)
}

// Ack says how much output the view has drawn.
func (m *Manager) Ack(id string, bytes int) {
	s, err := m.get(id)
	if err != nil {
		return
	}
	s.mu.Lock()
	s.unacked = max(0, s.unacked-bytes)
	s.unread.Broadcast()
	s.mu.Unlock()
}

// Busy says whether the shell is running anything beyond itself.
func (m *Manager) Busy(id string) bool {
	s, err := m.get(id)
	if err != nil || s.cmd.Process == nil {
		return false
	}
	return len(descendants(context.Background(), s.cmd.Process.Pid)) > 0
}

// AnyBusy says whether a session for repo is busy, or any at all when repo is "".
func (m *Manager) AnyBusy(repo string) bool {
	m.mu.Lock()
	var ids []string
	for id, s := range m.sessions {
		if repo == "" || s.repo == repo {
			ids = append(ids, id)
		}
	}
	m.mu.Unlock()
	for _, id := range ids {
		if m.Busy(id) {
			return true
		}
	}
	return false
}

// Close ends the session and everything running in it, and returns once it has.
func (m *Manager) Close(id string) error {
	s, err := m.get(id)
	if err != nil {
		return nil
	}
	if s.cmd.Process != nil {
		killTree(s.cmd.Process.Pid)
	}
	s.console.Close()
	select {
	case <-s.done:
	case <-time.After(3 * time.Second):
	}
	return nil
}

// CloseRepo ends every session for a repository.
func (m *Manager) CloseRepo(repo string) {
	m.mu.Lock()
	var ids []string
	for id, s := range m.sessions {
		if s.repo == repo {
			ids = append(ids, id)
		}
	}
	m.mu.Unlock()
	for _, id := range ids {
		_ = m.Close(id)
	}
}

// CloseAll ends every session, for quitting the app.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	m.mu.Unlock()
	for _, id := range ids {
		_ = m.Close(id)
	}
}

// Shell chooses what to run. choice is "" for the default, or on Windows
// "powershell", "git-bash" or "cmd".
func Shell(choice, gitBinary string) (name string, args []string) {
	if runtime.GOOS == "windows" {
		return windowsShell(choice, gitBinary)
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	return shell, []string{"-l"}
}

func windowsShell(choice, gitBinary string) (string, []string) {
	if choice == "cmd" {
		return "cmd.exe", nil
	}
	if choice == "git-bash" {
		// git-bash.exe would open its own window.
		if bash := gitBashBinary(gitBinary); bash != "" {
			return bash, []string{"--login", "-i"}
		}
	}
	if path, err := lookPath("pwsh.exe"); err == nil {
		return path, []string{"-NoLogo"}
	}
	return "powershell.exe", []string{"-NoLogo"}
}

// gitBashBinary finds bin\bash.exe from git.exe in Git for Windows' bin or cmd.
func gitBashBinary(gitBinary string) string {
	if gitBinary == "" {
		return ""
	}
	dir := filepath.Dir(gitBinary)
	for _, root := range []string{filepath.Dir(dir), filepath.Dir(filepath.Dir(dir))} {
		candidate := filepath.Join(root, "bin", "bash.exe")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

// Environment is the app's own, with a terminal type, and the directory of the
// Git the app uses on the PATH, since Windows's shells other than Git Bash don't
// add it themselves.
func Environment(base []string, gitBinary string) []string {
	env := make([]string, 0, len(base)+3)
	var path string
	for _, entry := range base {
		key, value, _ := strings.Cut(entry, "=")
		switch {
		case strings.EqualFold(key, "PATH"):
			path = value
		case key == "TERM", key == "COLORTERM":
		default:
			env = append(env, entry)
		}
	}
	if gitBinary != "" && filepath.IsAbs(gitBinary) {
		dir := filepath.Dir(gitBinary)
		if !strings.Contains(string(os.PathListSeparator)+path+string(os.PathListSeparator), string(os.PathListSeparator)+dir+string(os.PathListSeparator)) {
			path = dir + string(os.PathListSeparator) + path
		}
	}
	return append(env, "PATH="+path, "TERM=xterm-256color", "COLORTERM=truecolor")
}
