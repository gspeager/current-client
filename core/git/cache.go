package git

import (
	"context"
	"fmt"
	"hash/fnv"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/sync/singleflight"
)

// flights makes callers that ask the same question at the same time share
// one git process.
var flights singleflight.Group

type cacheEntry struct {
	stamp uint64
	value any
}

var cache = struct {
	sync.Mutex
	entries map[string]cacheEntry
}{entries: map[string]cacheEntry{}}

// cachedByRefs keeps read's answer until a ref, HEAD, the config or one of
// the also files changes, whoever changes it, so callers must treat a
// returned slice as read-only. Without a stamp (not a repository yet) every
// call reads.
func cachedByRefs[T any](ctx context.Context, repoPath, name string, read func() (T, error), also ...string) (T, error) {
	stamp, ok := refsStamp(ctx, repoPath, also)
	if !ok {
		return read()
	}
	key := name + "\x00" + repoPath
	cache.Lock()
	entry, found := cache.entries[key]
	cache.Unlock()
	if found && entry.stamp == stamp {
		return entry.value.(T), nil
	}
	value, err, _ := flights.Do(fmt.Sprintf("%s\x00%d", key, stamp), func() (any, error) {
		value, err := read()
		if err == nil {
			cache.Lock()
			cache.entries[key] = cacheEntry{stamp: stamp, value: value}
			cache.Unlock()
		}
		return value, err
	})
	if err != nil {
		var zero T
		return zero, err
	}
	return value.(T), nil
}

// shared runs read once for callers that ask at the same time, without
// keeping the answer, for data that changes without a ref moving.
func shared[T any](repoPath, name string, read func() (T, error)) (T, error) {
	value, err, _ := flights.Do(name+"\x00"+repoPath, func() (any, error) { return read() })
	if err != nil {
		var zero T
		return zero, err
	}
	return value.(T), nil
}

// refsStamp fingerprints the files git changes when a ref moves: HEAD, loose
// and packed refs, their reflogs, the config (upstreams) and each worktree's
// HEAD. Sizes as well as times, since reflogs only grow and a coarse
// filesystem clock could give two updates the same time.
func refsStamp(ctx context.Context, repoPath string, also []string) (uint64, bool) {
	dirs, err := gitDirs(ctx, repoPath)
	if err != nil {
		return 0, false
	}
	gitDir, commonDir := dirs.gitDir, dirs.commonDir
	h := fnv.New64a()
	add := func(path string, info fs.FileInfo) {
		fmt.Fprintf(h, "%s\x00%d\x00%d\n", path, info.Size(), info.ModTime().UnixNano())
	}
	for _, path := range append([]string{
		filepath.Join(gitDir, "HEAD"),
		filepath.Join(gitDir, "logs", "HEAD"),
		filepath.Join(commonDir, "packed-refs"),
		filepath.Join(commonDir, "config"),
	}, also...) {
		if info, err := os.Stat(path); err == nil {
			add(path, info)
		}
	}
	worktreeHeads, _ := filepath.Glob(filepath.Join(commonDir, "worktrees", "*", "HEAD"))
	for _, path := range worktreeHeads {
		if info, err := os.Stat(path); err == nil {
			add(path, info)
		}
	}
	for _, dir := range []string{"refs", "reftable", filepath.Join("logs", "refs")} {
		_ = filepath.WalkDir(filepath.Join(commonDir, dir), func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}
			if info, err := entry.Info(); err == nil {
				add(path, info)
			}
			return nil
		})
	}
	return h.Sum64(), true
}
