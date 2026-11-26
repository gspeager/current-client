package readthrough

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
	"github.com/gspeager/current-client/core/snapshot"
)

// A mark is a commit whose tree holds each read file twice: under read/ as it
// was read, and under base/ as it was in what it was compared with. A file added
// or deleted has only one. Keeping both lets a later version be compared with
// what was read, and tells a new base's changes from the branch's own. The
// commit's parent is the head at the last mark, so a rebase doesn't lose it.
//
// Refs are named by a digest, since a key is a ref's full name or a worktree's
// path and one can hold another's folder. The message is the key.
const (
	markPrefix = "refs/current-client/read/"
	readFolder = "read/"
	baseFolder = "base/"
)

var identity = gitexec.Identity("read@localhost")

// marking serialises writes, as each one reads the last mark and replaces it.
var marking sync.Mutex

func markRef(key string) string {
	digest := sha256.Sum256([]byte(key))
	return markPrefix + hex.EncodeToString(digest[:20])
}

// Mark is a file's content when it was read, and its base then.
type Mark struct {
	Read git.FileVersion
	Base git.FileVersion
}

func Marks(ctx context.Context, repoPath, key string) (map[string]Mark, error) {
	result, err := gitexec.Run(ctx, gitexec.Command{
		Dir:  repoPath,
		Args: []string{"ls-tree", "-r", "-z", "--full-tree", markRef(key)},
	})
	if err != nil {
		return nil, gitexec.WrapRunError(err)
	}
	marks := map[string]Mark{}
	if result.ExitCode != 0 {
		return marks, nil
	}
	for path, version := range parseVersions(result.Stdout) {
		if name, ok := strings.CutPrefix(path, readFolder); ok {
			mark := marks[name]
			mark.Read = version
			marks[name] = mark
		} else if name, ok := strings.CutPrefix(path, baseFolder); ok {
			mark := marks[name]
			mark.Base = version
			marks[name] = mark
		}
	}
	return marks, nil
}

// SetRead marks files as read, or as not. They're the versions the reader saw: one
// that has changed since in the working tree isn't marked.
func SetRead(ctx context.Context, repoPath, key string, r Range, files []File, read bool) error {
	marking.Lock()
	defer marking.Unlock()
	if read && r.Uncommitted {
		unchanged, err := keepUnchanged(ctx, repoPath, r, files)
		if err != nil {
			return err
		}
		files = unchanged
	}
	ref := markRef(key)
	entries := map[string]git.FileVersion{}
	for _, file := range files {
		// A renamed file may be marked under the name it had.
		if file.OrigPath != "" {
			entries[readFolder+file.OrigPath], entries[baseFolder+file.OrigPath] = git.FileVersion{}, git.FileVersion{}
		}
	}
	for _, file := range files {
		entries[readFolder+file.Path], entries[baseFolder+file.Path] = git.FileVersion{}, git.FileVersion{}
		if read {
			entries[readFolder+file.Path], entries[baseFolder+file.Path] = file.New, file.Old
		}
	}
	m := marker{ctx, repoPath, nil}
	tree, err := m.writeTree(ref, entries, len(r.Head))
	if err != nil {
		return err
	}
	commit, err := m.run("", "commit-tree", tree, "-p", r.Head, "-m", key)
	if err != nil {
		return err
	}
	_, err = m.run("", "update-ref", ref, commit)
	return err
}

// keepUnchanged writes the working tree's blobs, which listing the files doesn't,
// and leaves out the files that have changed since they were listed.
func keepUnchanged(ctx context.Context, repoPath string, r Range, files []File) ([]File, error) {
	tree, _, err := snapshot.WorkingTree(ctx, repoPath, nil)
	if err != nil {
		return nil, err
	}
	changes, err := git.TreeChanges(ctx, repoPath, nil, r.Base, tree)
	if err != nil {
		return nil, err
	}
	now := map[string]git.FileVersion{}
	for _, change := range changes {
		now[change.Path] = change.New
	}
	return slices.DeleteFunc(slices.Clone(files), func(file File) bool {
		version, listed := now[file.Path]
		return !listed || version != file.New
	}), nil
}

type marker struct {
	ctx      context.Context
	repoPath string
	env      []string
}

// writeTree writes a tree of entries on top of what from holds, if it exists. An
// entry with no version is taken out.
func (m marker) writeTree(from string, entries map[string]git.FileVersion, idLength int) (string, error) {
	dir, err := os.MkdirTemp("", "current-client-index")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	index := marker{m.ctx, m.repoPath, append([]string{"GIT_INDEX_FILE=" + filepath.Join(dir, "index")}, m.env...)}
	if from != "" {
		if _, err := index.run("", "rev-parse", "--verify", "--quiet", from); err == nil {
			if _, err := index.run("", "read-tree", from); err != nil {
				return "", err
			}
		}
	}
	var input strings.Builder
	for _, path := range slices.Sorted(maps.Keys(entries)) {
		version := entries[path]
		if version == (git.FileVersion{}) {
			version = git.FileVersion{Mode: "0", Blob: strings.Repeat("0", idLength)}
		}
		fmt.Fprintf(&input, "%s %s\t%s\x00", version.Mode, version.Blob, path)
	}
	if _, err := index.run(input.String(), "update-index", "-z", "--replace", "--index-info"); err != nil {
		return "", err
	}
	// Without its objects a partial clone would fetch them.
	return index.run("", "write-tree", "--missing-ok")
}

func (m marker) run(input string, args ...string) (string, error) {
	result, err := gitexec.RunChecked(m.ctx, gitexec.Command{
		Dir:   m.repoPath,
		Env:   append(append([]string{}, m.env...), identity...),
		Args:  args,
		Stdin: strings.NewReader(input),
	})
	return strings.TrimSpace(result.Stdout), err
}

func Forget(ctx context.Context, repoPath string, gone func(key string) bool) error {
	marks, err := allMarks(ctx, repoPath)
	if err != nil {
		return err
	}
	for ref, key := range marks {
		if gone(key) {
			if _, err := gitexec.RunIn(ctx, repoPath, "update-ref", "-d", ref); err != nil {
				return err
			}
		}
	}
	return nil
}

func allMarks(ctx context.Context, repoPath string) (map[string]string, error) {
	result, err := gitexec.RunIn(ctx, repoPath, "for-each-ref", "--format=%(refname)%00%(contents:subject)", markPrefix)
	if err != nil {
		return nil, err
	}
	marks := map[string]string{}
	for _, line := range strings.Split(result.Stdout, "\n") {
		if ref, key, ok := strings.Cut(line, "\x00"); ok {
			marks[ref] = key
		}
	}
	return marks, nil
}

func Rename(ctx context.Context, repoPath, oldKey, newKey string) error {
	marking.Lock()
	defer marking.Unlock()
	marker := marker{ctx: ctx, repoPath: repoPath}
	old, err := marker.run("", "rev-parse", "--verify", "--quiet", markRef(oldKey))
	if err != nil {
		return nil
	}
	commit, err := marker.run("", "commit-tree", old+"^{tree}", "-p", old+"^", "-m", newKey)
	if err != nil {
		return err
	}
	if _, err := marker.run("", "update-ref", markRef(newKey), commit); err != nil {
		return err
	}
	_, err = marker.run("", "update-ref", "-d", markRef(oldKey))
	return err
}
