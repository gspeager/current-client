package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReviewBranchesFlagsAndUndo(t *testing.T) {
	useTempSettings(t)
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	commit := func(date, message string) {
		cmd := exec.Command("git", "-C", repo, "-c", "user.name=Ada", "-c", "user.email=ada@example.com", "commit", "-q", "--allow-empty", "-m", message)
		cmd.Env = append(cmd.Environ(), "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	commit("2020-01-01T00:00:00Z", "first")
	gitIn(t, repo, "branch", "stale-one")
	commit(now, "second")
	gitIn(t, repo, "switch", "-q", "-c", "unpushed")
	if err := os.WriteFile(filepath.Join(repo, "new.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "add", "new.txt")
	commit(now, "third")
	gitIn(t, repo, "switch", "-q", "main")

	var s BranchService
	review, err := s.ReviewBranches(repo)
	if err != nil {
		t.Fatal(err)
	}
	rows := map[string]BranchReviewRowInfo{}
	for _, row := range review.Rows {
		rows[row.Name] = row
	}
	if main := rows["main"]; main.Deletable || !main.Current || main.Stale || main.Merged {
		t.Fatalf("main = %+v", main)
	}
	if stale := rows["stale-one"]; !stale.Stale || !stale.Merged || !stale.Deletable || stale.BehindBase != 1 {
		t.Fatalf("stale-one = %+v", stale)
	}
	if unpushed := rows["unpushed"]; !unpushed.Unpushed || unpushed.Stale || unpushed.Merged || unpushed.Author.Initials == "" {
		t.Fatalf("unpushed = %+v", unpushed)
	}

	tip, err := s.DeleteBranch(repo, "unpushed", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CreateBranchAt(repo, "unpushed", tip); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(mustOutput(t, repo, "rev-parse", "unpushed"))); got != tip {
		t.Fatalf("restored at %s, want %s", got, tip)
	}
}
