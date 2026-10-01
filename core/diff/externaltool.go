package diff

import (
	"context"
	"fmt"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

// OpenInExternalTool requires diff.tool or merge.tool to be set: otherwise
// git difftool auto-probes PATH and can pick a terminal tool like vimdiff,
// which hangs forever without a terminal. It blocks until the tool exits
// because git deletes the temp files it hands the tool afterwards.
func OpenInExternalTool(ctx context.Context, repoPath, path string, cached bool) error {
	if configuredDiffTool(ctx, repoPath) == "" {
		return fmt.Errorf("no diff tool configured — set diff.tool (or merge.tool) in your git config")
	}
	args := []string{"difftool", "--no-prompt"}
	if cached {
		args = append(args, "--cached")
	}
	args = append(args, "--", path)
	_, err := gitexec.NewExecutor("").RunChecked(ctx, gitexec.Command{
		Dir:  repoPath,
		Args: args,
	})
	return err
}

func configuredDiffTool(ctx context.Context, repoPath string) string {
	for _, key := range []string{"diff.tool", "merge.tool"} {
		result, err := gitexec.NewExecutor("").Run(ctx, gitexec.Command{
			Dir:  repoPath,
			Args: []string{"config", "--get", key},
		})
		if err != nil || result.ExitCode != 0 {
			continue
		}
		if tool := strings.TrimSpace(result.Stdout); tool != "" {
			return tool
		}
	}
	return ""
}
