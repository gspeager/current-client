package git

import "context"

func FormatPatch(ctx context.Context, repoPath, sha string) (string, error) {
	result, err := runResult(ctx, repoPath, "format-patch", "-1", sha, "--stdout")
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}

func ApplyPatch(ctx context.Context, repoPath, patchPath string) error {
	_, err := runResult(ctx, repoPath, "am", patchPath)
	return err
}

// DiffPatch exports staged and unstaged changes for `git apply`. Untracked
// files are left out, since including them would require staging them.
func DiffPatch(ctx context.Context, repoPath string) (string, error) {
	return diffPatch(ctx, repoPath)
}

func DiffPatchForPaths(ctx context.Context, repoPath string, paths []string) (string, error) {
	if len(paths) == 0 {
		return "", nil
	}
	return diffPatch(ctx, repoPath, append([]string{"--"}, paths...)...)
}

func diffPatch(ctx context.Context, repoPath string, pathspec ...string) (string, error) {
	result, err := runResult(ctx, repoPath, append([]string{"diff", "HEAD", "--binary"}, pathspec...)...)
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}
