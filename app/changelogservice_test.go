package app

import (
	"os/exec"
	"testing"
)

func TestChangelogBeforeTheFirstCommitIsEmptyNotAnError(t *testing.T) {
	dir := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	s := &ChangelogService{}

	if tag, err := s.LatestTag(dir, "HEAD"); tag != "" || err != nil {
		t.Errorf("LatestTag = %q, %v; want no tag and no error", tag, err)
	}
	sections, err := s.Build(dir, "", "HEAD", "", false)
	if err != nil || sections == nil || len(sections) != 0 {
		t.Errorf("Build = %#v, %v; want an empty list, not null or an error", sections, err)
	}
}
