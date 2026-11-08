package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/internal/notes"
)

type NotesService struct{}

// NoteInfo.Stale means the commit, branch, tag or file it's on is gone. The
// note stays, so nothing written is lost to a rebase or a clean-up.
type NoteInfo struct {
	Kind      string `json:"kind"`
	Target    string `json:"target"`
	Subject   string `json:"subject"`
	Text      string `json:"text"`
	UpdatedAt string `json:"updatedAt"`
	Stale     bool   `json:"stale"`
}

// Worktrees share one repository, so they share its notes.
func notesKey(ctx context.Context, repoPath string) string {
	common, err := git.CommonDir(ctx, repoPath)
	if err != nil {
		return repoPath
	}
	if !filepath.IsAbs(common) {
		common = filepath.Join(repoPath, common)
	}
	if resolved, err := filepath.EvalSymlinks(common); err == nil {
		return resolved
	}
	return common
}

func (s *NotesService) List(repoPath string) ([]NoteInfo, error) {
	ctx := context.Background()
	stored, err := notes.List(notesKey(ctx, repoPath))
	if err != nil {
		return nil, err
	}
	refs, err := git.ListRefs(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	present := func(kind git.RefKind, name string) bool {
		return slices.ContainsFunc(refs, func(ref git.Ref) bool { return ref.Kind == kind && ref.Name == name })
	}
	info := []NoteInfo{}
	for _, note := range stored {
		stale := false
		switch note.Kind {
		case notes.KindBranch:
			stale = !present(git.RefBranch, note.Target)
		case notes.KindTag:
			stale = !present(git.RefTag, note.Target)
		case notes.KindCommit:
			stale = !git.CommitReachable(ctx, repoPath, note.Target)
		case notes.KindFile:
			_, statErr := os.Stat(filepath.Join(repoPath, filepath.FromSlash(note.Target)))
			stale = statErr != nil
		}
		info = append(info, NoteInfo{note.Kind, note.Target, note.Subject, note.Text, note.UpdatedAt.Format(time.RFC3339), stale})
	}
	return info, nil
}

// Set writes the note, or removes it when text is empty.
func (s *NotesService) Set(repoPath, kind, target, subject, text string) error {
	if !slices.Contains([]string{notes.KindCommit, notes.KindBranch, notes.KindTag, notes.KindFile}, kind) {
		return fmt.Errorf("unknown kind of note: %s", kind)
	}
	return notes.Set(notesKey(context.Background(), repoPath), notes.Note{
		Kind:      kind,
		Target:    target,
		Subject:   subject,
		Text:      strings.TrimSpace(text),
		UpdatedAt: time.Now().UTC(),
	})
}
