package watch

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gspeager/current-client/internal/gittest"
)

func initWatchRepo(t *testing.T) string {
	t.Helper()
	dir := gittest.InitRepo(t)
	gittest.WriteFile(t, dir, "tracked.txt", "v1")
	gittest.WriteFile(t, dir, ".gitignore", "ignored/\n")
	gittest.Run(t, dir, "add", "tracked.txt", ".gitignore")
	gittest.Run(t, dir, "commit", "-m", "initial")
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

func TestWatcherIgnoresReadsOfAModifiedFile(t *testing.T) {
	dir := initWatchRepo(t)
	var w Watcher
	changed := startRecording(t, &w, dir)
	defer w.Stop()

	path := filepath.Join(dir, "tracked.txt")
	if err := os.WriteFile(path, []byte("v2"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if fired, _ := waitForChange(t, changed, 5*time.Second); !fired {
		t.Fatal("onChange did not fire after editing a tracked file")
	}

	// git status and diff read the modified file, which only moves its access time.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	for i := range 3 {
		if err := os.Chtimes(path, time.Now().Add(time.Duration(i+1)*time.Hour), info.ModTime()); err != nil {
			t.Fatalf("Chtimes: %v", err)
		}
		if fired, _ := waitForChange(t, changed, time.Second); fired {
			t.Fatalf("read %d: onChange fired for an access-time change, want it ignored", i+1)
		}
	}

	if runtime.GOOS == "windows" {
		return
	}
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	if fired, _ := waitForChange(t, changed, 5*time.Second); !fired {
		t.Fatal("onChange did not fire after making a tracked file executable")
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

	gittest.Run(t, dir, "commit", "--allow-empty", "-m", "external commit")

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
		gittest.Run(t, dir, args...)
		fired, refChanged := waitForChange(t, changed, 5*time.Second)
		if !fired || !refChanged {
			t.Fatalf("git %v: fired=%v refChanged=%v, want a ref change", args, fired, refChanged)
		}
	}

	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	gittest.Run(t, dir, "stash")
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
		gittest.Run(t, dir, "commit", "--allow-empty", "-m", "external commit")
		if fired, refChanged := waitForChange(t, changed, 5*time.Second); !fired || !refChanged {
			t.Fatalf("commit %d: fired=%v refChanged=%v, want every commit reported", i+1, fired, refChanged)
		}
	}
}

func TestWatcherInAWorktreeSeesBranchesMadeInTheMainRepository(t *testing.T) {
	dir := initWatchRepo(t)
	worktree := filepath.Join(t.TempDir(), "wt")
	gittest.Run(t, dir, "worktree", "add", "-q", "-b", "side", worktree)
	var w Watcher
	changed := startRecording(t, &w, worktree)
	defer w.Stop()

	gittest.Run(t, dir, "branch", "made-elsewhere")

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
