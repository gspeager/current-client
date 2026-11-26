package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gspeager/current-client/internal/gittest"
)

func inboxKinds(rows []InboxRow) map[string]InboxRow {
	byKind := map[string]InboxRow{}
	for _, r := range rows {
		byKind[r.Kind+"/"+r.Branch] = r
	}
	return byKind
}

func TestInboxLocalRowsWithEveryHostOff(t *testing.T) {
	useTempSettings(t)
	remote := t.TempDir()
	gittest.Run(t, remote, "init", "-q", "--bare", "-b", "main")
	repo := t.TempDir()
	gittest.Run(t, repo, "init", "-q", "-b", "main")
	gittest.Run(t, repo, "config", "user.name", "T")
	gittest.Run(t, repo, "config", "user.email", "t@example.com")
	commit := func(msg, date string) {
		t.Helper()
		t.Setenv("GIT_COMMITTER_DATE", date)
		t.Setenv("GIT_AUTHOR_DATE", date)
		gittest.Run(t, repo, "commit", "-q", "--allow-empty", "-m", msg)
	}
	commit("initial", "2026-10-01T10:00:00Z")
	gittest.Run(t, repo, "remote", "add", "origin", remote)
	gittest.Run(t, repo, "push", "-q", "-u", "origin", "main")
	gittest.Run(t, repo, "switch", "-q", "-c", "feature")
	commit("feature work", "2026-10-02T10:00:00Z")
	gittest.Run(t, repo, "branch", "-q", "--set-upstream-to=origin/main", "feature")
	gittest.Run(t, repo, "switch", "-q", "-c", "old", "main")
	commit("old work", "2025-01-01T10:00:00Z")
	gittest.Run(t, repo, "switch", "-q", "-c", "done", "main")
	gittest.Run(t, repo, "branch", "-q", "--set-upstream-to=origin/main", "done")
	gittest.Run(t, repo, "switch", "-q", "main")
	if err := os.WriteFile(filepath.Join(repo, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	repos := (&InboxService{}).Rows([]string{repo, filepath.Join(t.TempDir(), "missing")}, true)
	if len(repos) != 2 || repos[0].Error != "" || repos[1].Error == "" {
		t.Fatalf("Rows = %+v", repos)
	}
	rows := inboxKinds(repos[0].Rows)
	if r := rows["dirty/"]; r.Count != 1 {
		t.Errorf("dirty row = %+v", r)
	}
	if r := rows["unpushed/feature"]; r.Count != 1 {
		t.Errorf("unpushed row = %+v", r)
	}
	if r, ok := rows["stale/old"]; !ok || r.Count < 600 {
		t.Errorf("stale row = %+v", r)
	}
	if r := rows["merged/done"]; r.Detail != "origin/main" {
		t.Errorf("merged row = %+v", r)
	}
	if _, ok := rows["stale/done"]; ok {
		t.Error("a merged branch is not also stale")
	}
	if repos[0].HostEnabled || repos[0].FetchedAt != nil {
		t.Fatalf("no host: %+v", repos[0])
	}
	for _, r := range repos[0].Rows {
		if r.FetchedAt != nil {
			t.Errorf("no host row with the host off: %+v", r)
		}
	}
}
