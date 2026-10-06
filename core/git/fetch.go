package git

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

func Fetch(ctx context.Context, repoPath, remote string) error {
	_, err := runResult(ctx, repoPath, "fetch", remote)
	return err
}

func FetchAll(ctx context.Context, repoPath string) error {
	_, err := runResult(ctx, repoPath, "fetch", "--all")
	return err
}

// FetchAllPrune is FetchAll that also deletes remote-tracking branches whose
// branch no longer exists on the remote.
func FetchAllPrune(ctx context.Context, repoPath string) error {
	_, err := runResult(ctx, repoPath, "fetch", "--all", "--prune")
	return err
}

// LastFetchTime reads FETCH_HEAD's mtime; nil means never fetched.
func LastFetchTime(ctx context.Context, repoPath string) (*time.Time, error) {
	dir, err := GitDir(ctx, repoPath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(filepath.Join(dir, "FETCH_HEAD"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	modTime := info.ModTime()
	return &modTime, nil
}
