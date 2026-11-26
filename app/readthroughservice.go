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

	"github.com/gspeager/current-client/core/board"
	"github.com/gspeager/current-client/core/diff"
	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/readthrough"
	"github.com/gspeager/current-client/internal/notes"
	"github.com/gspeager/current-client/internal/structure"
)

type ReadThroughService struct{}

type ReadThroughRequest struct {
	// Ref is a ref's full name; with Path set, the worktree at Path is read instead.
	Ref  string `json:"ref"`
	Path string `json:"path"`
	// Target is what the ref is compared with. "" picks a stacked branch's parent
	// or the default branch.
	Target string `json:"target"`
	// Uncommitted includes uncommitted changes, where the ref is checked out.
	Uncommitted bool `json:"uncommitted"`
}

type ReadThroughInfo struct {
	// Key names the marks and notes of what's read.
	Key  string `json:"key"`
	Name string `json:"name"`
	// Path is the working tree the read through's commands run in.
	Path   string `json:"path"`
	Target string `json:"target"`
	Base   string `json:"base"`
	Head   string `json:"head"`
	// CheckedOut is whether the ref is what's checked out at Path.
	CheckedOut  bool              `json:"checkedOut"`
	Uncommitted bool              `json:"uncommitted"`
	Files       []ReadThroughFile `json:"files"`
	Skipped     int               `json:"skipped"`
	// Marked counts the files that have been read at some time.
	Marked int `json:"marked"`
}

type ReadThroughFile struct {
	Status   string `json:"status"`
	Path     string `json:"path"`
	OrigPath string `json:"origPath"`
	Added    int    `json:"added"`
	Removed  int    `json:"removed"`
	Binary   bool   `json:"binary"`
	// Kind is "", "test", "generated" or "lockfile".
	Kind string `json:"kind"`
	Read bool   `json:"read"`
	// Again means the file changed since it was read, by SinceAdded and
	// SinceRemoved lines.
	Again bool `json:"again"`
	// Gone means the file was read and the change no longer holds it.
	Gone         bool `json:"gone"`
	SinceAdded   int  `json:"sinceAdded"`
	SinceRemoved int  `json:"sinceRemoved"`
	SinceBinary  bool `json:"sinceBinary"`
	// Conflict means the new base's changes are among the Since lines.
	Conflict bool   `json:"conflict"`
	OldMode  string `json:"oldMode"`
	OldBlob  string `json:"oldBlob"`
	NewMode  string `json:"newMode"`
	NewBlob  string `json:"newBlob"`
}

// settleBase passes readthrough.SettleBase the stack links from the settings.
func (info *ReadThroughInfo) settleBase(ctx context.Context, repoPath string) (err error) {
	info.Target, info.Base, err = readthrough.SettleBase(ctx, repoPath, info.Key, info.Name, info.Target, info.Head, linksOf(repoPath))
	return err
}

func (info *ReadThroughInfo) resolve(ctx context.Context, repoPath string, request ReadThroughRequest) error {
	target, err := readthrough.Resolve(ctx, repoPath, request.Ref, request.Path)
	info.Key, info.Name, info.Head, info.CheckedOut = target.Key, target.Name, target.Head, target.CheckedOut
	// A ref not checked out in a worktree of its own is read in repoPath.
	if target.Path != "" {
		info.Path = target.Path
	}
	return err
}

// forgetMarks drops the marks of branches and worktrees that are gone. The
// screen opens without it, so a failure can wait for the next time.
func forgetMarks(ctx context.Context, repoPath string) {
	_ = readthrough.ForgetGone(ctx, repoPath)
}

func (info ReadThroughInfo) changes() readthrough.Range {
	return readthrough.Range{Base: info.Base, Head: info.Head, Uncommitted: info.Uncommitted}
}

