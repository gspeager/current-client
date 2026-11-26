package hostapi

import (
	"strconv"
	"strings"
)

// Fact is one thing that bears on whether a change can merge.
type Fact struct {
	Label string `json:"label"`
	Value string `json:"value"`
	// Level is "ok", "warn", "info" or "unknown".
	Level string `json:"level"`
}

// ChangeFacts is what a change's own answer from the host says about merging
// it: draft, checks, reviews, threads and conflicts.
func ChangeFacts(change Change) []Fact {
	var facts []Fact
	add := func(label, value, level string) { facts = append(facts, Fact{label, value, level}) }
	if change.Draft {
		add("Draft", "marked as a draft", "warn")
	}
	switch change.CI {
	case CISuccess:
		add("Checks", "passed", "ok")
	case CIFailed:
		add("Checks", "failed", "warn")
	case CIRunning, CIPending:
		add("Checks", "still running", "warn")
	case "":
		add("Checks", "no result in the cache", "unknown")
	default:
		add("Checks", string(change.CI), "info")
	}
	facts = append(facts, reviewsFact(change.Reviews))
	if change.UnresolvedThreads > 0 {
		add("Threads", strconv.Itoa(change.UnresolvedThreads)+" unresolved", "warn")
	}
	switch {
	case !change.ConflictsKnown:
		add("Conflicts", "not known until the change is opened on the host", "unknown")
	case change.HasConflicts:
		add("Conflicts", "conflicts with "+change.TargetBranch, "warn")
	default:
		add("Conflicts", "none", "ok")
	}
	return facts
}

// ProtectionFact is the rule covering target, if any.
func ProtectionFact(rules []BranchRule, target string) (Fact, bool) {
	rule, ok := MatchRule(rules, target)
	if !ok {
		return Fact{}, false
	}
	value := "protected; the host applies its rules on merge"
	if rule.Pattern != target {
		value += " (rule " + rule.Pattern + ")"
	}
	return Fact{"Protection", value, "info"}, true
}

// reviewsFact names every reviewer, verdicts first as the list orders them. It
// warns while changes are requested or a review still waits.
func reviewsFact(reviews []Review) Fact {
	names := map[ReviewState][]string{}
	for _, r := range reviews {
		name := r.Reviewer.Name
		if name == "" {
			name = r.Reviewer.Username
		}
		names[r.State] = append(names[r.State], name)
	}
	var parts []string
	for _, group := range []struct {
		state  ReviewState
		prefix string
	}{
		{ReviewChangesRequested, "changes requested by "},
		{ReviewApproved, "approved by "},
		{ReviewRequested, "waiting on "},
		{ReviewCommented, "commented by "},
	} {
		if len(names[group.state]) > 0 {
			parts = append(parts, group.prefix+strings.Join(names[group.state], ", "))
		}
	}
	level := "info"
	switch {
	case len(parts) == 0:
		return Fact{"Reviews", "none", "info"}
	case len(names[ReviewChangesRequested]) > 0 || len(names[ReviewRequested]) > 0:
		level = "warn"
	case len(names[ReviewApproved]) > 0:
		level = "ok"
	}
	return Fact{"Reviews", strings.Join(parts, "; "), level}
}
