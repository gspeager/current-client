package git

import (
	"context"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestParseStatusFixtures(t *testing.T) {
	const blobA = "1111111111111111111111111111111111111111"
	const blobB = "2222222222222222222222222222222222222222"
	const zero = "0000000000000000000000000000000000000000"

	tests := []struct {
		name   string
		record string
		want   Status
	}{
		{
			name:   "modified, unstaged only",
			record: "1 .M N... 100644 100644 100644 " + blobA + " " + blobA + " file.txt",
			want:   Status{Path: "file.txt", IndexStatus: '.', WorktreeStatus: 'M'},
		},
		{
			name:   "modified, staged only",
			record: "1 M. N... 100644 100644 100644 " + blobA + " " + blobB + " file.txt",
			want:   Status{Path: "file.txt", IndexStatus: 'M', WorktreeStatus: '.'},
		},
		{
			name:   "added, staged",
			record: "1 A. N... 000000 100644 100644 " + zero + " " + blobB + " newfile.txt",
			want:   Status{Path: "newfile.txt", IndexStatus: 'A', WorktreeStatus: '.'},
		},
		{
			name:   "deleted, staged",
			record: "1 D. N... 100644 000000 000000 " + blobA + " " + zero + " deleted.txt",
			want:   Status{Path: "deleted.txt", IndexStatus: 'D', WorktreeStatus: '.'},
		},
		{
			name:   "untracked",
			record: "? untracked.txt",
			want:   Status{Path: "untracked.txt", IndexStatus: '?', WorktreeStatus: '?'},
		},
		{
			name:   "ignored",
			record: "! ignored.txt",
			want:   Status{Path: "ignored.txt", IndexStatus: '!', WorktreeStatus: '!'},
		},
		{
			name:   "unmerged, both modified",
			record: "u UU N... 100644 100644 100644 100644 " + blobA + " " + blobA + " " + blobB + " conflicted.txt",
			want:   Status{Path: "conflicted.txt", IndexStatus: 'U', WorktreeStatus: 'U', Conflicted: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseStatus(tt.record + "\x00")
			if err != nil {
				t.Fatalf("ParseStatus: %v", err)
			}
			if len(got) != 1 || got[0] != tt.want {
				t.Fatalf("got %+v, want [%+v]", got, tt.want)
			}
		})
	}
}

func TestParseStatusRename(t *testing.T) {
	const blob = "4444444444444444444444444444444444444444"
	output := "2 R. N... 100644 100644 100644 " + blob + " " + blob + " R100 newname.txt\x00oldname.txt\x00"

	got, err := ParseStatus(output)
	if err != nil {
		t.Fatalf("ParseStatus: %v", err)
	}
	want := Status{Path: "newname.txt", OrigPath: "oldname.txt", IndexStatus: 'R', WorktreeStatus: '.'}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %+v, want [%+v]", got, want)
	}
}

func TestGetStatusRealRepo(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init")
	gittest.Run(t, dir, "config", "user.email", "test@example.com")
	gittest.Run(t, dir, "config", "user.name", "Test")

	gittest.CommitFile(t, dir, "committed.txt", "v1", "initial")

	gittest.WriteFile(t, dir, "committed.txt", "v2") // unstaged modification
	gittest.WriteFile(t, dir, "staged.txt", "new")
	gittest.Run(t, dir, "add", "staged.txt") // staged addition
	gittest.WriteFile(t, dir, "untracked.txt", "new")

	statuses, err := GetStatus(context.Background(), dir)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	byPath := make(map[string]Status)
	for _, s := range statuses {
		byPath[s.Path] = s
	}

	if got := byPath["committed.txt"]; got.WorktreeStatus != 'M' {
		t.Fatalf("committed.txt = %+v, want unstaged M", got)
	}
	if got := byPath["staged.txt"]; got.IndexStatus != 'A' {
		t.Fatalf("staged.txt = %+v, want staged A", got)
	}
	if got := byPath["untracked.txt"]; got.WorktreeStatus != '?' {
		t.Fatalf("untracked.txt = %+v, want untracked", got)
	}
}
