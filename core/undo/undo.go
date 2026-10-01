// Package undo reverses the most recent operation that moved HEAD, using the
// HEAD reflog. Undoing moves HEAD with a reset (or switches back), which adds
// its own reflog entry, so undoing an undo redoes it.
package undo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gspeager/current-client/core/git"
)

const (
	Commit     = "commit"
	Amend      = "amend"
	Merge      = "merge"
	Pull       = "pull"
	Rebase     = "rebase"
	Reset      = "reset"
	CherryPick = "cherry-pick"
	Revert     = "revert"
	Switch     = "switch"
)

// rebaseScan bounds how far back the start of a rebase is looked for.
const rebaseScan = 500

var (
	ErrNothingToUndo  = errors.New("Nothing to undo.")
	ErrLocalChanges   = errors.New("Commit or stash local changes first; undoing this would overwrite them.")
	ErrHistoryChanged = errors.New("History changed since the preview. Check the undo again.")
)

// Plan says what Apply will do. Removed and Restored count the commits that
// leave and come back to the branch; Pushed is true when a removed commit is
// already on the upstream.
type Plan struct {
	Operation string
	Detail    string
	Branch    string // "" when HEAD is detached
	From      string
	To        string
	SwitchTo  string
	Mode      git.ResetMode
	Removed   int
	Restored  int
	Pushed    bool
}

func Preview(ctx context.Context, repoPath string) (Plan, error) {
	conflict, err := git.DetectConflictState(ctx, repoPath)
	if err != nil {
		return Plan{}, err
	}
	if conflict.Operation != git.ConflictNone {
		return Plan{}, fmt.Errorf("Finish or abort the %s first.", conflict.Operation)
	}
	entries, err := git.Reflog(ctx, repoPath, rebaseScan)
	if err != nil {
		return Plan{}, err
	}
	if len(entries) < 2 {
		return Plan{}, ErrNothingToUndo
	}
	head, detail, _ := strings.Cut(entries[0].Action, ": ")
	plan := Plan{Operation: operation(head), Detail: detail, From: entries[0].SHA}
	switch plan.Operation {
	case "":
		return Plan{}, fmt.Errorf("The last operation (%s) can't be undone here.", head)
	case Switch:
		return switchPlan(ctx, repoPath, plan)
	case Commit, Amend:
		plan.Mode = git.ResetSoft
	default:
		plan.Mode = git.ResetKeep
	}
	plan.To = entries[1].SHA
	if plan.Operation == Rebase {
		start := rebaseStart(entries)
		if start < 0 || start+1 >= len(entries) {
			return Plan{}, errors.New("The start of the rebase is no longer in the reflog.")
		}
		plan.To = entries[start+1].SHA
	}
	if plan.Mode == git.ResetKeep {
		if err := requireClean(ctx, repoPath); err != nil {
			return Plan{}, err
		}
	}
	return withCounts(ctx, repoPath, plan)
}

func Apply(ctx context.Context, repoPath string, plan Plan) error {
	head, err := git.ResolveCommit(ctx, repoPath, "HEAD")
	if err != nil {
		return err
	}
	if head != plan.From {
		return ErrHistoryChanged
	}
	if plan.Operation == Switch {
		return git.CheckoutBranch(ctx, repoPath, plan.SwitchTo)
	}
	return git.Reset(ctx, repoPath, plan.To, plan.Mode)
}

func operation(head string) string {
	switch {
	case head == "commit":
		return Commit
	case head == "commit (amend)":
		return Amend
	case head == "commit (merge)", strings.HasPrefix(head, "merge "):
		return Merge
	case (strings.HasPrefix(head, "rebase") || strings.HasPrefix(head, "pull")) && strings.HasSuffix(head, "(finish)"):
		return Rebase
	case head == "pull" || strings.HasPrefix(head, "pull "):
		return Pull
	case head == "reset":
		return Reset
	case head == "cherry-pick":
		return CherryPick
	case head == "revert":
		return Revert
	case head == "checkout":
		return Switch
	}
	return ""
}

func rebaseStart(entries []git.ReflogEntry) int {
	for i, e := range entries {
		head, _, _ := strings.Cut(e.Action, ": ")
		if strings.HasSuffix(head, "(start)") {
			return i
		}
	}
	return -1
}

// switchPlan switches back to the branch HEAD came from ("moving from a to b").
func switchPlan(ctx context.Context, repoPath string, plan Plan) (Plan, error) {
	from, _, ok := strings.Cut(strings.TrimPrefix(plan.Detail, "moving from "), " to ")
	if !ok {
		return Plan{}, ErrNothingToUndo
	}
	if _, err := git.ResolveCommit(ctx, repoPath, "refs/heads/"+from); err != nil {
		return Plan{}, fmt.Errorf("%s isn't a local branch, so switching back can't be undone here.", from)
	}
	plan.SwitchTo = from
	status, err := git.CurrentBranchStatus(ctx, repoPath)
	if err != nil {
		return Plan{}, err
	}
	plan.Branch = branchName(status.Current)
	return plan, nil
}

func requireClean(ctx context.Context, repoPath string) error {
	statuses, err := git.GetStatus(ctx, repoPath)
	if err != nil {
		return err
	}
	for _, s := range statuses {
		if s.IndexStatus != '?' {
			return ErrLocalChanges
		}
	}
	return nil
}

func withCounts(ctx context.Context, repoPath string, plan Plan) (Plan, error) {
	status, err := git.CurrentBranchStatus(ctx, repoPath)
	if err != nil {
		return Plan{}, err
	}
	plan.Branch = branchName(status.Current)
	if plan.Removed, err = git.CountCommits(ctx, repoPath, plan.To+".."+plan.From); err != nil {
		return Plan{}, err
	}
	if plan.Restored, err = git.CountCommits(ctx, repoPath, plan.From+".."+plan.To); err != nil {
		return Plan{}, err
	}
	if status.Upstream != "" && plan.Removed > 0 {
		unpushed, err := git.CountCommits(ctx, repoPath, plan.From, "^"+plan.To, "^"+status.Upstream)
		if err != nil {
			return Plan{}, err
		}
		plan.Pushed = unpushed < plan.Removed
	}
	return plan, nil
}

func branchName(current string) string {
	if current == "HEAD" {
		return ""
	}
	return current
}
