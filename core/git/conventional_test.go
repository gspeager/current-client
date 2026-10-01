package git

import (
	"context"
	"fmt"
	"testing"

	"github.com/gspeager/current-client/core/internal/gittest"
)

func TestParseConventional(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		body    string
		want    ConventionalCommit
	}{
		{"type only", "feat: add changelog", "", ConventionalCommit{Type: "feat", Description: "add changelog"}},
		{"scope", "fix(graph): keep lane color", "", ConventionalCommit{Type: "fix", Scope: "graph", Description: "keep lane color"}},
		{"scope with slashes", "feat(core/git): parse types", "", ConventionalCommit{Type: "feat", Scope: "core/git", Description: "parse types"}},
		{"bang", "feat!: drop v0", "", ConventionalCommit{Type: "feat", Breaking: true, Description: "drop v0"}},
		{"bang with scope", "feat(config)!: drop v0", "", ConventionalCommit{Type: "feat", Scope: "config", Breaking: true, Description: "drop v0"}},
		{"uppercase type", "FEAT: shout", "", ConventionalCommit{Type: "feat", Description: "shout"}},
		{"footer", "refactor: rename", "Body text.\n\nBREAKING CHANGE: settings keys renamed", ConventionalCommit{Type: "refactor", Breaking: true, BreakingNote: "settings keys renamed", Description: "rename"}},
		{"hyphen footer", "fix: x", "BREAKING-CHANGE: y", ConventionalCommit{Type: "fix", Breaking: true, BreakingNote: "y", Description: "x"}},
		{"footer mid-line ignored", "fix: x", "see BREAKING CHANGE: y", ConventionalCommit{Type: "fix", Description: "x"}},
		{"plain subject", "Add branch switcher UI", "", ConventionalCommit{}},
		{"merge commit", "Merge branch 'feature' into main", "", ConventionalCommit{}},
		{"git revert", `Revert "feat: add changelog"`, "This reverts commit abc.", ConventionalCommit{}},
		{"revert type", "revert: feat: add changelog", "", ConventionalCommit{Type: "revert", Description: "feat: add changelog"}},
		{"no space after colon", "feat:add", "", ConventionalCommit{}},
		{"empty description", "feat: ", "", ConventionalCommit{}},
		{"empty scope", "feat(): x", "", ConventionalCommit{}},
		{"footer without type", "Plain subject", "BREAKING CHANGE: y", ConventionalCommit{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseConventional(tt.subject, tt.body); got != tt.want {
				t.Errorf("ParseConventional(%q, %q) = %+v, want %+v", tt.subject, tt.body, got, tt.want)
			}
		})
	}
}

func TestFollowsConventionalCommits(t *testing.T) {
	tests := []struct {
		name         string
		conventional int
		plain        int
		want         bool
	}{
		{"majority", 3, 2, true},
		{"exactly half", 2, 2, false},
		{"minority", 1, 3, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := gittest.InitRepo(t)
			for i := 0; i < tt.plain; i++ {
				gittest.CommitFile(t, dir, "a.txt", fmt.Sprint("plain", i), fmt.Sprint("Plain change ", i))
			}
			for i := 0; i < tt.conventional; i++ {
				gittest.CommitFile(t, dir, "a.txt", fmt.Sprint("cc", i), fmt.Sprint("feat: change ", i))
			}
			got, err := FollowsConventionalCommits(context.Background(), dir)
			if err != nil {
				t.Fatalf("FollowsConventionalCommits: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFollowsConventionalCommitsIgnoresMerges(t *testing.T) {
	dir := gittest.InitRepo(t)
	gittest.CommitFile(t, dir, "a.txt", "1", "feat: base")
	gittest.Run(t, dir, "checkout", "-b", "side")
	gittest.CommitFile(t, dir, "b.txt", "1", "fix: side")
	gittest.Run(t, dir, "checkout", "main")
	gittest.CommitFile(t, dir, "c.txt", "1", "Plain change")
	gittest.Run(t, dir, "merge", "--no-ff", "-m", "Merge branch 'side'", "side")

	got, err := FollowsConventionalCommits(context.Background(), dir)
	if err != nil {
		t.Fatalf("FollowsConventionalCommits: %v", err)
	}
	if !got {
		t.Error("the merge commit should not count against the format")
	}
}

func TestFollowsConventionalCommitsNoCommitsYet(t *testing.T) {
	got, err := FollowsConventionalCommits(context.Background(), gittest.InitRepo(t))
	if err != nil || got {
		t.Errorf("got %v, %v; want false, nil", got, err)
	}
}
