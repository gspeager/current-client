package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestFormatPatchAndApplyReproducesCommit(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1\n", "initial")
	parentSHA := gittest.Run(t, dir, "rev-parse", "HEAD")

	gittest.CommitFile(t, dir, "file.txt", "v2\n", "add v2 line")
	targetSHA := gittest.Run(t, dir, "rev-parse", "HEAD")

	patch, err := FormatPatch(context.Background(), dir, targetSHA)
	if err != nil {
		t.Fatalf("FormatPatch: %v", err)
	}
	if !strings.Contains(patch, "Subject: [PATCH] add v2 line") {
		t.Fatalf("patch = %q, want a Subject header with the commit message", patch)
	}

	// Apply it to a fresh clone checked out at the commit's parent.
	clone := t.TempDir()
	gittest.Run(t, "", "clone", "-q", "-c", "core.autocrlf=false", dir, clone)
	gittest.Run(t, clone, "checkout", "-q", parentSHA)
	gittest.Run(t, clone, "config", "user.email", "test@example.com")
	gittest.Run(t, clone, "config", "user.name", "Test")

	patchPath := filepath.Join(t.TempDir(), "commit.patch")
	if err := os.WriteFile(patchPath, []byte(patch), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := ApplyPatch(context.Background(), clone, patchPath); err != nil {
		t.Fatalf("ApplyPatch: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(clone, "file.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "v2\n" {
		t.Fatalf("file.txt = %q, want %q", content, "v2\n")
	}
	subject := gittest.Run(t, clone, "log", "-1", "--pretty=%s")
	if subject != "add v2 line" {
		t.Fatalf("applied commit subject = %q, want %q", subject, "add v2 line")
	}
}

func TestDiffPatchCapturesStagedAndUnstagedAndApplies(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.WriteFile(t, dir, "staged.txt", "v1\n")
	gittest.WriteFile(t, dir, "unstaged.txt", "v1\n")
	gittest.Run(t, dir, "add", "staged.txt", "unstaged.txt")
	gittest.Run(t, dir, "commit", "-m", "initial")
	headSHA := gittest.Run(t, dir, "rev-parse", "HEAD")

	gittest.WriteFile(t, dir, "staged.txt", "v2\n")
	gittest.Run(t, dir, "add", "staged.txt")
	gittest.WriteFile(t, dir, "unstaged.txt", "v2\n") // left unstaged deliberately

	patch, err := DiffPatch(context.Background(), dir)
	if err != nil {
		t.Fatalf("DiffPatch: %v", err)
	}
	if !strings.Contains(patch, "staged.txt") || !strings.Contains(patch, "unstaged.txt") {
		t.Fatalf("patch = %q, want both staged.txt and unstaged.txt", patch)
	}

	// Apply it to a fresh, clean clone at the same commit.
	clone := t.TempDir()
	gittest.Run(t, "", "clone", "-q", "-c", "core.autocrlf=false", dir, clone)
	gittest.Run(t, clone, "checkout", "-q", headSHA)

	patchPath := filepath.Join(t.TempDir(), "working-tree.patch")
	if err := os.WriteFile(patchPath, []byte(patch), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	applyCmd := exec.Command("git", "apply", patchPath)
	applyCmd.Dir = clone
	if out, err := applyCmd.CombinedOutput(); err != nil {
		t.Fatalf("git apply: %v\n%s", err, out)
	}

	staged, err := os.ReadFile(filepath.Join(clone, "staged.txt"))
	if err != nil {
		t.Fatalf("ReadFile staged.txt: %v", err)
	}
	if string(staged) != "v2\n" {
		t.Fatalf("staged.txt = %q, want %q", staged, "v2\n")
	}
	unstaged, err := os.ReadFile(filepath.Join(clone, "unstaged.txt"))
	if err != nil {
		t.Fatalf("ReadFile unstaged.txt: %v", err)
	}
	if string(unstaged) != "v2\n" {
		t.Fatalf("unstaged.txt = %q, want %q", unstaged, "v2\n")
	}
}

func TestDiffPatchForPathsScopesToGivenFiles(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.WriteFile(t, dir, "a.txt", "v1\n")
	gittest.WriteFile(t, dir, "b.txt", "v1\n")
	gittest.Run(t, dir, "add", "a.txt", "b.txt")
	gittest.Run(t, dir, "commit", "-m", "initial")

	gittest.WriteFile(t, dir, "a.txt", "v2\n")
	gittest.WriteFile(t, dir, "b.txt", "v2\n")

	patch, err := DiffPatchForPaths(context.Background(), dir, []string{"a.txt"})
	if err != nil {
		t.Fatalf("DiffPatchForPaths: %v", err)
	}
	if !strings.Contains(patch, "a.txt") {
		t.Fatalf("patch = %q, want it to mention a.txt", patch)
	}
	if strings.Contains(patch, "b.txt") {
		t.Fatalf("patch = %q, want it to exclude b.txt (not in the requested paths)", patch)
	}
}

func TestDiffPatchForPathsEmptyReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init", "-b", "main")
	patch, err := DiffPatchForPaths(context.Background(), dir, nil)
	if err != nil {
		t.Fatalf("DiffPatchForPaths: %v", err)
	}
	if patch != "" {
		t.Fatalf("patch = %q, want empty for no paths", patch)
	}
}

func TestImportPatchAppliesAPlainDiffToTheWorkingTree(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1\n", "initial")
	gittest.WriteFile(t, dir, "file.txt", "v2\n")
	patch, err := DiffPatch(context.Background(), dir)
	if err != nil {
		t.Fatalf("DiffPatch: %v", err)
	}
	patchPath := filepath.Join(t.TempDir(), "working-tree.patch")
	if err := os.WriteFile(patchPath, []byte(patch), 0o644); err != nil {
		t.Fatal(err)
	}
	gittest.Run(t, dir, "checkout", "--", "file.txt")
	head := gittest.Run(t, dir, "rev-parse", "HEAD")

	asCommits, err := ImportPatch(context.Background(), dir, patchPath)
	if err != nil {
		t.Fatalf("ImportPatch: %v", err)
	}
	if asCommits {
		t.Error("asCommits = true, want a plain diff applied to the working tree")
	}
	if content, _ := os.ReadFile(filepath.Join(dir, "file.txt")); string(content) != "v2\n" {
		t.Errorf("file.txt = %q, want the patched content", content)
	}
	if got := gittest.Run(t, dir, "rev-parse", "HEAD"); got != head {
		t.Error("HEAD moved; a plain diff mustn't make a commit")
	}
}

func TestImportPatchMakesCommitsFromAMailboxPatch(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1\n", "initial")
	gittest.CommitFile(t, dir, "file.txt", "v2\n", "add v2 line")
	patch, err := FormatPatch(context.Background(), dir, "HEAD")
	if err != nil {
		t.Fatalf("FormatPatch: %v", err)
	}
	patchPath := filepath.Join(t.TempDir(), "commit.patch")
	if err := os.WriteFile(patchPath, []byte(patch), 0o644); err != nil {
		t.Fatal(err)
	}
	gittest.Run(t, dir, "reset", "-q", "--hard", "HEAD~1")

	asCommits, err := ImportPatch(context.Background(), dir, patchPath)
	if err != nil {
		t.Fatalf("ImportPatch: %v", err)
	}
	if !asCommits {
		t.Error("asCommits = false, want git am to have made a commit")
	}
	if subject := gittest.Run(t, dir, "log", "-1", "--format=%s"); subject != "add v2 line" {
		t.Errorf("HEAD subject = %q, want the patch's commit", subject)
	}
}

func TestApplyPatchConflictLeavesConflictState(t *testing.T) {
	dir, patchPath := initAmConflictRepo(t)

	if err := ApplyPatch(context.Background(), dir, patchPath); err == nil {
		t.Fatal("expected an error applying a conflicting patch")
	}

	state, err := DetectConflictState(context.Background(), dir)
	if err != nil {
		t.Fatalf("DetectConflictState: %v", err)
	}
	if state.Operation != ConflictAm {
		t.Fatalf("Operation = %q, want %q", state.Operation, ConflictAm)
	}
}
