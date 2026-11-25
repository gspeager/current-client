package app

import (
	"cmp"
	"context"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/gspeager/current-client/core/diff"
	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
	"github.com/gspeager/current-client/core/readthrough"
	"github.com/gspeager/current-client/internal/notes"
)

type ReadThroughService struct{}

const worktreeKey = "worktree:"

// ReadThroughRequest is a ref by its full name, or with Path the worktree there.
// Target is what it's compared with, the parent of a stacked branch or the
// default branch when it's "". Uncommitted asks for uncommitted changes too,
// where the ref is checked out.
type ReadThroughRequest struct {
	Ref         string `json:"ref"`
	Path        string `json:"path"`
	Target      string `json:"target"`
	Uncommitted bool   `json:"uncommitted"`
}

// ReadThroughInfo.Key names what's read, for its marks and notes. Path is the
// working tree its commands run in, and CheckedOut whether the ref is what's
// checked out there.
type ReadThroughInfo struct {
	Key         string            `json:"key"`
	Name        string            `json:"name"`
	Path        string            `json:"path"`
	Target      string            `json:"target"`
	Base        string            `json:"base"`
	Head        string            `json:"head"`
	CheckedOut  bool              `json:"checkedOut"`
	Uncommitted bool              `json:"uncommitted"`
	Files       []ReadThroughFile `json:"files"`
	Skipped     int               `json:"skipped"`
}

type ReadThroughFile struct {
	Status   string `json:"status"`
	Path     string `json:"path"`
	OrigPath string `json:"origPath"`
	Added    int    `json:"added"`
	Removed  int    `json:"removed"`
	Binary   bool   `json:"binary"`
	// "", "test", "generated" or "lockfile".
	Kind    string `json:"kind"`
	Read    bool   `json:"read"`
	OldMode string `json:"oldMode"`
	OldBlob string `json:"oldBlob"`
	NewMode string `json:"newMode"`
	NewBlob string `json:"newBlob"`
}

func (info ReadThroughInfo) changes() readthrough.Range {
	return readthrough.Range{Base: info.Base, Head: info.Head, Uncommitted: info.Uncommitted}
}

func (s *ReadThroughService) Open(repoPath string, request ReadThroughRequest) (ReadThroughInfo, error) {
	ctx := context.Background()
	info := ReadThroughInfo{Files: []ReadThroughFile{}, Path: repoPath, Target: request.Target}
	if err := info.resolve(ctx, repoPath, request); err != nil {
		return info, err
	}
	info.Uncommitted = request.Uncommitted && info.CheckedOut
	forgetMarks(ctx, repoPath)

	branch := strings.TrimPrefix(info.Key, "refs/heads/")
	builtOn := ""
	for _, link := range linksOf(repoPath) {
		if link.Branch == branch && (info.Target == "" || info.Target == link.Parent) {
			info.Target, builtOn = link.Parent, link.ParentTip
		}
	}
	if info.Target == "" {
		target, err := git.CompareBase(ctx, repoPath)
		if err != nil || target == "" {
			return info, err
		}
		info.Target = target
	}
	target, err := git.ResolveCommit(ctx, repoPath, info.Target)
	if err != nil {
		return info, &gitexec.AppError{Message: info.Target + " doesn't exist.", Detail: err.Error()}
	}
	info.Base, err = readthrough.Base(ctx, repoPath, target, info.Head, builtOn)
	if errors.Is(err, git.ErrUnrelatedHistories) {
		return info, &gitexec.AppError{Message: info.Name + " and " + info.Target + " share no history."}
	}
	if err != nil {
		return info, err
	}
	files, skipped, err := readthrough.Files(ctx, info.Path, info.changes())
	if err != nil {
		return info, err
	}
	marks, err := readthrough.Marks(ctx, repoPath, info.Key)
	if err != nil {
		return info, err
	}
	info.Skipped = skipped
	info.Files = mapSlice(files, func(f readthrough.File) ReadThroughFile {
		return ReadThroughFile{
			Status:   f.Status,
			Path:     f.Path,
			OrigPath: f.OrigPath,
			Added:    f.Added,
			Removed:  f.Removed,
			Binary:   f.Binary,
			Kind:     string(f.Kind),
			Read:     readthrough.IsRead(marks, f),
			OldMode:  f.Old.Mode,
			OldBlob:  f.Old.Blob,
			NewMode:  f.New.Mode,
			NewBlob:  f.New.Blob,
		}
	})
	return info, nil
}

