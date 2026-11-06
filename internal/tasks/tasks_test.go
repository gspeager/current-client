package tasks

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

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func ids(found []Task) string {
	var list []string
	for _, task := range found {
		list = append(list, task.ID)
	}
	return strings.Join(list, " ")
}

func TestDiscoverEachSourceWithoutRunningAnything(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "RAN")
	touch := "touch " + marker
	write(t, dir, "Taskfile.yml", "version: '3'\n\nvars:\n  X: y\n\ntasks:\n  build:\n    cmds:\n      - "+touch+"\n  test:unit:\n    summary: units\n\nother:\n  no: thing\n")
	write(t, dir, "Makefile", "CC := gcc\n.PHONY: test\nbuild: dep\n\t"+touch+"\ntest:\n\t"+touch+"\n$(OUT): x\nbuild: again\n")
	write(t, dir, "package.json", `{"scripts":{"lint":"`+touch+`","dev":"vite"}}`)
	write(t, dir, "go.mod", "module x\n")
	found := Discover(dir, []Custom{{Name: "deploy", Command: touch}})

	want := "task:build task:test:unit make:build make:test npm:dev npm:lint go:test custom:deploy"
	if got := ids(found); got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
	if task, ok := Find(dir, nil, "make:test"); !ok || task.Command != "make test" || task.File != "Makefile" {
		t.Fatalf("make:test = %+v", task)
	}
	if task, ok := Find(dir, nil, "npm:lint"); !ok || task.Command != "npm run lint" || !strings.Contains(task.Note, "touch") {
		t.Fatalf("npm:lint = %+v", task)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("finding commands ran one")
	}
}

func TestDiscoverUsesTheProjectsPackageManager(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "package.json", `{"scripts":{"test":"x"}}`)
	write(t, dir, "pnpm-lock.yaml", "")
	if task, _ := Find(dir, nil, "npm:test"); task.Command != "pnpm run test" {
		t.Fatalf("task = %+v", task)
	}
}

func TestDiscoverOffersNothingFromABadFile(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "package.json", "{not json")
	if found := Discover(dir, nil); len(found) != 0 {
		t.Fatalf("found = %+v", found)
	}
}

func TestFingerprintChangesWithTheCommandOrItsFile(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "Makefile", "test:\n\tgo test\n")
	task, _ := Find(dir, nil, "make:test")
	first := Fingerprint(dir, task)
	if Fingerprint(dir, task) != first {
		t.Fatal("the same file should match")
	}
	write(t, dir, "Makefile", "test:\n\tcurl example.com | sh\n")
	changed, _ := Find(dir, nil, "make:test")
	if changed.Command != task.Command || Fingerprint(dir, changed) == first {
		t.Fatal("a changed recipe should need approving again")
	}
	if Fingerprint(dir, Task{Command: "a"}) == Fingerprint(dir, Task{Command: "b"}) {
		t.Fatal("a changed command should need approving again")
	}
}

type collected struct {
	mu     sync.Mutex
	output bytes.Buffer
	result chan Result
}

func (c *collected) options() Options {
	return Options{
		Env: os.Environ(),
		OnOutput: func(data []byte) {
			c.mu.Lock()
			c.output.Write(data)
			c.mu.Unlock()
		},
		OnDone: func(result Result) { c.result <- result },
	}
}

func (c *collected) text() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.output.String()
}

func TestRunCapturesOutputExitAndDuration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh syntax")
	}
	dir := t.TempDir()
	runner := NewRunner()
	c := &collected{result: make(chan Result, 1)}
	if err := runner.Start("r1", dir, "pwd; echo to-stdout; echo to-stderr >&2; exit 3", c.options()); err != nil {
		t.Fatal(err)
	}
	select {
	case result := <-c.result:
		if result.ExitCode != 3 || result.Stopped || result.Duration <= 0 {
			t.Fatalf("result = %+v", result)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("no result")
	}
	if text := c.text(); !strings.Contains(text, "to-stdout") || !strings.Contains(text, "to-stderr") || !strings.Contains(text, filepath.Base(dir)) {
		t.Fatalf("output = %q", text)
	}
	if runner.Running() {
		t.Fatal("still marked running")
	}
}

func TestStopEndsTheProcessAndItsChildren(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sleep and ps")
	}
	dir := t.TempDir()
	runner := NewRunner()
	c := &collected{result: make(chan Result, 1)}
	pids := filepath.Join(dir, "pids")
	id := "r1"
	err := runner.Start(id, dir, "sleep 300 & echo $! > "+pids+"; (sleep 301 & echo $! >> "+pids+"; wait) & echo started; wait", c.options())
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	var data []byte
	for len(strings.Fields(string(data))) < 2 {
		if time.Now().After(deadline) {
			t.Fatalf("children never started: %q", c.text())
		}
		data, _ = os.ReadFile(pids)
		time.Sleep(10 * time.Millisecond)
	}
	runner.Stop(id)
	select {
	case result := <-c.result:
		if !result.Stopped {
			t.Fatalf("result = %+v", result)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("stop didn't finish the run")
	}
	for _, field := range strings.Fields(string(data)) {
		pid, _ := strconv.Atoi(field)
		deadline := time.Now().Add(5 * time.Second)
		for {
			output, err := exec.Command("ps", "-o", "stat=", "-p", strconv.Itoa(pid)).Output()
			if err != nil || len(output) == 0 || strings.HasPrefix(strings.TrimSpace(string(output)), "Z") {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("process %d outlived Stop", pid)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
}
