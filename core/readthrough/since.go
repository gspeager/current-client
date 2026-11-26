package readthrough

import (
	"cmp"
	"context"
	"errors"
	"slices"

	"github.com/gspeager/current-client/core/diff"
	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
)

// Reading is a range with what has been read of it.
type Reading struct {
	Files   []ReadFile
	Skipped int
	// How many files have been read at some time.
	Marked int
}

// ReadFile is a file of the range. Again means it has changed since it was read,
// and Since is how.
type ReadFile struct {
	File
	Read  bool
	Again bool
	Since Since
	// Gone means it was read and the range doesn't change it any more. It's only
	// here to show what changed since.
	Gone bool
}

// Since is a file compared with what was read of it.
type Since struct {
	Added   int
	Removed int
	Binary  bool
	// What was read didn't merge with the new base, so the base's own changes are
	// among these.
	Conflict bool

	from git.FileVersion
}

// Read lists what the range changes, with what has been read of it under key. What
// was read is first brought onto the range's base, so a rebase that changes
// nothing leaves a file read, and the base's own changes aren't taken for the
// branch's.
func Read(ctx context.Context, repoPath, key string, r Range) (Reading, error) {
	env, discard := git.ScratchObjects(ctx, repoPath)
	defer discard()
	return read(ctx, repoPath, env, key, r)
}

func read(ctx context.Context, repoPath string, env []string, key string, r Range) (Reading, error) {
	files, skipped, err := listFiles(ctx, repoPath, env, r)
	if err != nil {
		return Reading{}, err
	}
	marks, err := Marks(ctx, repoPath, key)
	if err != nil {
		return Reading{}, err
	}
	reading := Reading{Skipped: skipped, Marked: len(marks)}
	listed := map[string]bool{}
	for _, file := range files {
		listed[file.Path] = true
	}
	for _, file := range files {
		mark, marked := marks[file.Path]
		// Renamed since it was read.
		if before, there := marks[file.OrigPath]; !marked && there && !listed[file.OrigPath] {
			mark, marked = before, true
			marks[file.Path] = before
			delete(marks, file.OrigPath)
		}
		reading.Files = append(reading.Files, ReadFile{
			File: file, Read: marked && mark.Read == file.New, Again: marked && mark.Read != file.New,
		})
	}
	var gone []string
	for path := range marks {
		if !listed[path] {
			gone = append(gone, path)
		}
	}
	slices.Sort(gone)
	var atBase map[string]git.FileVersion
	if len(gone) > 0 {
		if atBase, err = (marker{ctx, repoPath, env}).versions(r.Base); err != nil {
			return Reading{}, err
		}
	}
	for _, path := range gone {
		version := atBase[path]
		if marks[path].Read == version {
			continue
		}
		file := ReadFile{Again: true, Gone: true}
		file.Path, file.Old, file.New = path, version, version
		reading.Files = append(reading.Files, file)
	}
	err = reading.compare(ctx, repoPath, env, marks, len(r.Head))
	return reading, err
}

// compare works out what changed since for the files that have, and which of them
// only took the new base's changes.
func (reading *Reading) compare(ctx context.Context, repoPath string, env []string, marks map[string]Mark, idLength int) error {
	m := marker{ctx, repoPath, env}
	var again []*ReadFile
	var rebased []*ReadFile
	for i := range reading.Files {
		file := &reading.Files[i]
		if !file.Again {
			continue
		}
		again = append(again, file)
		file.Since.from = marks[file.Path].Read
		if marks[file.Path].Base != file.Old {
			rebased = append(rebased, file)
		}
	}
	if len(again) == 0 {
		return nil
	}
	if err := carry(m, rebased, marks, idLength); err != nil {
		return err
	}
	from, to := map[string]git.FileVersion{}, map[string]git.FileVersion{}
	for _, file := range again {
		from[file.Path], to[file.Path] = file.Since.from, file.New
	}
	changes, err := m.changes(from, to, idLength)
	if err != nil {
		return err
	}
	for _, file := range again {
		change, changed := changes[file.Path]
		file.Since.Added, file.Since.Removed, file.Since.Binary = change.Added, change.Removed, change.Binary
		if changed {
			// One the range no longer changes has no status of its own.
			file.Status = cmp.Or(file.Status, change.Status)
			continue
		}
		file.Again, file.Read = false, true
	}
	reading.Files = slices.DeleteFunc(reading.Files, func(file ReadFile) bool { return file.Gone && !file.Again })
	return nil
}

