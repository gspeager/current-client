package git

import (
	"context"
	"path/filepath"
	"strings"
)

type Submodule struct {
	Path string
	// Commit is the one checked out, or the one recorded when not initialised.
	Commit      string
	Initialized bool
	// CommitChanged means it's checked out at a different commit from the
	// one the repository records.
	CommitChanged bool
	Conflicted    bool
}

// ListSubmodules includes nested submodules, with paths from the top.
func ListSubmodules(ctx context.Context, repoPath string) ([]Submodule, error) {
	result, err := runResult(ctx, repoPath, "submodule", "status", "--recursive")
	if err != nil {
		return nil, err
	}
	return parseSubmoduleStatus(result.Stdout), nil
}

// Each line is a state character (' ', '-', '+' or 'U'), the commit, the
// path, and for an initialised one its describe in parentheses.
func parseSubmoduleStatus(output string) []Submodule {
	var submodules []Submodule
	for _, line := range strings.Split(output, "\n") {
		if len(line) < 2 {
			continue
		}
		commit, rest, ok := strings.Cut(line[1:], " ")
		if !ok {
			continue
		}
		if i := strings.LastIndex(rest, " ("); i != -1 && strings.HasSuffix(rest, ")") {
			rest = rest[:i]
		}
		submodules = append(submodules, Submodule{
			Path:          rest,
			Commit:        commit,
			Initialized:   line[0] != '-',
			CommitChanged: line[0] == '+',
			Conflicted:    line[0] == 'U',
		})
	}
	return submodules
}

// UpdateSubmodules initialises and checks out submodules at the commits the
// repository records, fetching them as needed. No paths means all of them.
func UpdateSubmodules(ctx context.Context, repoPath string, paths []string) error {
	args := append([]string{"submodule", "update", "--init", "--recursive", "--"}, paths...)
	_, err := runResult(ctx, repoPath, args...)
	return err
}

type SubmoduleCommit struct {
	SHA     string
	Subject string
	// Added is true for a commit the new version has and the old doesn't; false
	// for one it dropped (the submodule moved back or to another branch).
	Added bool
}

// SubmoduleCommits lists what changed between two commits of the submodule
// at path. It needs the submodule initialised, with both commits fetched.
func SubmoduleCommits(ctx context.Context, repoPath, path, from, to string) ([]SubmoduleCommit, error) {
	result, err := runResult(ctx, filepath.Join(repoPath, filepath.FromSlash(path)),
		"log", "--left-right", "--format=%m%x09%h%x09%s", from+"..."+to)
	if err != nil {
		return nil, err
	}
	var commits []SubmoduleCommit
	for _, line := range strings.Split(strings.TrimSpace(result.Stdout), "\n") {
		fields := strings.SplitN(line, "\t", 3)
		if len(fields) == 3 {
			commits = append(commits, SubmoduleCommit{SHA: fields[1], Subject: fields[2], Added: fields[0] == ">"})
		}
	}
	return commits, nil
}
