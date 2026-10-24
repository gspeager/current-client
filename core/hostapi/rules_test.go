package hostapi

import "testing"

func TestMatchRulePrefersExactOverWildcard(t *testing.T) {
	rules := []BranchRule{
		{Pattern: "release/*", NoForcePush: true},
		{Pattern: "release/1.0", NoPush: true},
		{Pattern: "*-stable", NoDelete: true},
	}
	cases := map[string]string{
		"release/1.0":    "release/1.0",
		"release/2.0":    "release/*",
		"release/2.0/rc": "release/*",
		"v3-stable":      "*-stable",
		"main":           "",
	}
	for branch, want := range cases {
		rule, ok := MatchRule(rules, branch)
		if ok != (want != "") || rule.Pattern != want {
			t.Errorf("MatchRule(%q) = %q, %v; want %q", branch, rule.Pattern, ok, want)
		}
	}
}
