package app

import (
	"context"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/undo"
)

type UndoService struct{}

type UndoPlanInfo struct {
	Operation string        `json:"operation"`
	Detail    string        `json:"detail"`
	Branch    string        `json:"branch"`
	From      string        `json:"from"`
	To        string        `json:"to"`
	SwitchTo  string        `json:"switchTo"`
	Mode      git.ResetMode `json:"mode"` // "soft", "keep", or "" for a switch
	Removed   int           `json:"removed"`
	Restored  int           `json:"restored"`
	Pushed    bool          `json:"pushed"`
}

func (s *UndoService) Preview(repoPath string) (UndoPlanInfo, error) {
	plan, err := undo.Preview(context.Background(), repoPath)
	return UndoPlanInfo(plan), err
}

func (s *UndoService) Apply(repoPath string, plan UndoPlanInfo) error {
	return undo.Apply(context.Background(), repoPath, undo.Plan(plan))
}
