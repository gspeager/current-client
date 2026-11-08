// Package notes keeps private notes on commits, branches, tags and files in the
// app's own file, outside every repository, so no Git operation can carry them.
package notes

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gspeager/current-client/internal/config"
)

const (
	KindCommit = "commit"
	KindBranch = "branch"
	KindTag    = "tag"
	KindFile   = "file"
)

// Note.Subject is the commit's subject when written, so a note still makes sense
// after a rebase has rewritten the commit it was on.
type Note struct {
	Kind      string    `json:"kind"`
	Target    string    `json:"target"`
	Subject   string    `json:"subject,omitempty"`
	Text      string    `json:"text"`
	UpdatedAt time.Time `json:"updatedAt"`
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
		if existing.Kind != note.Kind || existing.Target != note.Target {
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
