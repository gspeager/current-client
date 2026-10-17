package overlap

import (
	"context"
	"reflect"
	"testing"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestPredict(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init", "-b", "main")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	gittest.Run(t, dir, "config", "core.autocrlf", "false")
	gittest.CommitFile(t, dir, "shared.txt", "base\n", "edit shared.txt")
	gittest.CommitFile(t, dir, "other.txt", "base\n", "edit other.txt")

	gittest.Run(t, dir, "switch", "-c", "conflicting")
	gittest.CommitFile(t, dir, "shared.txt", "theirs\n", "edit shared.txt")
	gittest.Run(t, dir, "switch", "-c", "unrelated", "main")
	gittest.CommitFile(t, dir, "new.txt", "new\n", "edit new.txt")
	gittest.Run(t, dir, "switch", "-c", "same-file-clean", "main")
	gittest.CommitFile(t, dir, "other.txt", "base\nappended\n", "edit other.txt")
	gittest.Run(t, dir, "branch", "merged", "main")
	gittest.Run(t, dir, "switch", "main")
	gittest.CommitFile(t, dir, "shared.txt", "ours\n", "edit shared.txt")
	gittest.Run(t, dir, "tag", "v1")

	remote := t.TempDir()
	gittest.Run(t, remote, "init", "--bare", "-b", "main")
	gittest.Run(t, dir, "remote", "add", "origin", remote)
	gittest.Run(t, dir, "push", "origin", "conflicting")

	got, err := Predict(context.Background(), dir)
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}
	want := []Overlap{
		{Branch: "conflicting", Files: []string{"shared.txt"}},
		{Branch: "origin/conflicting", Remote: true, Files: []string{"shared.txt"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Predict = %+v, want %+v", got, want)
	}

	// A second call is answered from the cache, and a moved branch is re-checked.
	gittest.Run(t, dir, "switch", "unrelated")
	gittest.CommitFile(t, dir, "shared.txt", "also theirs\n", "edit shared.txt")
	gittest.Run(t, dir, "switch", "main")
	got, err = Predict(context.Background(), dir)
	if err != nil {
		t.Fatalf("Predict after moving a branch: %v", err)
	}
	if len(got) != 3 || got[1].Branch != "unrelated" {
		t.Errorf("Predict after moving unrelated = %+v, want it to conflict too", got)
	}
}

func TestPredictEmptyRepository(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init", "-b", "main")
	if got, err := Predict(context.Background(), dir); err != nil || got != nil {
		t.Errorf("Predict = %+v, %v; want nothing for a repository with no commits", got, err)
	}
}

// Git 2.38 and 2.39 have merge-tree --write-tree but not --stdin.
func TestPredictFallsBackToOneMergeAtATime(t *testing.T) {
	mergeMany = func(context.Context, string, string, []string) ([][]string, error) {
		return nil, git.ErrMergeTreeStdinUnsupported
	}
	t.Cleanup(func() { mergeMany = git.MergeConflictsMany })

	dir := t.TempDir()
	gittest.Run(t, dir, "init", "-b", "main")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")
	gittest.Run(t, dir, "config", "core.autocrlf", "false")
	gittest.CommitFile(t, dir, "shared.txt", "base\n", "base")
	gittest.Run(t, dir, "switch", "-c", "conflicting")
	gittest.CommitFile(t, dir, "shared.txt", "theirs\n", "theirs")
	gittest.Run(t, dir, "switch", "-c", "unrelated", "main")
	gittest.CommitFile(t, dir, "new.txt", "new\n", "unrelated")
	gittest.Run(t, dir, "switch", "main")
	gittest.CommitFile(t, dir, "shared.txt", "ours\n", "ours")

	got, err := Predict(context.Background(), dir)
	want := []Overlap{{Branch: "conflicting", Files: []string{"shared.txt"}}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Predict = %+v, %v; want %+v", got, err, want)
	}
}

// An orphan branch such as gh-pages shares no history, which merge-tree
// refuses to merge; it mustn't hide the other branches' conflicts.
func TestPredictWithABranchOfUnrelatedHistory(t *testing.T) {
	for _, tt := range []struct {
		name      string
		mergeMany func(context.Context, string, string, []string) ([][]string, error)
	}{
		{"in one batch", git.MergeConflictsMany},
		{"one at a time", func(context.Context, string, string, []string) ([][]string, error) {
			return nil, git.ErrMergeTreeStdinUnsupported
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mergeMany = tt.mergeMany
			t.Cleanup(func() { mergeMany = git.MergeConflictsMany })

			dir := t.TempDir()
			gittest.Run(t, dir, "init", "-b", "main")
			gittest.Run(t, dir, "config", "user.email", "test@example.com")
			gittest.Run(t, dir, "config", "user.name", "Test")
			gittest.Run(t, dir, "config", "core.autocrlf", "false")
			gittest.CommitFile(t, dir, "shared.txt", "base\n", "base")
			gittest.Run(t, dir, "switch", "-c", "conflicting")
			gittest.CommitFile(t, dir, "shared.txt", "theirs\n", "theirs")
			gittest.Run(t, dir, "switch", "--orphan", "gh-pages")
			gittest.CommitFile(t, dir, "index.html", "site\n", "site")
			gittest.Run(t, dir, "switch", "main")
			gittest.CommitFile(t, dir, "shared.txt", "ours\n", "ours")

			got, err := Predict(context.Background(), dir)
			want := []Overlap{{Branch: "conflicting", Files: []string{"shared.txt"}}}
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Errorf("Predict = %+v, %v; want %+v", got, err, want)
			}
		})
	}
}
