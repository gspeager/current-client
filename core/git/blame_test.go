package git

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gspeager/current-client/core/internal/gittest"
)

const blameFixture = `41198c12d1cb5a666cce78d34b6bb73cccc316f9 1 1 1
author Alice
author-mail <a@b.com>
author-time 1790080705
author-tz -0600
committer Alice
committer-mail <a@b.com>
committer-time 1790080705
committer-tz -0600
summary add f
boundary
filename f.txt
	line1
e47c3b333ebfdb8dd309634f4b7a9c500bead071 2 2 1
author Bob
author-mail <bob@b.com>
author-time 1790080800
author-tz -0600
committer Bob
committer-mail <bob@b.com>
committer-time 1790080800
committer-tz -0600
summary edit f
previous 41198c12d1cb5a666cce78d34b6bb73cccc316f9 f.txt
filename f.txt
	line2-edited
41198c12d1cb5a666cce78d34b6bb73cccc316f9 3 3 1
	line3
e47c3b333ebfdb8dd309634f4b7a9c500bead071 4 4 1
	line4
`

func TestParseBlameFixture(t *testing.T) {
	got, err := parseBlame(blameFixture)
	if err != nil {
		t.Fatalf("parseBlame: %v", err)
	}
	want := []BlameLine{
		{SHA: "41198c12d1cb5a666cce78d34b6bb73cccc316f9", AuthorName: "Alice", AuthorEmail: "a@b.com", Date: time.Unix(1790080705, 0), Summary: "add f", LineNo: 1, Content: "line1"},
		{SHA: "e47c3b333ebfdb8dd309634f4b7a9c500bead071", AuthorName: "Bob", AuthorEmail: "bob@b.com", Date: time.Unix(1790080800, 0), Summary: "edit f", LineNo: 2, Content: "line2-edited"},
		// Repeated commits omit their metadata, as git does.
		{SHA: "41198c12d1cb5a666cce78d34b6bb73cccc316f9", AuthorName: "Alice", AuthorEmail: "a@b.com", Date: time.Unix(1790080705, 0), Summary: "add f", LineNo: 3, Content: "line3"},
		{SHA: "e47c3b333ebfdb8dd309634f4b7a9c500bead071", AuthorName: "Bob", AuthorEmail: "bob@b.com", Date: time.Unix(1790080800, 0), Summary: "edit f", LineNo: 4, Content: "line4"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestParseBlameEmpty(t *testing.T) {
	got, err := parseBlame("")
	if err != nil {
		t.Fatalf("parseBlame: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %+v, want empty", got)
	}
}

func TestBlameRealRepoMultipleContributors(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init", "-b", "main")
	gittest.Run(t, dir, "config", "core.autocrlf", "false")
	gittest.Run(t, dir, "config", "user.email", "alice@example.com")
	gittest.Run(t, dir, "config", "user.name", "Alice")
	gittest.CommitFile(t, dir, "f.txt", "line1\nline2\nline3\n", "add f")

	gittest.Run(t, dir, "config", "user.email", "bob@example.com")
	gittest.Run(t, dir, "config", "user.name", "Bob")
	gittest.CommitFile(t, dir, "f.txt", "line1\nline2-edited\nline3\nline4\n", "edit f")

	got, err := Blame(context.Background(), dir, "f.txt")
	if err != nil {
		t.Fatalf("Blame: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("got %d lines, want 4: %+v", len(got), got)
	}
	wantAuthors := []string{"Alice", "Bob", "Alice", "Bob"}
	wantContent := []string{"line1", "line2-edited", "line3", "line4"}
	for i, line := range got {
		if line.AuthorName != wantAuthors[i] {
			t.Fatalf("line %d: AuthorName = %q, want %q", i+1, line.AuthorName, wantAuthors[i])
		}
		if line.Content != wantContent[i] {
			t.Fatalf("line %d: Content = %q, want %q", i+1, line.Content, wantContent[i])
		}
		if line.LineNo != i+1 {
			t.Fatalf("line %d: LineNo = %d, want %d", i+1, line.LineNo, i+1)
		}
	}
	if got[0].Summary != "add f" || got[1].Summary != "edit f" {
		t.Fatalf("got summaries %q, %q; want %q, %q", got[0].Summary, got[1].Summary, "add f", "edit f")
	}
}
