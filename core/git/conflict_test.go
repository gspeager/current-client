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

// initConflictRepo leaves main and "theirs" editing the same line.
func initConflictRepo(t *testing.T) (dir string) {
	t.Helper()
	dir = gittest.InitRepo(t)

	gittest.CommitFile(t, dir, "file.txt", "base\n", "initial")

	gittest.Run(t, dir, "checkout", "-q", "-b", "theirs")
	gittest.CommitFile(t, dir, "file.txt", "theirs\n", "theirs change")

	gittest.Run(t, dir, "checkout", "-q", "main")
	gittest.CommitFile(t, dir, "file.txt", "ours\n", "ours change")

	return dir
}

func runGitExpectFailure(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err == nil {
		t.Fatalf("git %v: expected failure, succeeded:\n%s", args, out)
	}
}

func TestGetStatusReportsMergeConflict(t *testing.T) {
	dir := initConflictRepo(t)
	runGitExpectFailure(t, dir, "merge", "theirs")

	statuses, err := GetStatus(context.Background(), dir)
	if err != nil {
		t.Fatalf("GetStatus on a conflicted repo: %v", err)
	}

	var conflicted *Status
	for i := range statuses {
		if statuses[i].Path == "file.txt" {
			conflicted = &statuses[i]
		}
	}
	if conflicted == nil {
		t.Fatalf("GetStatus = %+v, want file.txt reported as unmerged", statuses)
	}
	// Both sides modified the file, which git reports as "UU".
	if conflicted.IndexStatus != 'U' || conflicted.WorktreeStatus != 'U' {
		t.Fatalf("file.txt status = %+v, want UU (both-modified conflict)", conflicted)
	}
	if !conflicted.Conflicted {
		t.Fatalf("file.txt Conflicted = false, want true for a UU entry")
	}

	content, err := os.ReadFile(filepath.Join(dir, "file.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(content), "<<<<<<<") {
		t.Fatalf("file.txt = %q, want conflict markers left in the working tree", content)
	}
}

func TestCheckoutBranchFailsCleanlyWhenWorkingTreeConflicts(t *testing.T) {
	dir := gittest.InitRepo(t)

	gittest.CommitFile(t, dir, "file.txt", "on main\n", "initial")
	gittest.Run(t, dir, "checkout", "-q", "-b", "feature")
	gittest.CommitFile(t, dir, "file.txt", "on feature\n", "feature change")
	gittest.Run(t, dir, "checkout", "-q", "main")

	// The target branch.s version would overwrite this uncommitted edit.
	gittest.WriteFile(t, dir, "file.txt", "uncommitted, conflicting\n")

	if err := CheckoutBranch(context.Background(), dir, "feature"); err == nil {
		t.Fatalf("CheckoutBranch succeeded, want it to refuse and leave the dirty file alone")
	}

	content, err := os.ReadFile(filepath.Join(dir, "file.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "uncommitted, conflicting\n" {
		t.Fatalf("file.txt = %q, want the failed checkout to leave the uncommitted edit untouched", content)
	}

	branches, err := ListBranches(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	for _, b := range branches {
		if b.Name == "main" && !b.Current {
			t.Fatalf("branches = %+v, want HEAD to remain on main after the failed checkout", branches)
		}
	}
}

func TestCheckoutBranchCarriesNonConflictingChanges(t *testing.T) {
	dir := gittest.InitRepo(t)

	gittest.WriteFile(t, dir, "file.txt", "shared\n")
	gittest.WriteFile(t, dir, "other.txt", "shared\n")
	gittest.Run(t, dir, "add", ".")
	gittest.Run(t, dir, "commit", "-m", "initial")
	gittest.Run(t, dir, "checkout", "-q", "-b", "feature")
	gittest.CommitFile(t, dir, "file.txt", "changed on feature\n", "feature change")
	gittest.Run(t, dir, "checkout", "-q", "main")

	// The target branch never touches this file, so git carries the edit across.
	gittest.WriteFile(t, dir, "other.txt", "uncommitted, unrelated\n")

	if err := CheckoutBranch(context.Background(), dir, "feature"); err != nil {
		t.Fatalf("CheckoutBranch: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "other.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "uncommitted, unrelated\n" {
		t.Fatalf("other.txt = %q, want the uncommitted edit carried across the checkout", content)
	}
}

func TestDetectConflictStateNoOperationInProgress(t *testing.T) {
	dir := initConflictRepo(t)

	state, err := DetectConflictState(context.Background(), dir)
	if err != nil {
		t.Fatalf("DetectConflictState: %v", err)
	}
	if state.Operation != ConflictNone {
		t.Fatalf("got %+v, want no operation in progress", state)
	}
}

func TestDetectConflictStateDuringMerge(t *testing.T) {
	dir := initConflictRepo(t)
	runGitExpectFailure(t, dir, "merge", "theirs")

	state, err := DetectConflictState(context.Background(), dir)
	if err != nil {
		t.Fatalf("DetectConflictState: %v", err)
	}
	if state.Operation != ConflictMerge {
		t.Fatalf("Operation = %q, want %q", state.Operation, ConflictMerge)
	}
	if len(state.ConflictedPaths) != 1 || state.ConflictedPaths[0] != "file.txt" {
		t.Fatalf("ConflictedPaths = %v, want [file.txt]", state.ConflictedPaths)
	}
}

func TestDetectConflictStateDuringCherryPick(t *testing.T) {
	dir := initConflictRepo(t)
	theirsSHA := gittest.Run(t, dir, "rev-parse", "theirs")
	runGitExpectFailure(t, dir, "cherry-pick", theirsSHA)

	state, err := DetectConflictState(context.Background(), dir)
	if err != nil {
		t.Fatalf("DetectConflictState: %v", err)
	}
	if state.Operation != ConflictCherryPick {
		t.Fatalf("Operation = %q, want %q", state.Operation, ConflictCherryPick)
	}
}

func TestDetectConflictStateDuringRevert(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1\n", "v1")
	gittest.CommitFile(t, dir, "file.txt", "v2\n", "v2")
	gittest.CommitFile(t, dir, "file.txt", "v3\n", "v3")

	// Reverting "v2" conflicts: HEAD (v3) no longer matches what the revert
	// of v2 expects to find.
	v2SHA := gittest.Run(t, dir, "rev-parse", "HEAD~1")
	runGitExpectFailure(t, dir, "revert", "--no-edit", v2SHA)

	state, err := DetectConflictState(context.Background(), dir)
	if err != nil {
		t.Fatalf("DetectConflictState: %v", err)
	}
	if state.Operation != ConflictRevert {
		t.Fatalf("Operation = %q, want %q", state.Operation, ConflictRevert)
	}
}

func TestDetectConflictStateDuringRebase(t *testing.T) {
	dir := initConflictRepo(t)
	runGitExpectFailure(t, dir, "rebase", "theirs")

	state, err := DetectConflictState(context.Background(), dir)
	if err != nil {
		t.Fatalf("DetectConflictState: %v", err)
	}
	if state.Operation != ConflictRebase {
		t.Fatalf("Operation = %q, want %q", state.Operation, ConflictRebase)
	}
}

func TestAbortConflictOperationMergeReturnsToPreConflictState(t *testing.T) {
	dir := initConflictRepo(t)
	runGitExpectFailure(t, dir, "merge", "theirs")

	if err := AbortConflictOperation(context.Background(), dir, ConflictMerge); err != nil {
		t.Fatalf("AbortConflictOperation: %v", err)
	}

	state, err := DetectConflictState(context.Background(), dir)
	if err != nil {
		t.Fatalf("DetectConflictState: %v", err)
	}
	if state.Operation != ConflictNone {
		t.Fatalf("got %+v, want no operation in progress after abort", state)
	}
	content, err := os.ReadFile(filepath.Join(dir, "file.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "ours\n" {
		t.Fatalf("file.txt = %q, want the pre-merge content restored", content)
	}
}

func TestContinueConflictOperationMergeCompletesAfterResolution(t *testing.T) {
	dir := initConflictRepo(t)
	runGitExpectFailure(t, dir, "merge", "theirs")

	gittest.WriteFile(t, dir, "file.txt", "resolved\n")
	gittest.Run(t, dir, "add", "file.txt")

	if err := ContinueConflictOperation(context.Background(), dir, ConflictMerge); err != nil {
		t.Fatalf("ContinueConflictOperation: %v", err)
	}

	state, err := DetectConflictState(context.Background(), dir)
	if err != nil {
		t.Fatalf("DetectConflictState: %v", err)
	}
	if state.Operation != ConflictNone {
		t.Fatalf("got %+v, want no operation in progress after continue", state)
	}
	content, err := os.ReadFile(filepath.Join(dir, "file.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "resolved\n" {
		t.Fatalf("file.txt = %q, want the resolved content committed", content)
	}
}

func TestContinueConflictOperationFailsWhileConflictsRemain(t *testing.T) {
	dir := initConflictRepo(t)
	runGitExpectFailure(t, dir, "merge", "theirs")

	if err := ContinueConflictOperation(context.Background(), dir, ConflictMerge); err == nil {
		t.Fatal("expected an error continuing with unresolved conflicts still staged as UU")
	}
}

func TestContinueConflictOperationNoOperationInProgress(t *testing.T) {
	dir := initConflictRepo(t)
	if err := ContinueConflictOperation(context.Background(), dir, ConflictNone); err == nil {
		t.Fatal("expected an error continuing with no operation in progress")
	}
}

func TestAbortConflictOperationNoOperationInProgress(t *testing.T) {
	dir := initConflictRepo(t)
	if err := AbortConflictOperation(context.Background(), dir, ConflictNone); err == nil {
		t.Fatal("expected an error aborting with no operation in progress")
	}
}

// initAmConflictRepo leaves an am-style rebase-apply state rather than a
// rebase.s rebase-merge state.
func initAmConflictRepo(t *testing.T) (dir, patchPath string) {
	t.Helper()
	dir = gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "base\n", "initial")
	gittest.CommitFile(t, dir, "file.txt", "theirs\n", "theirs change")

	patchPath = filepath.Join(t.TempDir(), "theirs.patch")
	result, err := runResult(context.Background(), dir, "format-patch", "-1", "HEAD", "--stdout")
	if err != nil {
		t.Fatalf("format-patch: %v", err)
	}
	if err := os.WriteFile(patchPath, []byte(result.Stdout), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	gittest.Run(t, dir, "reset", "--hard", "HEAD~1")
	gittest.CommitFile(t, dir, "file.txt", "ours\n", "ours change")
	return dir, patchPath
}

func TestDetectConflictStateDuringAm(t *testing.T) {
	dir, patchPath := initAmConflictRepo(t)
	runGitExpectFailure(t, dir, "am", patchPath)

	state, err := DetectConflictState(context.Background(), dir)
	if err != nil {
		t.Fatalf("DetectConflictState: %v", err)
	}
	if state.Operation != ConflictAm {
		t.Fatalf("Operation = %q, want %q", state.Operation, ConflictAm)
	}
}

func TestContinueConflictOperationAmCompletesAfterResolution(t *testing.T) {
	dir, patchPath := initAmConflictRepo(t)
	runGitExpectFailure(t, dir, "am", patchPath)

	gittest.WriteFile(t, dir, "file.txt", "resolved\n")
	gittest.Run(t, dir, "add", "file.txt")

	if err := ContinueConflictOperation(context.Background(), dir, ConflictAm); err != nil {
		t.Fatalf("ContinueConflictOperation: %v", err)
	}

	state, err := DetectConflictState(context.Background(), dir)
	if err != nil {
		t.Fatalf("DetectConflictState: %v", err)
	}
	if state.Operation != ConflictNone {
		t.Fatalf("got %+v, want no operation in progress after continue", state)
	}
}

func TestAbortConflictOperationAmReturnsToPreConflictState(t *testing.T) {
	dir, patchPath := initAmConflictRepo(t)
	runGitExpectFailure(t, dir, "am", patchPath)

	if err := AbortConflictOperation(context.Background(), dir, ConflictAm); err != nil {
		t.Fatalf("AbortConflictOperation: %v", err)
	}

	state, err := DetectConflictState(context.Background(), dir)
	if err != nil {
		t.Fatalf("DetectConflictState: %v", err)
	}
	if state.Operation != ConflictNone {
		t.Fatalf("got %+v, want no operation in progress after abort", state)
	}
	content, err := os.ReadFile(filepath.Join(dir, "file.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "ours\n" {
		t.Fatalf("file.txt = %q, want the pre-am content restored", content)
	}
}

func TestConflictSidesAreNamedByBranch(t *testing.T) {
	ctx := context.Background()
	dir := initConflictRepo(t)
	runGitExpectFailure(t, dir, "merge", "theirs")
	if got, err := ConflictSides(ctx, dir); err != nil || got != (ConflictSideNames{Ours: "main", Theirs: "theirs"}) {
		t.Fatalf("merge: sides = %+v, %v; want main and theirs", got, err)
	}
	gittest.Run(t, dir, "merge", "--abort")

	// A rebase swaps them: ours is the branch being rebased onto.
	gittest.Run(t, dir, "checkout", "-q", "theirs")
	runGitExpectFailure(t, dir, "rebase", "main")
	if got, err := ConflictSides(ctx, dir); err != nil || got != (ConflictSideNames{Ours: "main", Theirs: "theirs"}) {
		t.Fatalf("rebase: sides = %+v, %v; want main and theirs", got, err)
	}
	gittest.Run(t, dir, "rebase", "--abort")

	gittest.Run(t, dir, "checkout", "-q", "main")
	short := gittest.Run(t, dir, "rev-parse", "--short", "theirs")
	runGitExpectFailure(t, dir, "cherry-pick", "theirs")
	if got, err := ConflictSides(ctx, dir); err != nil || got != (ConflictSideNames{Ours: "main", Theirs: short}) {
		t.Fatalf("cherry-pick: sides = %+v, %v; want main and %s", got, err, short)
	}
}

func TestKeepConflictSide(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		side ConflictSide
		want string
	}{{Ours, "ours\n"}, {Theirs, "theirs\n"}} {
		dir := initConflictRepo(t)
		runGitExpectFailure(t, dir, "merge", "theirs")
		if line, err := ConflictMarkerLine(dir, "file.txt"); err != nil || line != 1 {
			t.Fatalf("marker line = %d, %v; want 1", line, err)
		}

		if err := KeepConflictSide(ctx, dir, "file.txt", tt.side); err != nil {
			t.Fatalf("KeepConflictSide %s: %v", tt.side, err)
		}
		data, _ := os.ReadFile(filepath.Join(dir, "file.txt"))
		if string(data) != tt.want {
			t.Errorf("%s: file = %q, want %q", tt.side, data, tt.want)
		}
		if status := gittest.Run(t, dir, "status", "--porcelain"); !strings.HasPrefix(status, "M  file.txt") && status != "" {
			t.Errorf("%s: status = %q, want the file staged and resolved", tt.side, status)
		}
		if line, _ := ConflictMarkerLine(dir, "file.txt"); line != 0 {
			t.Errorf("%s: marker line = %d after resolving, want 0", tt.side, line)
		}
	}
}

func TestKeepConflictSideThatDeletedTheFile(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		side   ConflictSide
		exists bool
	}{{Ours, false}, {Theirs, true}} {
		dir := gittest.InitRepo(t)
		gittest.CommitFile(t, dir, "file.txt", "base\n", "initial")
		gittest.Run(t, dir, "checkout", "-q", "-b", "theirs")
		gittest.CommitFile(t, dir, "file.txt", "theirs\n", "edit")
		gittest.Run(t, dir, "checkout", "-q", "main")
		gittest.Run(t, dir, "rm", "-q", "file.txt")
		gittest.Run(t, dir, "commit", "-q", "-m", "delete")
		runGitExpectFailure(t, dir, "merge", "theirs")

		if err := KeepConflictSide(ctx, dir, "file.txt", tt.side); err != nil {
			t.Fatalf("KeepConflictSide %s: %v", tt.side, err)
		}
		if _, err := os.Stat(filepath.Join(dir, "file.txt")); (err == nil) != tt.exists {
			t.Errorf("%s: file exists = %v, want %v", tt.side, err == nil, tt.exists)
		}
		if state, _ := DetectConflictState(ctx, dir); len(state.ConflictedPaths) != 0 {
			t.Errorf("%s: still conflicted: %v", tt.side, state.ConflictedPaths)
		}
	}
}

func TestConflictMarkerLineAfterALongLine(t *testing.T) {
	dir := t.TempDir()
	long := strings.Repeat("x", 10000)
	gittest.WriteFile(t, dir, "f.txt", long+"\nnot <<<<<<< a marker\n"+long+"<<<<<<< nor this\n<<<<<<< HEAD\nours\n")
	if line, err := ConflictMarkerLine(dir, "f.txt"); err != nil || line != 4 {
		t.Errorf("line = %d, %v; want 4", line, err)
	}
	gittest.WriteFile(t, dir, "clean.txt", "no markers")
	if line, err := ConflictMarkerLine(dir, "clean.txt"); err != nil || line != 0 {
		t.Errorf("clean: line = %d, %v; want 0", line, err)
	}
}