// carry merges what was read of each file onto its new base. Git before 2.38
// can't, so there every one is a conflict.
func carry(m marker, files []*ReadFile, marks map[string]Mark, idLength int) error {
	if len(files) == 0 {
		return nil
	}
	before, reader, after := map[string]git.FileVersion{}, map[string]git.FileVersion{}, map[string]git.FileVersion{}
	for _, file := range files {
		mark := marks[file.Path]
		before[file.Path], reader[file.Path], after[file.Path] = mark.Base, mark.Read, file.Old
	}
	base, err := m.commit(before, idLength)
	if err != nil {
		return err
	}
	ours, err := m.commit(reader, idLength, "-p", base)
	if err != nil {
		return err
	}
	theirs, err := m.commit(after, idLength, "-p", base)
	if err != nil {
		return err
	}
	merge, err := git.MergeInMemory(m.ctx, m.repoPath, m.env, ours, theirs)
	unsupported := errors.Is(err, git.ErrMergeTreeUnsupported)
	if err != nil && !unsupported {
		return err
	}
	var versions map[string]git.FileVersion
	if !unsupported {
		if versions, err = m.versions(merge.Tree); err != nil {
			return err
		}
	}
	for _, file := range files {
		if unsupported || slices.Contains(merge.Files, file.Path) {
			file.Since.Conflict = true
		} else {
			file.Since.from = versions[file.Path]
		}
	}
	return nil
}

func (m marker) commit(entries map[string]git.FileVersion, idLength int, parents ...string) (string, error) {
	tree, err := m.writeTree("", entries, idLength)
	if err != nil {
		return "", err
	}
	return m.run("", append([]string{"commit-tree", tree, "-m", "read"}, parents...)...)
}

func (m marker) changes(from, to map[string]git.FileVersion, idLength int) (map[string]git.TreeChange, error) {
	before, after, err := m.trees(from, to, idLength)
	if err != nil {
		return nil, err
	}
	changes, err := git.TreeChangesWithoutRenames(m.ctx, m.repoPath, m.env, before, after)
	byPath := map[string]git.TreeChange{}
	for _, change := range changes {
		byPath[change.Path] = change
	}
	return byPath, err
}

func (m marker) trees(from, to map[string]git.FileVersion, idLength int) (before, after string, err error) {
	if before, err = m.writeTree("", from, idLength); err != nil {
		return "", "", err
	}
	after, err = m.writeTree("", to, idLength)
	return before, after, err
}

func (m marker) versions(tree string) (map[string]git.FileVersion, error) {
	output, err := m.run("", "ls-tree", "-r", "-z", "--full-tree", tree)
	if err != nil {
		return nil, err
	}
	return parseVersions(output), nil
}

func parseVersions(output string) map[string]git.FileVersion {
	versions := map[string]git.FileVersion{}
	for _, entry := range git.ParseTree(output) {
		versions[entry.Path] = git.FileVersion{Mode: entry.Mode, Blob: entry.Blob}
	}
	return versions
}

var errNotSince = &gitexec.AppError{Message: "This file hasn't changed since it was read."}

// SinceDiff is a file's diff from what was read of it.
func SinceDiff(ctx context.Context, repoPath, key string, r Range, path string, ignoreWhitespace bool) (diff.FileDiff, error) {
	env, discard := git.ScratchObjects(ctx, repoPath)
	defer discard()
	reading, err := read(ctx, repoPath, env, key, r)
	if err != nil {
		return diff.FileDiff{}, err
	}
	for _, file := range reading.Files {
		if file.Path != path || !file.Again {
			continue
		}
		m := marker{ctx, repoPath, env}
		before, after, err := m.trees(map[string]git.FileVersion{path: file.Since.from}, map[string]git.FileVersion{path: file.New}, len(r.Head))
		if err != nil {
			return diff.FileDiff{}, err
		}
		return diff.GetRangeDiff(ctx, repoPath, env, before, after, "", path, ignoreWhitespace)
	}
	return diff.FileDiff{}, errNotSince
}
