package tasks

import (
	"slices"
	"sync"
	"time"
)

// Outcome is a run that ended by itself, and the code it ran on.
type Outcome struct {
	Command  string
	ExitCode int
	Duration time.Duration
	At       time.Time
	// The worktree's tree, "" when it couldn't be worked out.
	Tree string
	// The files that differ from when it started, so it ran on no one state.
	Changed []string
}

type resultKey struct{ worktree, task string }

// Results holds the last outcome of each task in each worktree, until the app
// closes. A run that was stopped has none, and leaves the last one.
type Results struct {
	mu      sync.Mutex
	last    map[resultKey]Outcome
	running map[resultKey]int
}

func NewResults() *Results {
	return &Results{last: map[resultKey]Outcome{}, running: map[resultKey]int{}}
}

func (r *Results) Started(worktree, task string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.running[resultKey{worktree, task}]++
}

// Ended takes nil for a run that was stopped, or didn't start.
func (r *Results) Ended(worktree, task string, outcome *Outcome) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := resultKey{worktree, task}
	if r.running[key]--; r.running[key] <= 0 {
		delete(r.running, key)
	}
	if outcome != nil {
		r.last[key] = *outcome
	}
}

// Of is the task's last outcome, if it has one, and whether it's running.
func (r *Results) Of(worktree, task string) (outcome Outcome, has, running bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := resultKey{worktree, task}
	outcome, has = r.last[key]
	return outcome, has, r.running[key] > 0
}

// Any says whether a task has an outcome in the worktree.
func (r *Results) Any(worktree string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key := range r.last {
		if key.worktree == worktree {
			return true
		}
	}
	return false
}

// What a result is worth now.
const (
	StateCurrent  = "current"
	StateOutdated = "outdated"
	// Something changed while it ran: an edit, or the task's own output.
	StateChanged = "changed"
	StateUnknown = "unknown"
)

// State compares the code the outcome is of with the worktree's tree now, and the
// command with what it is now.
func (o Outcome) State(tree, command string) string {
	switch {
	case o.Command != command:
		return StateOutdated
	case len(o.Changed) > 0:
		return StateChanged
	case o.Tree == "" || tree == "":
		return StateUnknown
	case o.Tree == tree:
		return StateCurrent
	}
	return StateOutdated
}

// MergeCustom is the repository's own commands and those kept under one worktree's
// path, as all of them were before.
func MergeCustom(repository, worktree []Custom) []Custom {
	merged := slices.Clone(repository)
	for _, item := range worktree {
		if !slices.ContainsFunc(merged, func(c Custom) bool { return c.Name == item.Name }) {
			merged = append(merged, item)
		}
	}
	return merged
}
