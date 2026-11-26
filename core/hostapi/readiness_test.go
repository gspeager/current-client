package hostapi

import "testing"

func TestChangeFacts(t *testing.T) {
	facts := ChangeFacts(Change{Draft: true, CI: CIFailed, UnresolvedThreads: 2, ConflictsKnown: true, HasConflicts: true, TargetBranch: "main",
		Reviews: []Review{{Reviewer: User{Name: "Ada"}, State: ReviewApproved}, {Reviewer: User{Username: "sam"}, State: ReviewRequested}}})
	want := []Fact{
		{"Draft", "marked as a draft", "warn"},
		{"Checks", "failed", "warn"},
		{"Reviews", "approved by Ada; waiting on sam", "warn"},
		{"Threads", "2 unresolved", "warn"},
		{"Conflicts", "conflicts with main", "warn"},
	}
	if len(facts) != len(want) {
		t.Fatalf("facts = %+v", facts)
	}
	for i := range want {
		if facts[i] != want[i] {
			t.Errorf("fact %d = %+v, want %+v", i, facts[i], want[i])
		}
	}
	if fact, ok := ProtectionFact([]BranchRule{{Pattern: "release/*"}}, "release/1.0"); !ok || fact.Value != "protected; the host applies its rules on merge (rule release/*)" {
		t.Errorf("protection = %+v, %v", fact, ok)
	}
}
