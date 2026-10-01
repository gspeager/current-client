package git

import (
	"context"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestParseHistoryFixture(t *testing.T) {
	output := "aaa1\x1fbbb1\x1fAlice\x1falice@example.com\x1f1700000000\x1fFirst commit\x1f\x1fG\x00" +
		"bbb1\x1f\x1fBob\x1fbob@example.com\x1f1699999000\x1fInitial commit\x1fLonger body\nwith two lines\n\x1fN\x00"

	got, err := parseHistory(output)
	if err != nil {
		t.Fatalf("parseHistory: %v", err)
	}
	want := []HistoryEntry{
		{
			SHA:         "aaa1",
			ParentSHAs:  []string{"bbb1"},
			AuthorName:  "Alice",
			AuthorEmail: "alice@example.com",
			Date:        time.Unix(1700000000, 0),
			Subject:     "First commit",
			Body:        "",
			Signature:   SignatureVerified,
		},
		{
			SHA:         "bbb1",
			ParentSHAs:  nil,
			AuthorName:  "Bob",
			AuthorEmail: "bob@example.com",
			Date:        time.Unix(1699999000, 0),
			Subject:     "Initial commit",
			Body:        "Longer body\nwith two lines",
			Signature:   SignatureNone,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestParseHistoryMergeCommitParents(t *testing.T) {
	output := "ccc1\x1faaa1 bbb1\x1fAlice\x1falice@example.com\x1f1700000000\x1fMerge branch\x1f\x1fN\x00"

	got, err := parseHistory(output)
	if err != nil {
		t.Fatalf("parseHistory: %v", err)
	}
	if len(got) != 1 || !reflect.DeepEqual(got[0].ParentSHAs, []string{"aaa1", "bbb1"}) {
		t.Fatalf("got %+v, want a single commit with two parents", got)
	}
}

func TestParseSignatureStatusCollapsesUntrustedCodesToUnverified(t *testing.T) {
	tests := []struct {
		code string
		want SignatureStatus
	}{
		{"G", SignatureVerified},
		{"N", SignatureNone},
		{"", SignatureNone},
		{"B", SignatureUnverified},
		{"U", SignatureUnverified},
		{"X", SignatureUnverified},
		{"Y", SignatureUnverified},
		{"R", SignatureUnverified},
		{"E", SignatureUnverified},
	}
	for _, tt := range tests {
		if got := parseSignatureStatus(tt.code); got != tt.want {
			t.Errorf("parseSignatureStatus(%q) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestHistoryRealRepoPagination(t *testing.T) {
	dir := gittest.InitRepo(t)

	var shas []string
	for i := 0; i < 5; i++ {
		gittest.CommitFile(t, dir, "file.txt", string(rune('a'+i)), "commit "+string(rune('a'+i)))
		shas = append(shas, gittest.Run(t, dir, "rev-parse", "HEAD"))
	}
	// shas is oldest-to-newest; git log lists newest-first.
	newestFirst := []string{shas[4], shas[3], shas[2], shas[1], shas[0]}

	page, err := History(context.Background(), dir, 2, 1, HistoryFilter{})
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(page) != 2 {
		t.Fatalf("got %d commits, want 2", len(page))
	}
	if page[0].SHA != newestFirst[1] || page[1].SHA != newestFirst[2] {
		t.Fatalf("got SHAs %q, %q; want %q, %q", page[0].SHA, page[1].SHA, newestFirst[1], newestFirst[2])
	}

	all, err := History(context.Background(), dir, 0, 0, HistoryFilter{})
	if err != nil {
		t.Fatalf("History (unlimited): %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("got %d commits, want 5", len(all))
	}
}

func TestHistoryRealMergeCommit(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "initial")
	gittest.Run(t, dir, "checkout", "-q", "-b", "feature")
	gittest.CommitFile(t, dir, "feature.txt", "v1", "feature work")
	gittest.Run(t, dir, "checkout", "-q", "main")
	gittest.Run(t, dir, "merge", "--no-ff", "-q", "-m", "merge feature", "feature")
	mergeSHA := gittest.Run(t, dir, "rev-parse", "HEAD")

	commits, err := History(context.Background(), dir, 1, 0, HistoryFilter{})
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(commits) != 1 || commits[0].SHA != mergeSHA {
		t.Fatalf("got %+v, want the merge commit first", commits)
	}
	if len(commits[0].ParentSHAs) != 2 {
		t.Fatalf("ParentSHAs = %v, want 2 parents", commits[0].ParentSHAs)
	}
}

func TestHistoryFilterByRef(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "on main")
	gittest.Run(t, dir, "checkout", "-q", "-b", "feature")
	gittest.CommitFile(t, dir, "feature.txt", "v1", "on feature")
	gittest.Run(t, dir, "checkout", "-q", "main")

	// Empty Ref means HEAD (main here) — the feature-only commit shouldn't appear.
	onMain, err := History(context.Background(), dir, 0, 0, HistoryFilter{})
	if err != nil {
		t.Fatalf("History (HEAD): %v", err)
	}
	for _, c := range onMain {
		if c.Subject == "on feature" {
			t.Fatalf("HEAD-only History includes the feature branch's commit: %+v", onMain)
		}
	}

	onFeature, err := History(context.Background(), dir, 0, 0, HistoryFilter{Ref: "feature"})
	if err != nil {
		t.Fatalf("History (feature): %v", err)
	}
	if len(onFeature) != 2 {
		t.Fatalf("got %d commits for feature, want 2 (both commits)", len(onFeature))
	}

	all, err := History(context.Background(), dir, 0, 0, HistoryFilter{Ref: "--all"})
	if err != nil {
		t.Fatalf("History (--all): %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("got %d commits for --all, want 2 (both branches' commits, deduped)", len(all))
	}
}

func TestHistoryFilterByAuthor(t *testing.T) {
	dir := t.TempDir()
	gittest.Run(t, dir, "init", "-b", "main")
	gittest.WriteFile(t, dir, "file.txt", "v1")
	gittest.Run(t, dir, "add", "file.txt")
	gittest.Run(t, dir, "-c", "user.name=Alice", "-c", "user.email=alice@example.com", "commit", "-m", "alice's commit")
	gittest.WriteFile(t, dir, "file.txt", "v2")
	gittest.Run(t, dir, "add", "file.txt")
	gittest.Run(t, dir, "-c", "user.name=Bob", "-c", "user.email=bob@example.com", "commit", "-m", "bob's commit")

	got, err := History(context.Background(), dir, 0, 0, HistoryFilter{Author: "Alice"})
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(got) != 1 || got[0].AuthorName != "Alice" {
		t.Fatalf("got %+v, want only Alice's commit", got)
	}
}

func TestHistoryFilterBySinceUntil(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.WriteFile(t, dir, "file.txt", "v1")
	gittest.Run(t, dir, "add", "file.txt")
	oldCommit := exec.Command("git", "commit", "-m", "old commit")
	oldCommit.Dir = dir
	oldCommit.Env = append(os.Environ(), "GIT_AUTHOR_DATE=2020-01-01T00:00:00", "GIT_COMMITTER_DATE=2020-01-01T00:00:00")
	if out, err := oldCommit.CombinedOutput(); err != nil {
		t.Fatalf("git commit (backdated): %v\n%s", err, out)
	}
	gittest.CommitFile(t, dir, "file.txt", "v2", "recent commit")

	got, err := History(context.Background(), dir, 0, 0, HistoryFilter{Since: "2024-01-01"})
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(got) != 1 || got[0].Subject != "recent commit" {
		t.Fatalf("got %+v, want only the recent commit", got)
	}
}

func TestSearchHistoryMatchesSubjectCaseInsensitive(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "file.txt", "v1", "Fix the LOGIN bug")
	gittest.CommitFile(t, dir, "file.txt", "v2", "unrelated change")

	got, err := SearchHistory(context.Background(), dir, "login", 0)
	if err != nil {
		t.Fatalf("SearchHistory: %v", err)
	}
	if len(got) != 1 || got[0].Subject != "Fix the LOGIN bug" {
		t.Fatalf("got %+v, want only the login-bug commit", got)
	}
}

func TestSearchHistoryRespectsLimit(t *testing.T) {
	dir := gittest.InitRepo(t)
	for i := 0; i < 3; i++ {
		gittest.CommitFile(t, dir, "file.txt", string(rune('a'+i)), "shared token commit "+string(rune('a'+i)))
	}

	got, err := SearchHistory(context.Background(), dir, "shared token", 2)
	if err != nil {
		t.Fatalf("SearchHistory: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d matches, want 2 (limit)", len(got))
	}
}

func TestCommitPositionCountsNewerCommits(t *testing.T) {
	dir := gittest.InitRepo(t)

	var shas []string
	for i := 0; i < 4; i++ {
		gittest.CommitFile(t, dir, "file.txt", string(rune('a'+i)), "commit "+string(rune('a'+i)))
		shas = append(shas, gittest.Run(t, dir, "rev-parse", "HEAD"))
	}

	pos, err := CommitPosition(context.Background(), dir, shas[1])
	if err != nil {
		t.Fatalf("CommitPosition: %v", err)
	}
	if pos != 2 {
		t.Fatalf("CommitPosition = %d, want 2 (two commits newer than shas[1])", pos)
	}

	headPos, err := CommitPosition(context.Background(), dir, shas[3])
	if err != nil {
		t.Fatalf("CommitPosition (HEAD): %v", err)
	}
	if headPos != 0 {
		t.Fatalf("CommitPosition(HEAD) = %d, want 0", headPos)
	}
}

func TestFileHistoryFollowsAcrossRename(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "old.txt", "v1", "create old.txt")
	gittest.CommitFile(t, dir, "old.txt", "v2", "edit before rename")
	gittest.Run(t, dir, "mv", "old.txt", "new.txt")
	gittest.Run(t, dir, "commit", "-m", "rename to new.txt")
	gittest.CommitFile(t, dir, "new.txt", "v3", "edit after rename")

	got, err := FileHistory(context.Background(), dir, "new.txt", 0, 0)
	if err != nil {
		t.Fatalf("FileHistory: %v", err)
	}
	subjects := make([]string, len(got))
	for i, e := range got {
		subjects[i] = e.Subject
	}
	want := []string{"edit after rename", "rename to new.txt", "edit before rename", "create old.txt"}
	if !reflect.DeepEqual(subjects, want) {
		t.Fatalf("got %v, want %v", subjects, want)
	}
}

func TestFileHistoryUnrelatedFileNotIncluded(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.WriteFile(t, dir, "a.txt", "v1")
	gittest.WriteFile(t, dir, "b.txt", "v1")
	gittest.Run(t, dir, "add", ".")
	gittest.Run(t, dir, "commit", "-m", "add both")
	gittest.CommitFile(t, dir, "b.txt", "v2", "edit b only")

	got, err := FileHistory(context.Background(), dir, "a.txt", 0, 0)
	if err != nil {
		t.Fatalf("FileHistory: %v", err)
	}
	if len(got) != 1 || got[0].Subject != "add both" {
		t.Fatalf("got %+v, want only the commit that touched a.txt", got)
	}
}

func TestHistoryTypeFilter(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "a.txt", "1", "feat: first")
	gittest.CommitFile(t, dir, "a.txt", "2", "fix(graph): lanes")
	gittest.CommitFile(t, dir, "a.txt", "3", "Plain change\n\nfix: only in the body")
	gittest.CommitFile(t, dir, "a.txt", "4", "FEAT(core/git): second")
	gittest.CommitFile(t, dir, "a.txt", "5", "refactor: rename\n\nBREAKING CHANGE: keys renamed")
	gittest.CommitFile(t, dir, "a.txt", "6", "feat!: third")

	subjects := func(filter HistoryFilter, limit, skip int) []string {
		t.Helper()
		entries, err := History(context.Background(), dir, limit, skip, filter)
		if err != nil {
			t.Fatalf("History: %v", err)
		}
		var got []string
		for _, e := range entries {
			got = append(got, e.Subject)
		}
		return got
	}

	tests := []struct {
		name        string
		filter      HistoryFilter
		limit, skip int
		want        []string
	}{
		{"type", HistoryFilter{Type: "feat"}, 0, 0, []string{"feat!: third", "FEAT(core/git): second", "feat: first"}},
		{"body match dropped", HistoryFilter{Type: "fix"}, 0, 0, []string{"fix(graph): lanes"}},
		{"breaking", HistoryFilter{Type: BreakingType}, 0, 0, []string{"feat!: third", "refactor: rename"}},
		{"paged", HistoryFilter{Type: "feat"}, 1, 1, []string{"FEAT(core/git): second"}},
		{"past the end", HistoryFilter{Type: "feat"}, 2, 5, nil},
		{"combined with author", HistoryFilter{Type: "feat", Author: "nobody"}, 0, 0, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := subjects(tt.filter, tt.limit, tt.skip)
			if strings.Join(got, "|") != strings.Join(tt.want, "|") {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