// resolve fills in what's read and where it's checked out.
func (info *ReadThroughInfo) resolve(ctx context.Context, repoPath string, request ReadThroughRequest) (err error) {
	if request.Path != "" {
		info.Path = request.Path
		info.CheckedOut = true
		info.Head, err = git.ResolveCommit(ctx, request.Path, "HEAD")
		if err != nil {
			return err
		}
		worktrees, err := git.ListWorktrees(ctx, request.Path)
		if err != nil {
			return err
		}
		info.Key, info.Name = worktreeKey+request.Path, filepath.Base(request.Path)
		for _, worktree := range worktrees {
			if !worktree.Current {
				continue
			}
			info.Key = worktreeKey + worktree.Path
			if worktree.Branch != "" {
				info.Key, info.Name = "refs/heads/"+worktree.Branch, worktree.Branch
			}
		}
		return nil
	}
	info.Key = request.Ref
	for _, prefix := range []string{"refs/heads/", "refs/remotes/", "refs/"} {
		if name, ok := strings.CutPrefix(request.Ref, prefix); ok {
			info.Name = name
			break
		}
	}
	if info.Head, err = git.ResolveCommit(ctx, repoPath, request.Ref); err != nil {
		return err
	}
	branches, err := git.ListBranches(ctx, repoPath)
	if err != nil {
		return err
	}
	for _, branch := range branches {
		if "refs/heads/"+branch.Name != request.Ref {
			continue
		}
		info.CheckedOut = branch.Current
		if _, err := os.Stat(branch.WorktreePath); !branch.Current && branch.WorktreePath != "" && err == nil {
			info.Path, info.CheckedOut = branch.WorktreePath, true
		}
	}
	return nil
}

// OfChange is a pull request as a request: the ref its head is at on this
// computer, "" when it isn't here, against its target as of the last fetch.
func (s *ReadThroughService) OfChange(repoPath, sourceBranch, targetBranch, headSHA string) (ReadThroughRequest, error) {
	ctx := context.Background()
	ref, err := git.RefAt(ctx, repoPath, headSHA, sourceBranch)
	if err != nil || ref == "" {
		return ReadThroughRequest{}, err
	}
	request := ReadThroughRequest{Ref: ref, Uncommitted: true}
	for _, target := range []string{"origin/" + targetBranch, targetBranch} {
		if _, err := git.ResolveCommit(ctx, repoPath, target); err == nil {
			request.Target = target
			break
		}
	}
	return request, nil
}

func (s *ReadThroughService) FileDiff(info ReadThroughInfo, origPath, path string, force, ignoreWhitespace bool) (FileDiff, error) {
	ctx := context.Background()
	revs := []string{info.Base}
	if info.Uncommitted {
		if stat, err := os.Stat(filepath.Join(info.Path, filepath.FromSlash(path))); err == nil && !force && isLarge(stat.Size()) {
			return FileDiff{TooLarge: true, SizeBytes: stat.Size()}, nil
		}
	} else {
		revs = append(revs, info.Head)
	}
	if large, size := largestVersion(ctx, info.Path, path, force, revs...); large {
		return FileDiff{TooLarge: true, SizeBytes: size}, nil
	}
	fd, err := readthrough.FileDiff(ctx, info.Path, info.changes(), origPath, path, ignoreWhitespace)
	if err != nil {
		return FileDiff{}, err
	}
	return toFileDiff(fd), nil
}

// SetRead marks the files as read, or as not. They're as Open listed them.
func (s *ReadThroughService) SetRead(info ReadThroughInfo, files []ReadThroughFile, read bool) error {
	listed := mapSlice(files, func(f ReadThroughFile) readthrough.File {
		file := readthrough.File{}
		file.Path = f.Path
		file.Old = git.FileVersion{Mode: f.OldMode, Blob: f.OldBlob}
		file.New = git.FileVersion{Mode: f.NewMode, Blob: f.NewBlob}
		return file
	})
	return readthrough.SetRead(context.Background(), info.Path, info.Key, info.changes(), listed, read)
}

