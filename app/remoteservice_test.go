package app

import (
	"testing"

	"github.com/gspeager/current-client/core/git"
)

func TestSplitRemoteBranch(t *testing.T) {
	remotes := []git.Remote{{Name: "origin"}, {Name: "team"}, {Name: "team/origin"}}
	tests := []struct {
		ref, remote, branch string
		ok                  bool
	}{
		{"origin/feature", "origin", "feature", true},
		{"origin/user/feature", "origin", "user/feature", true},
		{"team/origin/feature", "team/origin", "feature", true},
		{"team/feature", "team", "feature", true},
		{"upstream/feature", "", "", false},
		{"origin/", "", "", false},
	}
	for _, tt := range tests {
		remote, branch, ok := splitRemoteBranch(remotes, tt.ref)
		if remote != tt.remote || branch != tt.branch || ok != tt.ok {
			t.Errorf("splitRemoteBranch(%q) = %q, %q, %v; want %q, %q, %v", tt.ref, remote, branch, ok, tt.remote, tt.branch, tt.ok)
		}
	}
}