func (s *ReadThroughService) Open(ctx context.Context, repoPath string, request ReadThroughRequest) (ReadThroughInfo, error) {
	info := ReadThroughInfo{Files: []ReadThroughFile{}, Path: repoPath, Target: request.Target}
	if err := info.resolve(ctx, repoPath, request); err != nil {
		return info, err
	}
	info.Uncommitted = request.Uncommitted && info.CheckedOut
	forgetMarks(ctx, repoPath)

	if err := info.settleBase(ctx, repoPath); err != nil || info.Target == "" {
		return info, err
	}
	reading, err := readthrough.Read(ctx, info.Path, info.Key, info.changes())
	if err != nil {
		return info, err
	}
	info.Skipped, info.Marked = reading.Skipped, reading.Marked
	info.Files = mapSlice(reading.Files, func(f readthrough.ReadFile) ReadThroughFile {
		return ReadThroughFile{
			Status:       f.Status,
			Path:         f.Path,
			OrigPath:     f.OrigPath,
			Added:        f.Added,
			Removed:      f.Removed,
			Binary:       f.Binary,
			Kind:         string(f.Kind),
			Read:         f.Read,
			Again:        f.Again,
			Gone:         f.Gone,
			SinceAdded:   f.Since.Added,
			SinceRemoved: f.Since.Removed,
			SinceBinary:  f.Since.Binary,
			Conflict:     f.Since.Conflict,
			OldMode:      f.Old.Mode,
			OldBlob:      f.Old.Blob,
			NewMode:      f.New.Mode,
			NewBlob:      f.New.Blob,
		}
	})
	return info, nil
}

// OfChange turns a pull request into a request for the local ref at its head,
// compared with its target as of the last fetch. Ref is "" when the head isn't
// on this computer.
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
	if large, size := info.isLarge(ctx, path, force); large {
		return FileDiff{TooLarge: true, SizeBytes: size}, nil
	}
	fd, err := readthrough.FileDiff(ctx, info.Path, info.changes(), origPath, path, ignoreWhitespace)
	if err != nil {
		return FileDiff{}, err
	}
	return toFileDiff(fd), nil
}

func (s *ReadThroughService) SinceDiff(info ReadThroughInfo, path string, force, ignoreWhitespace bool) (FileDiff, error) {
	ctx := context.Background()
	if large, size := info.isLarge(ctx, path, force); large {
		return FileDiff{TooLarge: true, SizeBytes: size}, nil
	}
	fd, err := readthrough.SinceDiff(ctx, info.Path, info.Key, info.changes(), path, ignoreWhitespace)
	if err != nil {
		return FileDiff{}, err
	}
	return toFileDiff(fd), nil
}

func (info ReadThroughInfo) isLarge(ctx context.Context, path string, force bool) (bool, int64) {
	if !info.Uncommitted {
		return largestVersion(ctx, info.Path, path, force, info.Base, info.Head)
	}
	if stat, err := os.Stat(filepath.Join(info.Path, filepath.FromSlash(path))); err == nil && !force && isLarge(stat.Size()) {
		return true, stat.Size()
	}
	return largestVersion(ctx, info.Path, path, force, info.Base)
}

func (s *ReadThroughService) Findings(info ReadThroughInfo) (RiskReportInfo, error) {
	return riskReportInfo(readthrough.Findings(context.Background(), info.Path, info.changes(), configOrDefault().RiskRulesOff))
}

// SetRead takes files as Open listed them. A gone file is always unmarked, as
// there's nothing left of it to have read.
func (s *ReadThroughService) SetRead(info ReadThroughInfo, files []ReadThroughFile, read bool) error {
	var kept, gone []readthrough.File
	for _, f := range files {
		file := readthrough.File{}
		file.Path, file.OrigPath = f.Path, f.OrigPath
		file.Old = git.FileVersion{Mode: f.OldMode, Blob: f.OldBlob}
		file.New = git.FileVersion{Mode: f.NewMode, Blob: f.NewBlob}
		if f.Gone {
			gone = append(gone, file)
		} else {
			kept = append(kept, file)
		}
	}
	ctx := context.Background()
	set := func(files []readthrough.File, read bool) error {
		if len(files) == 0 {
			return nil
		}
		return readthrough.SetRead(ctx, info.Path, info.Key, info.changes(), files, read)
	}
	return errors.Join(set(kept, read), set(gone, false))
}

