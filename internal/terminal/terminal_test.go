package terminal

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type recorder struct {
	mu     sync.Mutex
	output bytes.Buffer
	code   chan int
}

func (r *recorder) text() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.output.String()
}

func (r *recorder) waitFor(t *testing.T, text string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !strings.Contains(r.text(), text) {
		if time.Now().After(deadline) {
			t.Fatalf("no %q in %q", text, r.text())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func start(t *testing.T, manager *Manager, dir string) (string, *recorder) {
	t.Helper()
	// Bare names, so the shell must come from the PATH and not the repository.
	// A plain sh, so a developer's own profile can't slow or change this.
	name, args := "sh", []string(nil)
	if runtime.GOOS == "windows" {
		name, args = "powershell.exe", []string{"-NoLogo"}
	}
	r := &recorder{code: make(chan int, 1)}
	id, err := manager.Start(dir, Options{
		Dir:  dir,
		Name: name,
		Args: args,
		Env:  append(Environment(os.Environ(), ""), "PS1=$ "),
		Cols: 80,
		Rows: 24,
		OnOutput: func(data []byte) {
			r.mu.Lock()
			r.output.Write(data)
			r.mu.Unlock()
			manager.Ack("", 0)
		},
		OnExit: func(code int) { r.code <- code },
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(id) })
	return id, r
}

func resolved(t *testing.T, dir string) string {
	t.Helper()
	path, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunsACommandInTheRepositoryAndReadsItsOutputAndExit(t *testing.T) {
	dir := t.TempDir()
	manager := NewManager()
	id, r := start(t, manager, dir)

	if err := manager.Write(id, []byte("pwd; echo finished-$((40+2))\n")); err != nil {
		t.Fatal(err)
	}
	r.waitFor(t, "finished-42")
	if !strings.Contains(r.text(), filepath.Base(resolved(t, dir))) {
		t.Fatalf("not in the repository: %q", r.text())
	}

	if err := manager.Write(id, []byte("exit 3\n")); err != nil {
		t.Fatal(err)
	}
	select {
	case code := <-r.code:
		if code != 3 {
			t.Fatalf("exit code = %d", code)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the shell didn't exit")
	}
	if err := manager.Write(id, []byte("x")); err == nil {
		t.Fatal("a closed terminal should refuse input")
	}
}

func TestBusyFollowsWhatTheShellIsRunning(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sleep")
	}
	dir := t.TempDir()
	manager := NewManager()
	id, r := start(t, manager, dir)
	if err := manager.Write(id, []byte("echo ready\n")); err != nil {
		t.Fatal(err)
	}
	r.waitFor(t, "ready")
	if manager.Busy(id) || manager.AnyBusy(dir) {
		t.Fatal("an idle shell isn't busy")
	}
	if err := manager.Write(id, []byte("sleep 30\n")); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for !manager.Busy(id) {
		if time.Now().After(deadline) {
			t.Fatal("sleep never showed as running")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !manager.AnyBusy(dir) || !manager.AnyBusy("") || manager.AnyBusy("/elsewhere") {
		t.Fatal("AnyBusy should follow the repository")
	}
}

func TestCloseLeavesNoProcessBehind(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sleep and ps")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "pids")
	manager := NewManager()
	id, r := start(t, manager, dir)
	script := "sleep 300 & echo $! > " + marker + "; sleep 301 & echo $! >> " + marker + "; echo started\n"
	if err := manager.Write(id, []byte(script)); err != nil {
		t.Fatal(err)
	}
	r.waitFor(t, "started\r\n")
	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Close(id); err != nil {
		t.Fatal(err)
	}
	for _, field := range strings.Fields(string(data)) {
		pid, _ := strconv.Atoi(field)
		deadline := time.Now().Add(5 * time.Second)
		for alive(pid) {
			if time.Now().After(deadline) {
				t.Fatalf("process %d outlived the terminal", pid)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	select {
	case <-r.code:
	case <-time.After(5 * time.Second):
		t.Fatal("no exit reported")
	}
}

func alive(pid int) bool {
	// A zombie still answers a signal, so ask ps what state it's in.
	output, err := exec.Command("ps", "-o", "stat=", "-p", strconv.Itoa(pid)).Output()
	return err == nil && len(output) > 0 && !strings.HasPrefix(strings.TrimSpace(string(output)), "Z")
}

func TestCloseAllAndRepoClose(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()
	manager := NewManager()
	first, _ := start(t, manager, dir)
	second, _ := start(t, manager, other)
	manager.CloseRepo(dir)
	if _, err := manager.get(first); err == nil {
		t.Fatal("the repository's shell should be gone")
	}
	if _, err := manager.get(second); err != nil {
		t.Fatal("another repository's shell should stay")
	}
	manager.CloseAll()
	if _, err := manager.get(second); err == nil {
		t.Fatal("CloseAll should end every shell")
	}
}

func TestResizeReachesTheShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses stty")
	}
	dir := t.TempDir()
	manager := NewManager()
	id, r := start(t, manager, dir)
	if err := manager.Resize(id, 123, 45); err != nil {
		t.Fatal(err)
	}
	if err := manager.Write(id, []byte("stty size\n")); err != nil {
		t.Fatal(err)
	}
	r.waitFor(t, "45 123")
}

func TestBackpressureDoesntLoseOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses seq")
	}
	dir := t.TempDir()
	manager := NewManager()
	var mu sync.Mutex
	total := 0
	var id string
	done := make(chan int, 1)
	var seen bytes.Buffer
	var err error
	id, err = manager.Start(dir, Options{
		Dir:  dir,
		Name: "/bin/sh",
		Args: []string{"-c", "seq 1 400000; echo finished"},
		Env:  Environment(os.Environ(), ""),
		Cols: 80, Rows: 24,
		OnOutput: func(data []byte) {
			mu.Lock()
			total += len(data)
			seen.Write(data)
			mu.Unlock()
			go manager.Ack(id, len(data))
		},
		OnExit: func(code int) { done <- code },
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("output stalled")
	}
	mu.Lock()
	defer mu.Unlock()
	if total < 2_000_000 || !strings.Contains(seen.String(), "399999") {
		t.Fatalf("read %d bytes", total)
	}
}

func TestHeldOutputWaitsForAttach(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/sh")
	}
	dir := t.TempDir()
	manager := NewManager()
	var mu sync.Mutex
	var seen bytes.Buffer
	id, err := manager.Start(dir, Options{
		Dir: dir, Name: "/bin/sh", Args: []string{"-c", "echo first; sleep 30"},
		Env: Environment(os.Environ(), ""), Cols: 80, Rows: 24, Hold: true,
		OnOutput: func(data []byte) {
			mu.Lock()
			seen.Write(data)
			mu.Unlock()
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close(id) })
	time.Sleep(200 * time.Millisecond)
	mu.Lock()
	early := seen.Len()
	mu.Unlock()
	if early != 0 {
		t.Fatalf("output before attach: %d bytes", early)
	}
	manager.Attach(id)
	deadline := time.Now().Add(5 * time.Second)
	for {
		mu.Lock()
		text := seen.String()
		mu.Unlock()
		if strings.Contains(text, "first") {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("no output after attach: %q", text)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestEnvironmentAddsGitAndATerminalType(t *testing.T) {
	sep := string(os.PathListSeparator)
	env := Environment([]string{"PATH=/usr/bin", "TERM=dumb", "HOME=/h"}, filepath.FromSlash("/opt/git/bin/git"))
	joined := strings.Join(env, "\n")
	if !strings.Contains(joined, "PATH="+filepath.FromSlash("/opt/git/bin")+sep+"/usr/bin") {
		t.Fatalf("env = %v", env)
	}
	if !strings.Contains(joined, "TERM=xterm-256color") || strings.Contains(joined, "TERM=dumb") || !strings.Contains(joined, "HOME=/h") {
		t.Fatalf("env = %v", env)
	}
	again := Environment([]string{"PATH=/opt/git/bin" + sep + "/usr/bin"}, "/opt/git/bin/git")
	if strings.Count(strings.Join(again, " "), "/opt/git/bin") != 1 {
		t.Fatalf("added twice: %v", again)
	}
}

func TestWindowsShellChoices(t *testing.T) {
	if name, _ := windowsShell("cmd", ""); name != "cmd.exe" {
		t.Fatalf("cmd = %q", name)
	}
	if name, args := windowsShell("", ""); !strings.HasSuffix(name, "powershell.exe") && !strings.HasSuffix(name, "pwsh.exe") && len(args) != 1 {
		t.Fatalf("default = %q %v", name, args)
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	bash := filepath.Join(root, "bin", "bash.exe")
	if err := os.WriteFile(bash, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := gitBashBinary(filepath.Join(root, "cmd", "git.exe")); got != bash {
		t.Fatalf("from cmd = %q", got)
	}
	if got := gitBashBinary(filepath.Join(root, "mingw64", "bin", "git.exe")); got != bash {
		t.Fatalf("from mingw64 = %q", got)
	}
	if name, args := windowsShell("git-bash", filepath.Join(root, "cmd", "git.exe")); name != bash || args[0] != "--login" {
		t.Fatalf("git bash = %q %v", name, args)
	}
}
