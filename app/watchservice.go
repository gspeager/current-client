package app

import (
	"github.com/gspeager/current-client/internal/watch"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const filesChangedEvent = "repo:files-changed"

func init() {
	application.RegisterEvent[FilesChangedEvent](filesChangedEvent)
}

type WatchService struct {
	watcher watch.Watcher
}

// FilesChangedEvent.RefChanged is false for plain working-tree edits, so the
// frontend can skip reloading history.
type FilesChangedEvent struct {
	RepoPath   string `json:"repoPath"`
	RefChanged bool   `json:"refChanged"`
}

// Start replaces any previous watch; only one repository is watched at a time.
func (s *WatchService) Start(repoPath string) error {
	return s.watcher.Start(repoPath, func(refChanged bool) {
		application.Get().Event.Emit(filesChangedEvent, FilesChangedEvent{RepoPath: repoPath, RefChanged: refChanged})
	})
}

func (s *WatchService) Stop() {
	s.watcher.Stop()
}
