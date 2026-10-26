package app

import (
	"context"

	"github.com/gspeager/current-client/core/codeowners"
	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/hostlink"
)

// CodeownersService reads the repository's CODEOWNERS from the working tree
// with the primary remote's host rules. No request is made.
type CodeownersService struct{}

// Owners maps each path to its owners; paths with none are left out.
func (s *CodeownersService) Owners(repoPath string, paths []string) (map[string][]string, error) {
	out := map[string][]string{}
	file, err := loadCodeowners(repoPath)
	if err != nil || file == nil {
		return out, err
	}
	for _, path := range paths {
		if owners := file.Owners(path); len(owners) > 0 {
			out[path] = owners
		}
	}
	return out, nil
}

// SuggestedReviewers is the owners of the files branch changes against
// target, each once in CODEOWNERS order. A target that only exists on the
// remote is read as <remote>/<target>.
func (s *CodeownersService) SuggestedReviewers(repoPath, branch, target string) ([]string, error) {
	file, err := loadCodeowners(repoPath)
	if err != nil || file == nil {
		return nil, err
	}
	ctx := context.Background()
	files, err := git.RefRangeChangedFiles(ctx, repoPath, target, branch)
	if err != nil {
		remotes, remoteErr := git.ListRemotes(ctx, repoPath)
		if remoteErr != nil || len(remotes) == 0 {
			return nil, err
		}
		if files, err = git.RefRangeChangedFiles(ctx, repoPath, primaryRemote(remotes).Name+"/"+target, branch); err != nil {
			return nil, err
		}
	}
	var names []string
	seen := map[string]bool{}
	for _, f := range files {
		for _, name := range file.Owners(f.Path) {
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	return names, nil
}

func loadCodeowners(repoPath string) (*codeowners.File, error) {
	style := codeowners.GitHub
	if remotes, err := git.ListRemotes(context.Background(), repoPath); err == nil {
		if links := hostLinksFor(remotes, configOrDefault().Hosts); links != nil && links.Style == hostlink.GitLab {
			style = codeowners.GitLab
		}
	}
	file, _, err := codeowners.Load(repoPath, style)
	return file, err
}
