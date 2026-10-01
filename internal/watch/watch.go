// Package watch reports working-tree and ref changes made outside the app.
package watch

import (
	"context"
	"path/filepath"
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

// Start replaces any previous watch. onChange is debounced; refChanged reports
// whether HEAD or its reflog moved, as opposed to only files being edited.
func (w *Watcher) Start(repoPath string, onChange func(refChanged bool)) error {
	w.Stop()

	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	paths, err := watchPaths(repoPath)
	if err != nil {
		_ = fw.Close()
		return err
	}
	for path := range paths {
		_ = fw.Add(path)
	}

	headPath, reflogPath := refPaths(repoPath)

	stopCh := make(chan struct{})
	w.mu.Lock()
	w.fw = fw
	w.stopCh = stopCh
	w.mu.Unlock()

	go run(fw, stopCh, repoPath, headPath, reflogPath, onChange)
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

func run(fw *fsnotify.Watcher, stopCh chan struct{}, repoPath, headPath, reflogPath string, onChange func(refChanged bool)) {
	var timer *time.Timer
	var refChanged atomic.Bool
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
			if event.Name == headPath || event.Name == reflogPath {
				refChanged.Store(true)
			}
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(debounceDelay, func() {
				onChange(refChanged.Swap(false))
				// Pick up directories created since the last change.
				if paths, err := watchPaths(repoPath); err == nil {
					for path := range paths {
						_ = fw.Add(path)
					}
				}
			})
		case _, ok := <-fw.Errors:
			if !ok {
				return
			}
		}
	}
}

func refPaths(repoPath string) (headPath, reflogPath string) {
	gitDir, err := git.GitDir(context.Background(), repoPath)
	if err != nil {
		return "", ""
	}
	return filepath.Join(gitDir, "HEAD"), filepath.Join(gitDir, "logs", "HEAD")
}

// watchPaths watches HEAD and logs/HEAD rather than all of .git: status
// rewrites .git/index, which would retrigger onChange in an endless loop.
func watchPaths(repoPath string) (map[string]bool, error) {
	ctx := context.Background()
	trackedPaths, err := git.WatchedPaths(ctx, repoPath)
	if err != nil {
		return nil, err
	}

	paths := map[string]bool{repoPath: true}
	if headPath, reflogPath := refPaths(repoPath); headPath != "" {
		paths[headPath] = true
		paths[reflogPath] = true
	}
	for _, p := range trackedPaths {
		paths[filepath.Join(repoPath, filepath.Dir(p))] = true
	}
	return paths, nil
}
