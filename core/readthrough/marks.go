package readthrough

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

var identity = []string{
	"GIT_AUTHOR_NAME=Current Client", "GIT_AUTHOR_EMAIL=read@localhost",
	"GIT_COMMITTER_NAME=Current Client", "GIT_COMMITTER_EMAIL=read@localhost",
}

// Marking reads the last mark and replaces it.
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

// Marks is what has been read under key, by path.
func Marks(ctx context.Context, repoPath, key string) (map[string]Mark, error) {
	result, err := gitexec.NewExecutor("").Run(ctx, gitexec.Command{
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
	for _, entry := range strings.Split(result.Stdout, "\x00") {
		header, path, ok := strings.Cut(entry, "\t")
		fields := strings.Fields(header)
		if !ok || len(fields) != 3 {
			continue
		}
		version := git.FileVersion{Mode: fields[0], Blob: fields[2]}
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

// IsRead says whether the file is as it was when read.
func IsRead(marks map[string]Mark, file File) bool {
	mark, marked := marks[file.Path]
	return marked && mark.Read == file.New
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
	dir, err := os.MkdirTemp("", "current-client-index")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	m := marker{ctx, repoPath, []string{"GIT_INDEX_FILE=" + filepath.Join(dir, "index")}}
	ref := markRef(key)
	if _, err := m.run("", "rev-parse", "--verify", "--quiet", ref); err == nil {
		if _, err := m.run("", "read-tree", ref); err != nil {
			return err
		}
	}
	var entries strings.Builder
	write := func(version git.FileVersion, path string) {
		if !read || version == (git.FileVersion{}) {
			version = git.FileVersion{Mode: "0", Blob: strings.Repeat("0", len(r.Head))}
		}
		fmt.Fprintf(&entries, "%s %s\t%s\x00", version.Mode, version.Blob, path)
	}
	for _, file := range files {
		write(file.New, readFolder+file.Path)
		write(file.Old, baseFolder+file.Path)
	}
	if _, err := m.run(entries.String(), "update-index", "-z", "--replace", "--index-info"); err != nil {
		return err
	}
	// Without its objects a partial clone would fetch them.
	tree, err := m.run("", "write-tree", "--missing-ok")
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

// keepUnchanged writes the working tree's content, which listing it doesn't, and
// leaves out the files that aren't as they were listed.
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

func (m marker) run(input string, args ...string) (string, error) {
	result, err := gitexec.NewExecutor("").RunChecked(m.ctx, gitexec.Command{
		Dir:   m.repoPath,
		Env:   append(append([]string{}, m.env...), identity...),
		Args:  args,
		Stdin: strings.NewReader(input),
	})
	return strings.TrimSpace(result.Stdout), err
}

// Forget removes the marks whose key is gone.
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

// Rename moves a mark to a renamed branch's key.
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
