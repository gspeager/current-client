package repository

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

type Repository struct {
	Path string
}

func Open(ctx context.Context, path string) (*Repository, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, &gitexec.AppError{Message: "Repository path not found.", Detail: err.Error()}
	}
	if !info.IsDir() {
		return nil, &gitexec.AppError{Message: "Repository path is not a folder.", Detail: path}
	}

	kind, err := revParse(ctx, path, "--is-bare-repository", "--is-inside-work-tree")
	if err != nil {
		return nil, err
	}
	switch {
	case kind[0] == "true":
		return nil, &gitexec.AppError{Message: "This is a bare repository, which has no working files. Open a clone of it instead.", Detail: path}
	case kind[1] != "true":
		return nil, &gitexec.AppError{Message: "This folder isn't part of a repository's working files.", Detail: path}
	}

	// Git reports file paths from the top of the working tree, so a subfolder
	// is opened as the whole repository.
	location, err := revParse(ctx, path, "--show-prefix", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	if location[0] != "" {
		path = filepath.FromSlash(location[1])
	}
	return &Repository{Path: path}, nil
}

// revParse returns one trimmed line per query, including empty ones.
func revParse(ctx context.Context, dir string, queries ...string) ([]string, error) {
	result, err := gitexec.NewExecutor("").RunChecked(ctx, gitexec.Command{
		Dir:  dir,
		Args: append([]string{"rev-parse"}, queries...),
	})
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimRight(result.Stdout, "\r\n"), "\n")
	for len(lines) < len(queries) {
		lines = append(lines, "")
	}
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	return lines, nil
}

func Init(ctx context.Context, path string) (*Repository, error) {
	_, err := gitexec.NewExecutor("").RunChecked(ctx, gitexec.Command{
		Args: []string{"init", path},
	})
	if err != nil {
		return nil, err
	}
	return &Repository{Path: path}, nil
}

func Clone(ctx context.Context, url, dest string) (*Repository, error) {
	_, err := gitexec.NewExecutor("").RunChecked(ctx, gitexec.Command{
		Args: []string{"clone", url, dest},
	})
	if err != nil {
		return nil, err
	}
	return &Repository{Path: dest}, nil
}
