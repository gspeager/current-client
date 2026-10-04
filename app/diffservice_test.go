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
