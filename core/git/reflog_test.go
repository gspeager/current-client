package git

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestParseReflogFixture(t *testing.T) {
	output := "sha1\x1fHEAD@{0}\x1freset: moving to sha1\x1f1000\x1fv2\x00" +
		"sha2\x1fHEAD@{1}\x1fcommit: v3\x1f2000\x1fv3\x00"
	got, err := parseReflog(output)
	if err != nil {
		t.Fatalf("parseReflog: %v", err)
	}
	want := []ReflogEntry{
		{SHA: "sha1", Selector: "HEAD@{0}", Action: "reset: moving to sha1", Date: time.Unix(1000, 0), Subject: "v2"},
		{SHA: "sha2", Selector: "HEAD@{1}", Action: "commit: v3", Date: time.Unix(2000, 0), Subject: "v3"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestParseReflogEmpty(t *testing.T) {
	got, err := parseReflog("")
	if err != nil {
		t.Fatalf("parseReflog: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %+v, want empty", got)
	}
}

func TestReflogRecoversCommitLostToHardReset(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "v1")
	v1SHA := gittest.Run(t, dir, "rev-parse", "HEAD")
	gittest.CommitFile(t, dir, "file.txt", "v2", "v2")
	v2SHA := gittest.Run(t, dir, "rev-parse", "HEAD")

	gittest.Run(t, dir, "reset", "--hard", v1SHA)

	// v2 is no longer reachable from any branch, but the reflog remembers it.
	entries, err := Reflog(context.Background(), dir, 0)
	if err != nil {
		t.Fatalf("Reflog: %v", err)
	}
	var found *ReflogEntry
	for i := range entries {
		if entries[i].SHA == v2SHA {
			found = &entries[i]
		}
	}
	if found == nil {
		t.Fatalf("got %+v, want an entry for the lost v2 commit %q", entries, v2SHA)
	}
	if found.Subject != "v2" {
		t.Fatalf("Subject = %q, want %q", found.Subject, "v2")
	}

	// Prove it's genuinely recoverable: branching from the reflog SHA works.
	if err := CreateBranchAt(context.Background(), dir, "recovered", found.SHA); err != nil {
		t.Fatalf("CreateBranchAt(recovered SHA): %v", err)
	}
	recoveredHead := gittest.Run(t, dir, "rev-parse", "recovered")
	if recoveredHead != v2SHA {
		t.Fatalf("recovered branch HEAD = %q, want %q", recoveredHead, v2SHA)
	}
}
