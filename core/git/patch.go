package git

import (
	"bufio"
	"context"
	"os"
	"regexp"
)

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

// ImportPatch applies a patch file the way its format calls for: a mailbox
// patch from git format-patch becomes commits with git am, and a plain diff
// is applied to the working tree with git apply, which changes nothing if
// any part of it doesn't apply. asCommits reports which happened.
func ImportPatch(ctx context.Context, repoPath, patchPath string) (asCommits bool, err error) {
	mailbox, err := isMailboxPatch(patchPath)
	if err != nil {
		return false, err
	}
	if mailbox {
		return true, ApplyPatch(ctx, repoPath, patchPath)
	}
	_, err = runResult(ctx, repoPath, "apply", patchPath)
	return false, err
}

// format-patch starts each patch with "From <sha> <fixed date>".
var mailboxFirstLine = regexp.MustCompile(`^From [0-9a-f]{40,64} `)

func isMailboxPatch(patchPath string) (bool, error) {
	f, err := os.Open(patchPath)
	if err != nil {
		return false, err
	}
	defer f.Close()
	line, err := bufio.NewReader(f).ReadString('\n')
	if err != nil && line == "" {
		return false, nil
	}
	return mailboxFirstLine.MatchString(line), nil
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
