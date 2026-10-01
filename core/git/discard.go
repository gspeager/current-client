package git

import "context"

func DiscardFile(ctx context.Context, repoPath, path string) error {
	return DiscardFiles(ctx, repoPath, []string{path})
}

// DiscardFiles deletes untracked paths with `git clean` and restores the
// rest with `git restore`.
func DiscardFiles(ctx context.Context, repoPath string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	statuses, err := GetStatus(ctx, repoPath)
	if err != nil {
		return err
	}
	untracked := make(map[string]bool)
	for _, s := range statuses {
		if s.IndexStatus == '?' {
			untracked[s.Path] = true
		}
	}

	var toClean, toRestore []string
	for _, p := range paths {
		if untracked[p] {
			toClean = append(toClean, p)
		} else {
			toRestore = append(toRestore, p)
		}
	}

	if len(toClean) > 0 {
		if _, err := runResult(ctx, repoPath, append([]string{"clean", "-f", "--"}, toClean...)...); err != nil {
			return err
		}
	}
	if len(toRestore) > 0 {
		if _, err := runResult(ctx, repoPath, append([]string{"restore", "--"}, toRestore...)...); err != nil {
			return err
		}
	}
	return nil
}
