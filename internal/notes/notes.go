// Package notes keeps private notes on commits, branches, tags and files in the
// app's own file, outside every repository, so no Git operation can carry them.
package notes

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/gspeager/current-client/internal/config"
)

const (
	KindCommit = "commit"
	KindBranch = "branch"
	KindTag    = "tag"
	KindFile   = "file"
	// KindReview is a note taken while reading a change through, on a file or a
	// line. Target is what's read, and it can hold many.
	KindReview = "review"
)

// Note.Subject is the commit's subject when written, so a note still makes sense
// after a rebase has rewritten the commit it was on.
type Note struct {
	Kind      string    `json:"kind"`
	Target    string    `json:"target"`
	Subject   string    `json:"subject,omitempty"`
	Text      string    `json:"text"`
	UpdatedAt time.Time `json:"updatedAt"`

	// Of a review note. Line is in the new version and OldLine in the old; with
	// neither, it's on the whole file. OldPath is a renamed file's. LineText and
	// Before are the line when written and the one before it.
	ID       string `json:"id,omitempty"`
	Path     string `json:"path,omitempty"`
	OldPath  string `json:"oldPath,omitempty"`
	Line     int    `json:"line,omitempty"`
	OldLine  int    `json:"oldLine,omitempty"`
	LineText string `json:"lineText,omitempty"`
	Before   string `json:"before,omitempty"`
}

func (n Note) sameAs(other Note) bool {
	if n.Kind == KindReview || other.Kind == KindReview {
		return n.Kind == other.Kind && n.ID == other.ID
	}
	return n.Kind == other.Kind && n.Target == other.Target
}

func linesOf(content []byte) []string {
	return strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
}

// Anchor records the line the note is on, from the content it was written on.
func (n *Note) Anchor(content []byte) {
	at, lines := n.Line+n.OldLine, linesOf(content)
	if at <= len(lines) {
		n.LineText = lines[at-1]
		if at > 1 {
			n.Before = lines[at-2]
		}
	}
}

// LineIn finds the note's line in content, where lines above it may have come or
// gone. It's false when that line has changed. A line with no letter or digit,
// such as a closing brace, is everywhere, so the one before it must match too.
func (n Note) LineIn(content []byte) (int, bool) {
	at := n.Line + n.OldLine
	lines := linesOf(content)
	common := !strings.ContainsFunc(n.LineText, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) })
	nearest := 0
	before := ""
	for i, text := range lines {
		matches := text == n.LineText && (!common || before == n.Before)
		before = text
		if !matches {
			continue
		}
		if nearest == 0 || abs(i+1-at) < abs(nearest-at) {
			nearest = i + 1
		}
	}
	return nearest, nearest > 0
}

func abs(n int) int {
	return max(n, -n)
}

type file struct {
	Version      int               `json:"version"`
	Repositories map[string][]Note `json:"repositories"`
}

var mu sync.Mutex

func path() (string, error) {
	settings, err := config.DefaultPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(settings), "notes.json"), nil
}

func load() (file, error) {
	loaded := file{Version: 1, Repositories: map[string][]Note{}}
	name, err := path()
	if err != nil {
		return loaded, err
	}
	data, err := os.ReadFile(name)
	if errors.Is(err, os.ErrNotExist) {
		return loaded, nil
	}
	if err != nil {
		return loaded, err
	}
	if err := json.Unmarshal(data, &loaded); err != nil {
		return loaded, err
	}
	if loaded.Repositories == nil {
		loaded.Repositories = map[string][]Note{}
	}
	return loaded, nil
}

// save replaces the file whole, so a crash can't leave half of it behind.
func save(contents file) error {
	name, err := path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(contents, "", "  ")
	if err != nil {
		return err
	}
	temp := name + ".tmp"
	if err := os.WriteFile(temp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(temp, name)
}

func List(repository string) ([]Note, error) {
	mu.Lock()
	defer mu.Unlock()
	contents, err := load()
	if err != nil {
		return nil, err
	}
	return contents.Repositories[repository], nil
}

// Set writes the note on a target, or removes it when text is empty.
func Set(repository string, note Note) error {
	mu.Lock()
	defer mu.Unlock()
	contents, err := load()
	if err != nil {
		return err
	}
	var kept []Note
	for _, existing := range contents.Repositories[repository] {
		if !existing.sameAs(note) {
			kept = append(kept, existing)
		}
	}
	if note.Text != "" {
		kept = append(kept, note)
	}
	if len(kept) == 0 {
		delete(contents.Repositories, repository)
	} else {
		contents.Repositories[repository] = kept
	}
	return save(contents)
}

// Edit changes a review note's text, or removes the note when text is empty.
func Edit(repository, id, text string) error {
	stored, err := List(repository)
	if err != nil {
		return err
	}
	for _, note := range stored {
		if note.Kind == KindReview && note.ID == id {
			note.Text, note.UpdatedAt = text, time.Now().UTC()
			return Set(repository, note)
		}
	}
	return errors.New("this note is gone")
}

// Rename moves a target's review notes to its new name.
func Rename(repository, target, renamed string) error {
	mu.Lock()
	defer mu.Unlock()
	contents, err := load()
	if err != nil {
		return err
	}
	moved := false
	for i, note := range contents.Repositories[repository] {
		if note.Kind == KindReview && note.Target == target {
			contents.Repositories[repository][i].Target = renamed
			moved = true
		}
	}
	if !moved {
		return nil
	}
	return save(contents)
}
