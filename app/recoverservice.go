package app

import (
	"context"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/snapshot"
)

type RecoverService struct{}

type SnapshotInfo struct {
	ID      string   `json:"id"`
	Kind    string   `json:"kind"`
	Tip     string   `json:"tip"`
	Date    string   `json:"date"`
	Action  string   `json:"action"`
	Branch  string   `json:"branch"`
	Skipped int      `json:"skipped"`
	Files   []string `json:"files"`
}

func snapshotLimit() int {
	return configOrDefault().SnapshotLimit
}

// snapshotBefore never stands in the way of the action: if the snapshot fails,
// the action still runs, as it did before snapshots existed.
func snapshotBefore(repoPath, action string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_, _ = snapshot.Take(ctx, repoPath, action, snapshotLimit())
}

func snapshotTip(repoPath, action, branch, tip string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_ = snapshot.TakeTip(ctx, repoPath, action, branch, tip, snapshotLimit())
}

func (s *RecoverService) List(repoPath string) ([]SnapshotInfo, error) {
	ctx := context.Background()
	snapshots, err := snapshot.List(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	info := []SnapshotInfo{}
	for _, item := range snapshots {
		entry := SnapshotInfo{
			ID:      item.ID,
			Kind:    item.Kind,
			Date:    item.At.Format(time.RFC3339),
			Action:  item.Action,
			Branch:  item.Branch,
			Skipped: item.Skipped,
			Files:   []string{},
		}
		if item.Kind == snapshot.KindTip {
			entry.Tip = item.Parent
		} else {
			files, err := git.ChangedFiles(ctx, repoPath, item.SHA)
			if err != nil {
				return nil, err
			}
			entry.Files = mapSlice(files, func(file git.ChangedFile) string { return file.Path })
		}
		info = append(info, entry)
	}
	return info, nil
}

// Restore puts back the snapshot's files, or only paths when given.
func (s *RecoverService) Restore(repoPath, id string, paths []string) error {
	return snapshot.Restore(context.Background(), repoPath, id, paths, snapshotLimit())
}

func (s *RecoverService) Delete(repoPath, id string) error {
	return snapshot.Delete(context.Background(), repoPath, id)
}
