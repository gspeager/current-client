package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGetFileContentReadsEachVersion(t *testing.T) {
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "user.name=T", "-c", "user.email=t@example.com"}, args...)...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "logo.png"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "-q", "-b", "main")
	write("committed")
	git("add", "logo.png")
	git("commit", "-q", "-m", "add logo")
	write("staged")
	git("add", "logo.png")
	write("working")

	s := &DiffService{}
	for _, tt := range []struct {
		source FileSource
		want   string
	}{
		{FileSource{Kind: "commit", Rev: "HEAD"}, "committed"},
		{FileSource{Kind: "index"}, "staged"},
		{FileSource{Kind: "worktree"}, "working"},
	} {
		got, err := s.GetFileContent(dir, "logo.png", tt.source)
		if err != nil || !got.Found || string(got.Data) != tt.want {
			t.Errorf("%+v = %+v, %v; want %q", tt.source, got, err, tt.want)
		}
	}

	if got, err := s.GetFileContent(dir, "logo.png", FileSource{Kind: "commit", Rev: "HEAD~1"}); err != nil || got.Found {
		t.Errorf("before the first commit = %+v, %v; want not found", got, err)
	}
	if _, err := s.GetFileContent(dir, "logo.png", FileSource{Kind: "branch"}); err == nil {
		t.Error("an unknown source kind was accepted")
	}
}

func TestStagedAndCommitDiffsHoldBackALargeFileUntilForced(t *testing.T) {
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "user.name=T", "-c", "user.email=t@example.com"}, args...)...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q", "-b", "main")
	git("commit", "-q", "--allow-empty", "-m", "initial")
	big := make([]byte, 2<<20)
	for i := range big {
		big[i] = 'a' + byte(i%26)
		if i%80 == 79 {
			big[i] = '\n'
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "big.txt"), big, 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "big.txt")

	s := &DiffService{}
	staged, err := s.GetIndexDiff(dir, "big.txt", "", false, false)
	if err != nil || !staged.TooLarge || staged.SizeBytes != int64(len(big)) {
		t.Fatalf("staged = TooLarge %v, size %d, err %v; want it held back", staged.TooLarge, staged.SizeBytes, err)
	}
	if forced, err := s.GetIndexDiff(dir, "big.txt", "", true, false); err != nil || forced.TooLarge || len(forced.Hunks) == 0 {
		t.Fatalf("forced staged diff = %+v, %v; want it loaded", forced.TooLarge, err)
	}

	git("commit", "-q", "-m", "add big")
	if commit, err := s.GetRefDiff(dir, "big.txt", "HEAD~1", "HEAD", false, false); err != nil || !commit.TooLarge {
		t.Fatalf("commit diff TooLarge = %v, %v; want it held back", commit.TooLarge, err)
	}
}
