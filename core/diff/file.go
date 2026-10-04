package diff

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

// ErrFileTooLarge is returned instead of reading a file bigger than the
// caller's limit.
var ErrFileTooLarge = errors.New("file is too large to read")

// ReadRevisionFile reads path as it is at rev (any commit-ish), or in the
// index when rev is "". found is false when path doesn't exist there.
func ReadRevisionFile(ctx context.Context, repoPath, rev, path string, maxBytes int64) (data []byte, found bool, err error) {
	object := rev + ":" + path
	exec := gitexec.NewExecutor("")
	size, err := exec.RunChecked(ctx, gitexec.Command{Dir: repoPath, Args: []string{"cat-file", "-s", object}})
	if err != nil {
		// cat-file fails the same way for a missing path and a missing rev.
		return nil, false, nil
	}
	if n, _ := strconv.ParseInt(strings.TrimSpace(size.Stdout), 10, 64); n > maxBytes {
		return nil, true, ErrFileTooLarge
	}
	result, err := exec.RunChecked(ctx, gitexec.Command{Dir: repoPath, Args: []string{"cat-file", "blob", object}})
	if err != nil {
		return nil, true, err
	}
	return []byte(result.Stdout), true, nil
}

// ReadWorkingFile reads path from the working tree. path must stay inside
// the repository.
func ReadWorkingFile(repoPath, path string, maxBytes int64) (data []byte, found bool, err error) {
	local := filepath.FromSlash(path)
	if !filepath.IsLocal(local) {
		return nil, false, errors.New("path is outside the repository")
	}
	full := filepath.Join(repoPath, local)
	info, err := os.Stat(full)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if info.Size() > maxBytes {
		return nil, true, ErrFileTooLarge
	}
	data, err = os.ReadFile(full)
	return data, err == nil, err
}