// ReadThroughNote is on a line of the new version (Line), a line of the old one
// (OldLine), or the whole file when both are 0.
type ReadThroughNote struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	OldPath  string `json:"oldPath"`
	Line     int    `json:"line"`
	OldLine  int    `json:"oldLine"`
	LineText string `json:"lineText"`
	Text     string `json:"text"`
	// Outdated means the noted line has changed since, so Line and OldLine are
	// where it was.
	Outdated bool `json:"outdated"`
}

// Notes is in the order of the files.
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

// contentFor is nil when the version the note is on is gone or too large.
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

// AddNote anchors the note to its line's text, so it can find the line again.
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

type ChangeStructureInfo struct {
	Areas     []ChangeArea   `json:"areas"`
	Links     []ChangeLink   `json:"links"`
	ReadFirst []ChangeReason `json:"readFirst"`
	Skim      []ChangeReason `json:"skim"`
	// Read is false where links couldn't be read: no language is read, or a
	// partial clone would fetch the base's files. Then there are only Areas.
	Read      bool     `json:"read"`
	Partial   bool     `json:"partial"`
	Languages []string `json:"languages"`
}

type ChangeArea struct {
	Path string `json:"path"`
	// State is "added" or "removed" for a folder that's new or gone.
	State   string   `json:"state"`
	Files   []string `json:"files"`
	Added   int      `json:"added"`
	Removed int      `json:"removed"`
}

type ChangeLink struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Count int    `json:"count"`
	State string `json:"state"`
}

type ChangeReason struct {
	Path   string `json:"path"`
	Folder bool   `json:"folder"`
	// Why is "used" (by Count of the change's areas or of all files), a file's
	// kind, "renamed", or "few" lines in all.
	Why   string `json:"why"`
	Count int    `json:"count"`
}

func (s *ReadThroughService) Structure(ctx context.Context, info ReadThroughInfo) (ChangeStructureInfo, error) {
	var changes []structure.Changed
	for _, f := range info.Files {
		// A gone file is listed only to say it changed since it was read.
		if !f.Gone {
			changes = append(changes, structure.Changed{Path: f.Path, OrigPath: f.OrigPath, Status: f.Status, Added: f.Added, Removed: f.Removed})
		}
	}
	result := ChangeStructureInfo{Partial: git.PartialClone(ctx, info.Path)}
	var now, then *structure.Graph
	if !result.Partial {
		head := info.Head
		var env []string
		if info.Uncommitted {
			var discard func()
			env, discard = git.ScratchWrites(ctx, info.Path)
			defer discard()
			var err error
			if head, err = board.Tree(ctx, info.Path, env); err != nil {
				return result, err
			}
		}
		var err error
		if now, err = structure.Build(ctx, info.Path, env, head); err != nil {
			return result, err
		}
		if then, err = structure.Build(ctx, info.Path, nil, info.Base); err != nil {
			return result, err
		}
		result.Languages = now.Languages
		result.Read = len(now.Languages) > 0
	}
	shape := structure.CompareChange(now, then, changes)
	result.Areas = mapSlice(shape.Areas, func(a structure.Area) ChangeArea { return ChangeArea(a) })
	result.Links = mapSlice(shape.Links, func(l structure.AreaLink) ChangeLink { return ChangeLink(l) })
	reason := func(r structure.Reason) ChangeReason { return ChangeReason(r) }
	result.ReadFirst, result.Skim = mapSlice(shape.ReadFirst, reason), mapSlice(shape.Skim, reason)
	return result, nil
}
