package watch

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func runGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func initWatchRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGitCommand(t, dir, "init", "-b", "main")
	runGitCommand(t, dir, "config", "user.email", "test@example.com")
	runGitCommand(t, dir, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("v1"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("ignored/\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	runGitCommand(t, dir, "add", "tracked.txt", ".gitignore")
	runGitCommand(t, dir, "commit", "-m", "initial")
	if err := os.MkdirAll(filepath.Join(dir, "ignored"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	return dir
}

func startRecording(t *testing.T, w *Watcher, dir string) <-chan bool {
	t.Helper()
	changed := make(chan bool, 8)
	if err := w.Start(dir, func(refChanged bool) {
		select {
		case changed <- refChanged:
		default:
		}
	}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	return changed
}

func waitForChange(t *testing.T, changed <-chan bool, timeout time.Duration) (bool, bool) {
	t.Helper()
	select {
	case refChanged := <-changed:
		return true, refChanged
	case <-time.After(timeout):
		return false, false
	}
}

func TestWatcherFiresOnTrackedFileEdit(t *testing.T) {
	dir := initWatchRepo(t)
	var w Watcher
	changed := startRecording(t, &w, dir)
	defer w.Stop()

	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("v2"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	fired, refChanged := waitForChange(t, changed, 5*time.Second)
	if !fired {
		t.Fatal("onChange did not fire after editing a tracked file")
	}
	if refChanged {
		t.Error("refChanged = true for an ordinary tracked-file edit, want false")
	}
}

func TestWatcherFiresOnNewUntrackedFile(t *testing.T) {
	dir := initWatchRepo(t)
	var w Watcher
	changed := startRecording(t, &w, dir)
	defer w.Stop()

	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("v1"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if fired, _ := waitForChange(t, changed, 5*time.Second); !fired {
		t.Fatal("onChange did not fire after creating a new untracked file")
	}
}

func TestWatcherIgnoresGitignoredDirectory(t *testing.T) {
	dir := initWatchRepo(t)
	var w Watcher
	changed := startRecording(t, &w, dir)
	defer w.Stop()

	if err := os.WriteFile(filepath.Join(dir, "ignored", "file.txt"), []byte("v1"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if fired, _ := waitForChange(t, changed, 2*time.Second); fired {
		t.Fatal("onChange fired for a change inside a gitignored directory, want it never watched")
	}
}

func TestWatcherIgnoresIndexChurn(t *testing.T) {
	dir := initWatchRepo(t)
	var w Watcher
	changed := startRecording(t, &w, dir)
	defer w.Stop()

	// status rewrites .git/index as a side effect; watching it caused an endless
	// refresh loop.
	indexPath := filepath.Join(dir, ".git", "index")
	if err := os.WriteFile(indexPath, []byte("not a real index, just churn"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if fired, _ := waitForChange(t, changed, 2*time.Second); fired {
		t.Fatal("onChange fired for a .git/index write, want index churn never watched")
	}
}

func TestWatcherFiresOnExternalCommit(t *testing.T) {
	dir := initWatchRepo(t)
	var w Watcher
	changed := startRecording(t, &w, dir)
	defer w.Stop()

	runGitCommand(t, dir, "commit", "--allow-empty", "-m", "external commit")

	fired, refChanged := waitForChange(t, changed, 5*time.Second)
	if !fired {
		t.Fatal("onChange did not fire after an external commit (logs/HEAD)")
	}
	if !refChanged {
		t.Error("refChanged = false for an external commit, want true")
	}
}

func TestWatcherReportsRefChangesMadeOutsideTheApp(t *testing.T) {
	dir := initWatchRepo(t)
	var w Watcher
	changed := startRecording(t, &w, dir)
	defer w.Stop()

	for _, args := range [][]string{
		{"branch", "spike"},
		{"tag", "v1.0"},
		// A new folder under refs/heads, then a second branch inside it.
		{"branch", "feature/one"},
		{"branch", "feature/two"},
		{"pack-refs", "--all"},
	} {
		runGitCommand(t, dir, args...)
		fired, refChanged := waitForChange(t, changed, 5*time.Second)
		if !fired || !refChanged {
			t.Fatalf("git %v: fired=%v refChanged=%v, want a ref change", args, fired, refChanged)
		}
	}

	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitCommand(t, dir, "stash")
	if fired, refChanged := waitForChange(t, changed, 5*time.Second); !fired || !refChanged {
		t.Fatalf("git stash: fired=%v refChanged=%v, want a ref change", fired, refChanged)
	}
}

func TestWatcherKeepsWatchingHEADAfterGitReplacesIt(t *testing.T) {
	dir := initWatchRepo(t)
	var w Watcher
	changed := startRecording(t, &w, dir)
	defer w.Stop()

	for i := range 3 {
		runGitCommand(t, dir, "commit", "--allow-empty", "-m", "external commit")
		if fired, refChanged := waitForChange(t, changed, 5*time.Second); !fired || !refChanged {
			t.Fatalf("commit %d: fired=%v refChanged=%v, want every commit reported", i+1, fired, refChanged)
		}
	}
}

func TestWatcherInAWorktreeSeesBranchesMadeInTheMainRepository(t *testing.T) {
	dir := initWatchRepo(t)
	worktree := filepath.Join(t.TempDir(), "wt")
	runGitCommand(t, dir, "worktree", "add", "-q", "-b", "side", worktree)
	var w Watcher
	changed := startRecording(t, &w, worktree)
	defer w.Stop()

	runGitCommand(t, dir, "branch", "made-elsewhere")

	if fired, refChanged := waitForChange(t, changed, 5*time.Second); !fired || !refChanged {
		t.Fatalf("fired=%v refChanged=%v, want the shared refs watched", fired, refChanged)
	}
}

func TestWatcherStopPreventsFurtherCallbacks(t *testing.T) {
	dir := initWatchRepo(t)
	var w Watcher
	changed := startRecording(t, &w, dir)
	w.Stop()

	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("v2"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if fired, _ := waitForChange(t, changed, 2*time.Second); fired {
		t.Fatal("onChange fired after Stop")
	}
}
