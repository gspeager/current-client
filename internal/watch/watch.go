// Package watch reports working-tree and ref changes made outside the app.
package watch

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gspeager/current-client/core/git"
)

const debounceDelay = 400 * time.Millisecond

// Watcher watches one repository at a time. The zero value is ready to use.
type Watcher struct {
	mu     sync.Mutex
	fw     *fsnotify.Watcher
	stopCh chan struct{}
}

// gitDirs are the repository's .git folders. In a worktree, HEAD lives in the
// worktree's own git dir and refs in the common one.
type gitDirs struct {
	gitDir    string
	commonDir string
}

// Start replaces any previous watch. onChange is debounced; refChanged reports
// whether HEAD, a branch, a tag, a remote branch or the stash moved, as
// opposed to only files being edited.
func (w *Watcher) Start(repoPath string, onChange func(refChanged bool)) error {
	w.Stop()

	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	dirs := findGitDirs(repoPath)
	paths, err := watchPaths(repoPath, dirs)
	if err != nil {
		_ = fw.Close()
		return err
	}
	for path := range paths {
		_ = fw.Add(path)
	}

	stopCh := make(chan struct{})
	w.mu.Lock()
	w.fw = fw
	w.stopCh = stopCh
	w.mu.Unlock()

	go run(fw, stopCh, repoPath, dirs, onChange)
	return nil
}

func (w *Watcher) Stop() {
	w.mu.Lock()
	fw, stopCh := w.fw, w.stopCh
	w.fw, w.stopCh = nil, nil
	w.mu.Unlock()

	if stopCh != nil {
		close(stopCh)
	}
	if fw != nil {
		_ = fw.Close()
	}
}

func run(fw *fsnotify.Watcher, stopCh chan struct{}, repoPath string, dirs gitDirs, onChange func(refChanged bool)) {
	var timer *time.Timer
	var refChanged, newDir atomic.Bool
	for {
		select {
		case <-stopCh:
			if timer != nil {
				timer.Stop()
			}
			return
		case event, ok := <-fw.Events:
			if !ok {
				return
			}
			ref, inGitDir := dirs.classify(event.Name)
			if inGitDir && !ref {
				continue
			}
			if ref {
				refChanged.Store(true)
			}
			if event.Has(fsnotify.Create) {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					newDir.Store(true)
				}
			}
			if timer != nil {
				timer.Stop()
			}
			// New folders are watched before onChange, so a change right after it isn't missed.
			timer = time.AfterFunc(debounceDelay, func() {
				if newDir.Swap(false) {
					if paths, err := watchPaths(repoPath, dirs); err == nil {
						for path := range paths {
							_ = fw.Add(path)
						}
					}
				}
				onChange(refChanged.Swap(false))
			})
		case _, ok := <-fw.Errors:
			if !ok {
				return
			}
		}
	}
}

func findGitDirs(repoPath string) gitDirs {
	ctx := context.Background()
	gitDir, err := git.GitDir(ctx, repoPath)
	if err != nil {
		return gitDirs{}
	}
	commonDir := gitDir
	if common, err := git.CommonDir(ctx, repoPath); err == nil {
		commonDir = common
	}
	return gitDirs{gitDir: filepath.Clean(gitDir), commonDir: filepath.Clean(commonDir)}
}

// classify says whether a path is in a git folder, and if so whether it's a
// ref: HEAD, its reflog, packed-refs or anything under refs/. Everything else
// there, the index above all, is ignored; status rewrites the index, so
// reacting to it would refresh forever.
func (d gitDirs) classify(path string) (ref, inGitDir bool) {
	if d.gitDir == "" {
		return false, false
	}
	path = filepath.Clean(path)
	switch path {
	case filepath.Join(d.gitDir, "HEAD"),
		filepath.Join(d.gitDir, "logs", "HEAD"),
		filepath.Join(d.commonDir, "packed-refs"):
		return true, true
	}
	if within(path, filepath.Join(d.commonDir, "refs")) {
		return !strings.HasSuffix(path, ".lock"), true
	}
	return false, within(path, d.gitDir) || within(path, d.commonDir)
}

func within(path, dir string) bool {
	return path == dir || strings.HasPrefix(path, dir+string(filepath.Separator))
}

// watchPaths watches folders rather than files: git replaces HEAD and refs by
// renaming a lock file over them, which drops a watch on the file itself.
func watchPaths(repoPath string, dirs gitDirs) (map[string]bool, error) {
	trackedPaths, err := git.WatchedPaths(context.Background(), repoPath)
	if err != nil {
		return nil, err
	}

	paths := map[string]bool{repoPath: true}
	for _, p := range trackedPaths {
		paths[filepath.Join(repoPath, filepath.Dir(p))] = true
	}
	if dirs.gitDir != "" {
		paths[dirs.gitDir] = true
		paths[filepath.Join(dirs.gitDir, "logs")] = true
		paths[dirs.commonDir] = true
		_ = filepath.WalkDir(filepath.Join(dirs.commonDir, "refs"), func(path string, entry fs.DirEntry, err error) error {
			if err == nil && entry.IsDir() {
				paths[path] = true
			}
			return nil
		})
	}
	return paths, nil
}
