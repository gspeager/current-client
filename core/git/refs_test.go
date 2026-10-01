package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseRefsFixture(t *testing.T) {
	output := strings.Join([]string{
		"aaa\t\trefs/heads/main",
		"bbb\t\trefs/heads/feature/x",
		"aaa\t\trefs/remotes/origin/main",
		"ccc\t\trefs/remotes/origin/HEAD", // symbolic; must be skipped
		"aaa\t\trefs/tags/v1.0.0",         // lightweight tag
		"ddd\taaa\trefs/tags/v2.0.0",      // annotated tag, dereferences to aaa
		"",
	}, "\n")

	refs := parseRefs(output)
	want := []Ref{
		{Kind: RefBranch, Name: "main", SHA: "aaa"},
		{Kind: RefBranch, Name: "feature/x", SHA: "bbb"},
		{Kind: RefRemoteBranch, Name: "origin/main", SHA: "aaa"},
		{Kind: RefTag, Name: "v1.0.0", SHA: "aaa"},
		{Kind: RefTag, Name: "v2.0.0", SHA: "aaa"},
	}
	if !reflect.DeepEqual(refs, want) {
		t.Fatalf("got %+v, want %+v", refs, want)
	}
}

func TestListRefsRealRepo(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	writeFile := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	run("init", "-q", "-b", "main")
	writeFile("a.txt", "one\n")
	run("add", ".")
	run("commit", "-q", "-m", "first")
	head := run("rev-parse", "HEAD")

	run("checkout", "-qb", "feature/x")
	run("checkout", "-q", "main")
	run("tag", "v1.0.0")
	run("tag", "-a", "v2.0.0", "-m", "annotated")

	refs, err := ListRefs(context.Background(), dir)
	if err != nil {
		t.Fatalf("ListRefs: %v", err)
	}

	byName := make(map[string]Ref, len(refs))
	for _, r := range refs {
		byName[r.Name] = r
	}

	for _, name := range []string{"main", "feature/x", "v1.0.0", "v2.0.0"} {
		r, ok := byName[name]
		if !ok {
			t.Fatalf("missing ref %q in %+v", name, refs)
		}
		if r.SHA != head {
			t.Fatalf("ref %q SHA = %s, want %s (both tags and both branches point at the same commit here)", name, r.SHA, head)
		}
	}
	if byName["main"].Kind != RefBranch {
		t.Fatalf("main kind = %v, want RefBranch", byName["main"].Kind)
	}
	if byName["v1.0.0"].Kind != RefTag || byName["v2.0.0"].Kind != RefTag {
		t.Fatalf("tag kinds wrong: %+v / %+v", byName["v1.0.0"], byName["v2.0.0"])
	}
	if _, ok := byName["origin/HEAD"]; ok {
		t.Fatalf("origin/HEAD symbolic ref should have been filtered out")
	}
}