// ReadThroughNote is on a line of the new version, a line of the old one, or the
// whole file. Outdated means that line has changed since, so Line and OldLine are
// where it was.
type ReadThroughNote struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	OldPath  string `json:"oldPath"`
	Line     int    `json:"line"`
	OldLine  int    `json:"oldLine"`
	LineText string `json:"lineText"`
	Text     string `json:"text"`
	Outdated bool   `json:"outdated"`
}

// Notes lists what was noted while reading, in the order of the files.
func (s *ReadThroughService) Notes(info ReadThroughInfo) ([]ReadThroughNote, error) {
	ctx := context.Background()
	stored, err := notes.List(notesKey(ctx, info.Path))
	if err != nil {
		return nil, err
	}
	list := []ReadThroughNote{}
	for _, note := range stored {
		if note.Kind != notes.KindReview || note.Target != info.Key {
			continue
		}
		line, oldLine, outdated := note.Line, note.OldLine, false
		if line+oldLine > 0 {
			content, err := info.contentFor(ctx, note)
			if err != nil {
				return nil, err
			}
			at, found := note.LineIn(content)
			outdated = !found
			if found && line > 0 {
				line = at
			} else if found {
				oldLine = at
			}
		}
		list = append(list, ReadThroughNote{note.ID, note.Path, note.OldPath, line, oldLine, note.LineText, note.Text, outdated})
	}
	slices.SortStableFunc(list, func(a, b ReadThroughNote) int {
		return cmp.Or(strings.Compare(a.Path, b.Path), cmp.Compare(a.Line+a.OldLine, b.Line+b.OldLine))
	})
	return list, nil
}

// contentFor is the version of the file the note is on, nil when it's gone.
func (info ReadThroughInfo) contentFor(ctx context.Context, note notes.Note) ([]byte, error) {
	var data []byte
	var err error
	switch {
	case note.OldLine > 0:
		data, _, err = diff.ReadRevisionFile(ctx, info.Path, info.Base, cmp.Or(note.OldPath, note.Path), maxFileContentBytes)
	case info.Uncommitted:
		data, _, err = diff.ReadWorkingFile(info.Path, note.Path, maxFileContentBytes)
	default:
		data, _, err = diff.ReadRevisionFile(ctx, info.Path, info.Head, note.Path, maxFileContentBytes)
	}
	if errors.Is(err, diff.ErrFileTooLarge) {
		return nil, nil
	}
	return data, err
}

// AddNote keeps a note with the line it's written on.
func (s *ReadThroughService) AddNote(info ReadThroughInfo, note ReadThroughNote) error {
	ctx := context.Background()
	added := notes.Note{
		Kind:      notes.KindReview,
		Target:    info.Key,
		Text:      strings.TrimSpace(note.Text),
		UpdatedAt: time.Now().UTC(),
		ID:        rand.Text(),
		Path:      note.Path,
		OldPath:   note.OldPath,
		Line:      note.Line,
		OldLine:   note.OldLine,
	}
	if note.Line+note.OldLine > 0 {
		content, err := info.contentFor(ctx, added)
		if err != nil {
			return err
		}
		added.Anchor(content)
	}
	return notes.Set(notesKey(ctx, info.Path), added)
}

// forgetMarks drops the marks of branches and worktrees that are gone. The
// screen opens without it, so a failure can wait for the next time.
func forgetMarks(ctx context.Context, repoPath string) {
	if gone, err := readThroughGone(ctx, repoPath); err == nil {
		_ = readthrough.Forget(ctx, repoPath, gone)
	}
}

// readThroughGone says whether what a key names has been deleted.
func readThroughGone(ctx context.Context, repoPath string) (func(key string) bool, error) {
	worktrees, err := git.ListWorktrees(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	refs, err := gitexec.RunIn(ctx, repoPath, "for-each-ref", "--format=%(refname)")
	if err != nil {
		return nil, err
	}
	present := map[string]bool{}
	for _, ref := range strings.Split(refs.Stdout, "\n") {
		present[ref] = true
	}
	for _, worktree := range worktrees {
		present[worktreeKey+worktree.Path] = true
	}
	return func(key string) bool { return !present[key] }, nil
}
