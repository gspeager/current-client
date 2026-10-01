package diff

import (
	"context"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func initExternalToolRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gittest.Run(t, dir, "init", "-b", "main")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	gittest.WriteFile(t, dir, "file.txt", "v1\n")
	gittest.Run(t, dir, "add", "file.txt")
	gittest.Run(t, dir, "commit", "-m", "v1")
	return dir
}

// The no-tool case isn.t run against real difftool, which can hang probing PATH.
func TestOpenInExternalToolFailsWithoutAConfiguredTool(t *testing.T) {
	dir := initExternalToolRepo(t)
	gittest.WriteFile(t, dir, "file.txt", "v2\n")

	if err := OpenInExternalTool(context.Background(), dir, "file.txt", false); err == nil {
		t.Fatal("expected an error with no diff.tool configured")
	}
}

func TestConfiguredDiffToolFallsBackToMergeTool(t *testing.T) {
	dir := initExternalToolRepo(t)
	gittest.Run(t, dir, "config", "merge.tool", "fakemerge")

	if got := configuredDiffTool(context.Background(), dir); got != "fakemerge" {
		t.Fatalf("configuredDiffTool = %q, want %q", got, "fakemerge")
	}
}

func TestOpenInExternalToolLaunchesConfiguredTool(t *testing.T) {
	dir := initExternalToolRepo(t)
	gittest.WriteFile(t, dir, "file.txt", "v2\n")
	gittest.Run(t, dir, "config", "difftool.faketool.cmd", "true")
	gittest.Run(t, dir, "config", "diff.tool", "faketool")

	if err := OpenInExternalTool(context.Background(), dir, "file.txt", false); err != nil {
		t.Fatalf("OpenInExternalTool: %v", err)
	}
}

func TestOpenInExternalToolCachedUsesIndexDiff(t *testing.T) {
	dir := initExternalToolRepo(t)
	gittest.WriteFile(t, dir, "file.txt", "v2\n")
	gittest.Run(t, dir, "add", "file.txt")
	gittest.Run(t, dir, "config", "difftool.faketool.cmd", "true")
	gittest.Run(t, dir, "config", "diff.tool", "faketool")

	if err := OpenInExternalTool(context.Background(), dir, "file.txt", true); err != nil {
		t.Fatalf("OpenInExternalTool (cached): %v", err)
	}
}
