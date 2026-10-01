package app

import (
	"context"

	"github.com/gspeager/current-client/core/git"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type PatchService struct{}

func (s *PatchService) ExportCommit(repoPath, sha, suggestedFilename string) error {
	patch, err := git.FormatPatch(context.Background(), repoPath, sha)
	if err != nil {
		return err
	}
	return savePatch(patch, suggestedFilename)
}

func (s *PatchService) ExportWorkingTree(repoPath, suggestedFilename string) error {
	patch, err := git.DiffPatch(context.Background(), repoPath)
	if err != nil {
		return err
	}
	return savePatch(patch, suggestedFilename)
}

func (s *PatchService) ExportWorkingTreePaths(repoPath string, paths []string, suggestedFilename string) error {
	patch, err := git.DiffPatchForPaths(context.Background(), repoPath, paths)
	if err != nil {
		return err
	}
	return savePatch(patch, suggestedFilename)
}

func (s *PatchService) ImportPatch(repoPath string) error {
	path, err := application.Get().Dialog.OpenFile().
		CanChooseFiles(true).
		CanChooseDirectories(false).
		SetTitle("Import Patch").
		AddFilter("Patch files", "*.patch").
		PromptForSingleSelection()
	if err != nil || path == "" {
		return nil
	}
	return git.ApplyPatch(context.Background(), repoPath, path)
}

func savePatch(patch, suggestedFilename string) error {
	_, err := saveFile("Export Patch", suggestedFilename, "Patch files", "*.patch", []byte(patch))
	return err
}
