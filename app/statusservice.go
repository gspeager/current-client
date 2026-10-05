package app

import (
	"context"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/repository"
)

type StatusService struct{}

type FileStatus struct {
	Path           string `json:"path"`
	OrigPath       string `json:"origPath"`
	IndexStatus    string `json:"indexStatus"`
	WorktreeStatus string `json:"worktreeStatus"`
	Conflicted     bool   `json:"conflicted"`
	IndexAdded     int    `json:"indexAdded"`
	IndexRemoved   int    `json:"indexRemoved"`
	IndexBinary    bool   `json:"indexBinary"`
	WorkAdded      int    `json:"workAdded"`
	WorkRemoved    int    `json:"workRemoved"`
	WorkBinary     bool   `json:"workBinary"`
	// Submodule is set for a submodule: what changed inside it.
	Submodule *SubmoduleState `json:"submodule"`
	// LFS is set when .gitattributes stores the file in Git LFS.
	LFS bool `json:"lfs"`
}

type SubmoduleState struct {
	CommitChanged bool `json:"commitChanged"`
	Modified      bool `json:"modified"`
	Untracked     bool `json:"untracked"`
}

// GetStatus reports staged and unstaged line counts separately, since a
// partially staged file has a diff on each side.
func (s *StatusService) GetStatus(repoPath string) ([]FileStatus, error) {
	ctx := context.Background()
	statuses, err := git.GetStatus(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	workStats, err := git.WorkingTreeNumstat(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	indexStats, err := git.IndexNumstat(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	workByPath := numstatByPath(workStats)
	indexByPath := numstatByPath(indexStats)
	lfs, err := git.LFSPaths(ctx, repoPath, mapSlice(statuses, func(st git.Status) string { return st.Path }))
	if err != nil {
		return nil, err
	}

	result := make([]FileStatus, len(statuses))
	for i, st := range statuses {
		fs := FileStatus{
			Path:           st.Path,
			OrigPath:       st.OrigPath,
			IndexStatus:    string(st.IndexStatus),
			WorktreeStatus: string(st.WorktreeStatus),
			Conflicted:     st.Conflicted,
			LFS:            lfs[st.Path],
		}
		if st.Submodule != nil {
			state := SubmoduleState(*st.Submodule)
			fs.Submodule = &state
		}
		if n, ok := workByPath[st.Path]; ok {
			fs.WorkAdded, fs.WorkRemoved, fs.WorkBinary = n.Added, n.Removed, n.Binary
		}
		if n, ok := indexByPath[st.Path]; ok {
			fs.IndexAdded, fs.IndexRemoved, fs.IndexBinary = n.Added, n.Removed, n.Binary
		}
		result[i] = fs
	}
	return result, nil
}

func numstatByPath(entries []git.NumstatEntry) map[string]git.NumstatEntry {
	byPath := make(map[string]git.NumstatEntry, len(entries))
	for _, n := range entries {
		byPath[n.Path] = n
	}
	return byPath
}

func (s *StatusService) StageFile(repoPath, path string) error {
	return git.StageFile(context.Background(), repoPath, path)
}

func (s *StatusService) UnstageFile(repoPath, path string) error {
	return git.UnstageFile(context.Background(), repoPath, path)
}

func (s *StatusService) StageFiles(repoPath string, paths []string) error {
	return git.StageFiles(context.Background(), repoPath, paths)
}

func (s *StatusService) UnstageFiles(repoPath string, paths []string) error {
	return git.UnstageFiles(context.Background(), repoPath, paths)
}

func (s *StatusService) DiscardFile(repoPath, path string) error {
	return git.DiscardFile(context.Background(), repoPath, path)
}

func (s *StatusService) DiscardFiles(repoPath string, paths []string) error {
	return git.DiscardFiles(context.Background(), repoPath, paths)
}

func (s *StatusService) ListFiles(repoPath string) ([]string, error) {
	return git.WatchedPaths(context.Background(), repoPath)
}

func (s *StatusService) AddToGitignore(repoPath, pattern string) error {
	return repository.AddToGitignore(repoPath, pattern)
}
