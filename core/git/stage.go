package git

import "context"

func StageFile(ctx context.Context, repoPath, path string) error {
	return StageFiles(ctx, repoPath, []string{path})
}

func UnstageFile(ctx context.Context, repoPath, path string) error {
	return UnstageFiles(ctx, repoPath, []string{path})
}

// StageFiles uses one git process for all paths; concurrent per-file
// processes race for .git/index.lock.
func StageFiles(ctx context.Context, repoPath string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	_, err := runResult(ctx, repoPath, append([]string{"add", "--"}, paths...)...)
	return err
}

func UnstageFiles(ctx context.Context, repoPath string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	_, err := runResult(ctx, repoPath, append([]string{"reset", "--"}, paths...)...)
	return err
}
