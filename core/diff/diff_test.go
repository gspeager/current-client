package diff

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestParseUnifiedDiffModified(t *testing.T) {
	const fixture = `diff --git a/file.txt b/file.txt
index 83db48f..bf269c4 100644
--- a/file.txt
+++ b/file.txt
@@ -1,4 +1,5 @@
 unchanged line
-removed line
+added line
+another added line
 unchanged line 2
`

	got, err := ParseUnifiedDiff(fixture)
	if err != nil {
		t.Fatalf("ParseUnifiedDiff: %v", err)
	}

	want := FileDiff{
		OldPath: "file.txt",
		NewPath: "file.txt",
		Hunks: []Hunk{
			{
				Header:   "@@ -1,4 +1,5 @@",
				OldStart: 1,
				OldLines: 4,
				NewStart: 1,
				NewLines: 5,
				Lines: []Line{
					{Kind: LineContext, OldLine: 1, NewLine: 1, Content: "unchanged line"},
					{Kind: LineRemoved, OldLine: 2, Content: "removed line"},
					{Kind: LineAdded, NewLine: 2, Content: "added line"},
					{Kind: LineAdded, NewLine: 3, Content: "another added line"},
					{Kind: LineContext, OldLine: 3, NewLine: 4, Content: "unchanged line 2"},
				},
				Raw: "@@ -1,4 +1,5 @@\n unchanged line\n-removed line\n+added line\n+another added line\n unchanged line 2\n",
			},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestParseUnifiedDiffMultipleHunks(t *testing.T) {
	const fixture = `diff --git a/file.txt b/file.txt
index 83db48f..bf269c4 100644
--- a/file.txt
+++ b/file.txt
@@ -1,2 +1,2 @@
-old top
+new top
 context
@@ -10,2 +10,3 @@
 context
+new bottom
`

	got, err := ParseUnifiedDiff(fixture)
	if err != nil {
		t.Fatalf("ParseUnifiedDiff: %v", err)
	}
	if len(got.Hunks) != 2 {
		t.Fatalf("got %d hunks, want 2", len(got.Hunks))
	}
	if got.Hunks[1].OldStart != 10 || got.Hunks[1].NewStart != 10 {
		t.Fatalf("second hunk = %+v, want OldStart/NewStart 10", got.Hunks[1])
	}
}

func TestParseUnifiedDiffNoNewlineAtEOF(t *testing.T) {
	const fixture = `diff --git a/file.txt b/file.txt
index 83db48f..bf269c4 100644
--- a/file.txt
+++ b/file.txt
@@ -1 +1 @@
-old
\ No newline at end of file
+new
\ No newline at end of file
`

	got, err := ParseUnifiedDiff(fixture)
	if err != nil {
		t.Fatalf("ParseUnifiedDiff: %v", err)
	}
	want := []Line{
		{Kind: LineRemoved, OldLine: 1, Content: "old"},
		{Kind: LineAdded, NewLine: 1, Content: "new"},
	}
	if len(got.Hunks) != 1 || !reflect.DeepEqual(got.Hunks[0].Lines, want) {
		t.Fatalf("got %+v, want lines %+v", got, want)
	}
}

func TestParseUnifiedDiffEmpty(t *testing.T) {
	got, err := ParseUnifiedDiff("")
	if err != nil {
		t.Fatalf("ParseUnifiedDiff: %v", err)
	}
	if !reflect.DeepEqual(got, FileDiff{}) {
		t.Fatalf("got %+v, want empty FileDiff", got)
	}
}

func TestParseUnifiedDiffBinary(t *testing.T) {
	const fixture = `diff --git a/image.png b/image.png
index 83db48f..bf269c4 100644
Binary files a/image.png and b/image.png differ
`
	got, err := ParseUnifiedDiff(fixture)
	if err != nil {
		t.Fatalf("ParseUnifiedDiff: %v", err)
	}
	if !got.Binary {
		t.Fatal("Binary = false, want true")
	}
	if len(got.Hunks) != 0 {
		t.Fatalf("got %d hunks for a binary file, want 0", len(got.Hunks))
	}
}

func TestParseUnifiedDiffMalformedHunkHeader(t *testing.T) {
	const fixture = `diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ not a real header @@
 context
`
	if _, err := ParseUnifiedDiff(fixture); err == nil {
		t.Fatal("expected error for malformed hunk header, got nil")
	}
}

func TestGetWorkingTreeDiffRealRepo(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")

	gittest.WriteFile(t, dir, "file.txt", "line one\nline two\nline three\n")
	gittest.Run(t, dir, "add", "file.txt")
	gittest.Run(t, dir, "commit", "-m", "initial")

	gittest.WriteFile(t, dir, "file.txt", "line one\nline TWO\nline three\n")

	got, err := GetWorkingTreeDiff(context.Background(), dir, "file.txt", false, false)
	if err != nil {
		t.Fatalf("GetWorkingTreeDiff: %v", err)
	}
	if len(got.Hunks) != 1 {
		t.Fatalf("got %d hunks, want 1: %+v", len(got.Hunks), got)
	}

	var added, removed []string
	for _, l := range got.Hunks[0].Lines {
		switch l.Kind {
		case LineAdded:
			added = append(added, l.Content)
		case LineRemoved:
			removed = append(removed, l.Content)
		}
	}
	if len(added) != 1 || added[0] != "line TWO" {
		t.Fatalf("added lines = %+v, want [\"line TWO\"]", added)
	}
	if len(removed) != 1 || removed[0] != "line two" {
		t.Fatalf("removed lines = %+v, want [\"line two\"]", removed)
	}
}

func TestGetWorkingTreeDiffBinaryFile(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")

	writeBytes(t, dir, "image.png", []byte{0x89, 0x50, 0x4e, 0x47, 0x00, 0x01, 0x02})
	gittest.Run(t, dir, "add", "image.png")
	gittest.Run(t, dir, "commit", "-m", "initial")

	writeBytes(t, dir, "image.png", []byte{0x89, 0x50, 0x4e, 0x47, 0xff, 0xfe, 0xfd})

	got, err := GetWorkingTreeDiff(context.Background(), dir, "image.png", false, false)
	if err != nil {
		t.Fatalf("GetWorkingTreeDiff: %v", err)
	}
	if !got.Binary {
		t.Fatal("Binary = false, want true")
	}
	if len(got.Hunks) != 0 {
		t.Fatalf("got %d hunks for a binary file, want 0", len(got.Hunks))
	}
}

func TestGetWorkingTreeDiffLargeFile(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")

	line := strings.Repeat("x", 79) + "\n"
	var big strings.Builder
	for big.Len() <= LargeFileThreshold {
		big.WriteString(line)
	}
	gittest.WriteFile(t, dir, "big.txt", big.String())
	gittest.Run(t, dir, "add", "big.txt")
	gittest.Run(t, dir, "commit", "-m", "initial")

	gittest.WriteFile(t, dir, "big.txt", big.String()+line)

	got, err := GetWorkingTreeDiff(context.Background(), dir, "big.txt", false, false)
	if err != nil {
		t.Fatalf("GetWorkingTreeDiff: %v", err)
	}
	if !got.TooLarge {
		t.Fatal("TooLarge = false, want true")
	}
	if got.SizeBytes <= LargeFileThreshold {
		t.Fatalf("SizeBytes = %d, want > %d", got.SizeBytes, LargeFileThreshold)
	}
	if len(got.Hunks) != 0 {
		t.Fatalf("got %d hunks for a file over the size guard, want 0", len(got.Hunks))
	}

	forced, err := GetWorkingTreeDiff(context.Background(), dir, "big.txt", true, false)
	if err != nil {
		t.Fatalf("GetWorkingTreeDiff (forced): %v", err)
	}
	if forced.TooLarge {
		t.Fatal("TooLarge = true with force=true, want false")
	}
	if len(forced.Hunks) == 0 {
		t.Fatal("forced load produced no hunks, want the real diff")
	}
}

func TestGetWorkingTreeDiffIgnoreWhitespace(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")

	gittest.WriteFile(t, dir, "file.txt", "line one\nline two\nline three\n")
	gittest.Run(t, dir, "add", "file.txt")
	gittest.Run(t, dir, "commit", "-m", "initial")

	gittest.WriteFile(t, dir, "file.txt", "line one\nline two   \nline three\n") // trailing whitespace only

	withSpace, err := GetWorkingTreeDiff(context.Background(), dir, "file.txt", false, false)
	if err != nil {
		t.Fatalf("GetWorkingTreeDiff: %v", err)
	}
	if len(withSpace.Hunks) == 0 {
		t.Fatal("expected the whitespace-only change to show up by default")
	}

	ignored, err := GetWorkingTreeDiff(context.Background(), dir, "file.txt", false, true)
	if err != nil {
		t.Fatalf("GetWorkingTreeDiff (ignoreWhitespace): %v", err)
	}
	if len(ignored.Hunks) != 0 {
		t.Fatalf("got %d hunks with ignoreWhitespace on a whitespace-only change, want 0: %+v", len(ignored.Hunks), ignored)
	}
}

func TestGetWorkingTreeDiffNoChanges(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")

	gittest.WriteFile(t, dir, "file.txt", "unchanged\n")
	gittest.Run(t, dir, "add", "file.txt")
	gittest.Run(t, dir, "commit", "-m", "initial")

	got, err := GetWorkingTreeDiff(context.Background(), dir, "file.txt", false, false)
	if err != nil {
		t.Fatalf("GetWorkingTreeDiff: %v", err)
	}
	if len(got.Hunks) != 0 {
		t.Fatalf("got %d hunks, want 0", len(got.Hunks))
	}
}

func TestStageHunkStagesOnlyThatHunk(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")

	var initial strings.Builder
	for i := 1; i <= 60; i++ {
		fmt.Fprintf(&initial, "line %d\n", i)
	}
	gittest.WriteFile(t, dir, "file.txt", initial.String())
	gittest.Run(t, dir, "add", "file.txt")
	gittest.Run(t, dir, "commit", "-m", "initial")

	lines := strings.Split(strings.TrimSuffix(initial.String(), "\n"), "\n")
	lines[4] = "line 5 CHANGED"
	lines[54] = "line 55 CHANGED"
	gittest.WriteFile(t, dir, "file.txt", strings.Join(lines, "\n")+"\n")

	fd, err := GetWorkingTreeDiff(context.Background(), dir, "file.txt", false, false)
	if err != nil {
		t.Fatalf("GetWorkingTreeDiff: %v", err)
	}
	if len(fd.Hunks) != 2 {
		t.Fatalf("got %d hunks, want 2 (test setup assumption broken): %+v", len(fd.Hunks), fd.Hunks)
	}

	if err := StageHunk(context.Background(), dir, "file.txt", fd.Hunks[0].Raw); err != nil {
		t.Fatalf("StageHunk: %v", err)
	}

	staged := runGitOutput(t, dir, "diff", "--cached")
	if !strings.Contains(staged, "line 5 CHANGED") {
		t.Fatalf("staged diff missing the staged hunk's change:\n%s", staged)
	}
	if strings.Contains(staged, "line 55 CHANGED") {
		t.Fatalf("staged diff contains the other hunk's change, want only one staged:\n%s", staged)
	}

	unstaged := runGitOutput(t, dir, "diff")
	if strings.Contains(unstaged, "line 5 CHANGED") {
		t.Fatalf("working tree diff still shows the staged hunk's change:\n%s", unstaged)
	}
	if !strings.Contains(unstaged, "line 55 CHANGED") {
		t.Fatalf("working tree diff missing the still-unstaged hunk's change:\n%s", unstaged)
	}

	if err := UnstageHunk(context.Background(), dir, "file.txt", fd.Hunks[0].Raw); err != nil {
		t.Fatalf("UnstageHunk: %v", err)
	}
	if staged := runGitOutput(t, dir, "diff", "--cached"); strings.TrimSpace(staged) != "" {
		t.Fatalf("expected nothing staged after UnstageHunk, got:\n%s", staged)
	}
}

func TestDiscardHunkRevertsOnlyThatHunkInTheWorkingTree(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")

	var initial strings.Builder
	for i := 1; i <= 60; i++ {
		fmt.Fprintf(&initial, "line %d\n", i)
	}
	gittest.WriteFile(t, dir, "file.txt", initial.String())
	gittest.Run(t, dir, "add", "file.txt")
	gittest.Run(t, dir, "commit", "-m", "initial")

	lines := strings.Split(strings.TrimSuffix(initial.String(), "\n"), "\n")
	lines[4] = "line 5 CHANGED"
	lines[54] = "line 55 CHANGED"
	gittest.WriteFile(t, dir, "file.txt", strings.Join(lines, "\n")+"\n")

	fd, err := GetWorkingTreeDiff(context.Background(), dir, "file.txt", false, false)
	if err != nil {
		t.Fatalf("GetWorkingTreeDiff: %v", err)
	}
	if len(fd.Hunks) != 2 {
		t.Fatalf("got %d hunks, want 2 (test setup assumption broken): %+v", len(fd.Hunks), fd.Hunks)
	}

	if err := DiscardHunk(context.Background(), dir, "file.txt", fd.Hunks[0].Raw); err != nil {
		t.Fatalf("DiscardHunk: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "file.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(content), "line 5 CHANGED") {
		t.Fatalf("file still contains the discarded hunk's change:\n%s", content)
	}
	if !strings.Contains(string(content), "line 55 CHANGED") {
		t.Fatalf("file lost the other hunk's change, want it untouched:\n%s", content)
	}
	if staged := runGitOutput(t, dir, "diff", "--cached"); strings.TrimSpace(staged) != "" {
		t.Fatalf("DiscardHunk touched the index, want it left alone:\n%s", staged)
	}
}

func TestGetIndexDiffShowsOnlyStagedChanges(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")

	gittest.WriteFile(t, dir, "file.txt", "line one\nline two\nline three\n")
	gittest.Run(t, dir, "add", "file.txt")
	gittest.Run(t, dir, "commit", "-m", "initial")

	gittest.WriteFile(t, dir, "file.txt", "line one\nline TWO STAGED\nline three\n")
	gittest.Run(t, dir, "add", "file.txt")
	gittest.WriteFile(t, dir, "file.txt", "line one\nline TWO STAGED\nline three UNSTAGED\n")

	indexDiff, err := GetIndexDiff(context.Background(), dir, "file.txt", false)
	if err != nil {
		t.Fatalf("GetIndexDiff: %v", err)
	}
	if len(indexDiff.Hunks) != 1 {
		t.Fatalf("got %d hunks, want 1: %+v", len(indexDiff.Hunks), indexDiff)
	}
	if !hasAddedLine(indexDiff, "line TWO STAGED") {
		t.Fatalf("index diff missing staged change: %+v", indexDiff)
	}
	if hasAddedLine(indexDiff, "line three UNSTAGED") {
		t.Fatalf("index diff includes an unstaged change: %+v", indexDiff)
	}

	workingDiff, err := GetWorkingTreeDiff(context.Background(), dir, "file.txt", false, false)
	if err != nil {
		t.Fatalf("GetWorkingTreeDiff: %v", err)
	}
	if hasAddedLine(workingDiff, "line TWO STAGED") {
		t.Fatalf("working tree diff includes an already-staged change: %+v", workingDiff)
	}
	if !hasAddedLine(workingDiff, "line three UNSTAGED") {
		t.Fatalf("working tree diff missing the unstaged change: %+v", workingDiff)
	}
}

func TestGetRefDiffWorkingVsHead(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")

	gittest.WriteFile(t, dir, "file.txt", "line one\nline two\nline three\n")
	gittest.Run(t, dir, "add", "file.txt")
	gittest.Run(t, dir, "commit", "-m", "initial")

	gittest.WriteFile(t, dir, "file.txt", "line one\nline TWO STAGED\nline three\n")
	gittest.Run(t, dir, "add", "file.txt")
	gittest.WriteFile(t, dir, "file.txt", "line one\nline TWO STAGED\nline three UNSTAGED\n")

	got, err := GetRefDiff(context.Background(), dir, "file.txt", "HEAD", "", false)
	if err != nil {
		t.Fatalf("GetRefDiff: %v", err)
	}
	if !hasAddedLine(got, "line TWO STAGED") {
		t.Fatalf("working-vs-HEAD diff missing the staged change: %+v", got)
	}
	if !hasAddedLine(got, "line three UNSTAGED") {
		t.Fatalf("working-vs-HEAD diff missing the unstaged change: %+v", got)
	}
}

func TestGetRefDiffCommitToCommit(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")

	gittest.WriteFile(t, dir, "file.txt", "version one\n")
	gittest.Run(t, dir, "add", "file.txt")
	gittest.Run(t, dir, "commit", "-m", "first")
	firstSHA := strings.TrimSpace(runGitOutput(t, dir, "rev-parse", "HEAD"))

	gittest.WriteFile(t, dir, "file.txt", "version two\n")
	gittest.Run(t, dir, "add", "file.txt")
	gittest.Run(t, dir, "commit", "-m", "second")
	secondSHA := strings.TrimSpace(runGitOutput(t, dir, "rev-parse", "HEAD"))

	got, err := GetRefDiff(context.Background(), dir, "file.txt", firstSHA, secondSHA, false)
	if err != nil {
		t.Fatalf("GetRefDiff: %v", err)
	}
	if !hasRemovedLine(got, "version one") || !hasAddedLine(got, "version two") {
		t.Fatalf("commit-to-commit diff missing the expected change: %+v", got)
	}
}

func TestGetRefDiffBranchToBranch(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init", "-b", "main") // explicit initial branch name; default varies by git config
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")

	gittest.WriteFile(t, dir, "file.txt", "main content\n")
	gittest.Run(t, dir, "add", "file.txt")
	gittest.Run(t, dir, "commit", "-m", "initial")
	gittest.Run(t, dir, "branch", "feature")
	gittest.Run(t, dir, "checkout", "feature")
	gittest.WriteFile(t, dir, "file.txt", "feature content\n")
	gittest.Run(t, dir, "add", "file.txt")
	gittest.Run(t, dir, "commit", "-m", "feature change")
	gittest.Run(t, dir, "checkout", "main")

	got, err := GetRefDiff(context.Background(), dir, "file.txt", "main", "feature", false)
	if err != nil {
		t.Fatalf("GetRefDiff: %v", err)
	}
	if !hasRemovedLine(got, "main content") || !hasAddedLine(got, "feature content") {
		t.Fatalf("branch-to-branch diff missing the expected change: %+v", got)
	}
}

func TestGetRefDiffTagToTag(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")

	gittest.WriteFile(t, dir, "file.txt", "v1 content\n")
	gittest.Run(t, dir, "add", "file.txt")
	gittest.Run(t, dir, "commit", "-m", "v1")
	gittest.Run(t, dir, "tag", "v1")

	gittest.WriteFile(t, dir, "file.txt", "v2 content\n")
	gittest.Run(t, dir, "add", "file.txt")
	gittest.Run(t, dir, "commit", "-m", "v2")
	gittest.Run(t, dir, "tag", "v2")

	got, err := GetRefDiff(context.Background(), dir, "file.txt", "v1", "v2", false)
	if err != nil {
		t.Fatalf("GetRefDiff: %v", err)
	}
	if !hasRemovedLine(got, "v1 content") || !hasAddedLine(got, "v2 content") {
		t.Fatalf("tag-to-tag diff missing the expected change: %+v", got)
	}
}

func hasAddedLine(fd FileDiff, content string) bool {
	for _, h := range fd.Hunks {
		for _, l := range h.Lines {
			if l.Kind == LineAdded && l.Content == content {
				return true
			}
		}
	}
	return false
}

func hasRemovedLine(fd FileDiff, content string) bool {
	for _, h := range fd.Hunks {
		for _, l := range h.Lines {
			if l.Kind == LineRemoved && l.Content == content {
				return true
			}
		}
	}
	return false
}

func runGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(out)
}

func writeBytes(t *testing.T, dir, name string, content []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), content, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestParseUnifiedDiffKeepsEachHunksRawTextSeparate(t *testing.T) {
	first := "@@ -1,2 +1,2 @@\n-a\n+A\n b\n"
	second := "@@ -10,1 +10,1 @@\n-y\n+Y\n\\ No newline at end of file\n"
	got, err := ParseUnifiedDiff("diff --git a/f b/f\n--- a/f\n+++ b/f\n" + first + second)
	if err != nil {
		t.Fatalf("ParseUnifiedDiff: %v", err)
	}
	if len(got.Hunks) != 2 {
		t.Fatalf("got %d hunks, want 2", len(got.Hunks))
	}
	if got.Hunks[0].Raw != first || got.Hunks[1].Raw != second {
		t.Fatalf("raw = %q / %q, want %q / %q", got.Hunks[0].Raw, got.Hunks[1].Raw, first, second)
	}
}

func stagedTwoHunkRepo(t *testing.T) (string, []string, FileDiff) {
	t.Helper()
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	var initial strings.Builder
	for i := 1; i <= 60; i++ {
		fmt.Fprintf(&initial, "line %d\n", i)
	}
	gittest.WriteFile(t, dir, "file.txt", initial.String())
	gittest.Run(t, dir, "add", "file.txt")
	gittest.Run(t, dir, "commit", "-m", "initial")

	lines := strings.Split(strings.TrimSuffix(initial.String(), "\n"), "\n")
	lines[4] = "line 5 CHANGED"
	lines[54] = "line 55 CHANGED"
	gittest.WriteFile(t, dir, "file.txt", strings.Join(lines, "\n")+"\n")
	gittest.Run(t, dir, "add", "file.txt")

	fd, err := GetIndexDiff(context.Background(), dir, "file.txt", false)
	if err != nil || len(fd.Hunks) != 2 {
		t.Fatalf("GetIndexDiff = %d hunks, %v; want 2 (test setup assumption broken)", len(fd.Hunks), err)
	}
	return dir, lines, fd
}

func TestDiscardStagedHunkRevertsIndexAndWorkingTree(t *testing.T) {
	dir, _, fd := stagedTwoHunkRepo(t)

	if err := DiscardStagedHunk(context.Background(), dir, "file.txt", fd.Hunks[0].Raw); err != nil {
		t.Fatalf("DiscardStagedHunk: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "file.txt"))
	staged := runGitOutput(t, dir, "diff", "--cached")
	for name, text := range map[string]string{"working tree": string(content), "index diff": staged} {
		if strings.Contains(text, "line 5 CHANGED") {
			t.Errorf("%s still has the discarded hunk:\n%s", name, text)
		}
		if !strings.Contains(text, "line 55 CHANGED") {
			t.Errorf("%s lost the other hunk:\n%s", name, text)
		}
	}
}

func TestDiscardStagedHunkRefusesWhenLaterEditsOverlap(t *testing.T) {
	dir, lines, fd := stagedTwoHunkRepo(t)
	lines[4] = "line 5 EDITED AGAIN"
	gittest.WriteFile(t, dir, "file.txt", strings.Join(lines, "\n")+"\n")

	if err := DiscardStagedHunk(context.Background(), dir, "file.txt", fd.Hunks[0].Raw); err == nil {
		t.Fatal("DiscardStagedHunk succeeded over an unstaged edit to the same lines")
	}
	content, _ := os.ReadFile(filepath.Join(dir, "file.txt"))
	if !strings.Contains(string(content), "line 5 EDITED AGAIN") || !strings.Contains(runGitOutput(t, dir, "diff", "--cached"), "line 5 CHANGED") {
		t.Error("a failed discard changed the working tree or the index")
	}
}

func TestWorkingTreeDiffShowsAnUntrackedFileAsAdded(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "committed.txt", "x\n", "initial")
	if err := os.MkdirAll(filepath.Join(dir, "new dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	gittest.WriteFile(t, dir, "new dir/f.txt", "a\nb\n")

	fd, err := GetWorkingTreeDiff(context.Background(), dir, "new dir/f.txt", false, false)
	if err != nil {
		t.Fatalf("GetWorkingTreeDiff: %v", err)
	}
	if fd.NewPath != "new dir/f.txt" || len(fd.Hunks) != 1 {
		t.Fatalf("diff = %+v, want one hunk for the new file", fd)
	}
	var added []string
	for _, l := range fd.Hunks[0].Lines {
		if l.Kind == LineAdded {
			added = append(added, l.Content)
		}
	}
	if strings.Join(added, ",") != "a,b" {
		t.Fatalf("added lines = %q, want every line of the file", added)
	}
}

func TestWorkingTreeDiffOfAnIgnoredFileIsEmpty(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, ".gitignore", "*.log\n", "initial")
	gittest.WriteFile(t, dir, "debug.log", "noise\n")

	fd, err := GetWorkingTreeDiff(context.Background(), dir, "debug.log", false, false)
	if err != nil || len(fd.Hunks) != 0 {
		t.Fatalf("diff = %+v, %v; want nothing for an ignored file", fd, err)
	}
}

func TestRenamedIndexDiffShowsOnlyTheEdit(t *testing.T) {
	dir := gittest.InitRepo(t)
	var content strings.Builder
	for i := 1; i <= 20; i++ {
		fmt.Fprintf(&content, "line %d\n", i)
	}
	gittest.CommitFile(t, dir, "old.txt", content.String(), "initial")
	gittest.Run(t, dir, "mv", "old.txt", "new.txt")
	gittest.WriteFile(t, dir, "new.txt", strings.Replace(content.String(), "line 10\n", "line ten\n", 1))
	gittest.Run(t, dir, "add", "new.txt")

	fd, err := GetRenamedIndexDiff(context.Background(), dir, "old.txt", "new.txt", false)
	if err != nil {
		t.Fatalf("GetRenamedIndexDiff: %v", err)
	}
	if fd.OldPath != "old.txt" || fd.NewPath != "new.txt" {
		t.Fatalf("paths = %q → %q, want the rename", fd.OldPath, fd.NewPath)
	}
	added := 0
	for _, h := range fd.Hunks {
		for _, l := range h.Lines {
			if l.Kind == LineAdded {
				added++
			}
		}
	}
	if added != 1 {
		t.Fatalf("added %d lines, want just the edited one", added)
	}
}

func TestRevisionFileSize(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "f.txt", "12345", "initial")
	gittest.WriteFile(t, dir, "f.txt", "1234567")
	gittest.Run(t, dir, "add", "f.txt")

	if size, found := RevisionFileSize(context.Background(), dir, "HEAD", "f.txt"); !found || size != 5 {
		t.Errorf("HEAD size = %d, %v; want 5", size, found)
	}
	if size, found := RevisionFileSize(context.Background(), dir, "", "f.txt"); !found || size != 7 {
		t.Errorf("index size = %d, %v; want 7", size, found)
	}
	if _, found := RevisionFileSize(context.Background(), dir, "HEAD", "missing.txt"); found {
		t.Error("found a file that doesn't exist")
	}
}
