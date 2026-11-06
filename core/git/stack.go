package git

import (
	"context"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

// StackLink is one layer: Branch, built on Parent as it was at ParentTip.
type StackLink struct {
	Branch    string
	Parent    string
	ParentTip string
}

type StackMove struct {
	Branch string
	Before string
	// After is "" for the layer a restack stopped on, whose end isn't known.
	After string
}

// StackOrder puts each layer after its parent, and a layer's stack together.
func StackOrder(links []StackLink) []StackLink {
	layers := map[string]bool{}
	children := map[string][]StackLink{}
	for _, link := range links {
		layers[link.Branch] = true
		children[link.Parent] = append(children[link.Parent], link)
	}
	ordered := make([]StackLink, 0, len(links))
	var walk func(link StackLink)
	walk = func(link StackLink) {
		ordered = append(ordered, link)
		for _, child := range children[link.Branch] {
			walk(child)
		}
	}
	for _, link := range links {
		if !layers[link.Parent] {
			walk(link)
		}
	}
	return ordered
}

type restackStep struct {
	link    StackLink
	onto    string
	from    string
	commits int
	missing bool
	merged  bool
	moves   bool
}

// planRestack works out each layer in order. A layer that's gone or merged hands
// its children to its own parent, and they keep the tip they were built on, so
// only their own commits move, even after a squash merge.
func planRestack(ctx context.Context, repoPath string, links []StackLink) ([]restackStep, error) {
	replaced := map[string]string{}
	moved := map[string]bool{}
	var steps []restackStep
	for _, link := range StackOrder(links) {
		onto := link.Parent
		if next, ok := replaced[onto]; ok {
			onto = next
		}
		step := restackStep{link: link, onto: onto}
		// Qualified, as a tag of the same name would otherwise win.
		ontoRef := "refs/heads/" + onto
		tip, err := ResolveCommit(ctx, repoPath, "refs/heads/"+link.Branch)
		if err != nil {
			step.missing = true
			replaced[link.Branch] = onto
			steps = append(steps, step)
			continue
		}
		step.from = link.ParentTip
		if step.from == "" || !isAncestor(ctx, repoPath, step.from, tip) {
			base, err := MergeBase(ctx, repoPath, ontoRef, tip)
			if err != nil {
				return nil, err
			}
			step.from = base
		}
		if step.commits, err = CountCommits(ctx, repoPath, step.from+".."+tip); err != nil {
			return nil, err
		}
		if step.commits > 0 && mergedInto(ctx, repoPath, ontoRef, tip) {
			step.merged = true
			replaced[link.Branch] = onto
			steps = append(steps, step)
			continue
		}
		step.moves = moved[onto] || !isAncestor(ctx, repoPath, ontoRef, tip)
		moved[link.Branch] = step.moves
		steps = append(steps, step)
	}
	return steps, nil
}

func mergedInto(ctx context.Context, repoPath, base, tip string) bool {
	merged, err := MergedInto(ctx, repoPath, base, []string{tip})
	if err != nil {
		return isAncestor(ctx, repoPath, tip, base)
	}
	return merged[0]
}

type StackLayer struct {
	Branch  string
	Parent  string
	Commits int
	Missing bool
	Merged  bool
	Behind  bool
}

func StackLayers(ctx context.Context, repoPath string, links []StackLink) ([]StackLayer, error) {
	steps, err := planRestack(ctx, repoPath, links)
	if err != nil {
		return nil, err
	}
	layers := make([]StackLayer, 0, len(steps))
	for _, step := range steps {
		layers = append(layers, StackLayer{
			Branch:  step.link.Branch,
			Parent:  step.link.Parent,
			Commits: step.commits,
			Missing: step.missing,
			Merged:  step.merged,
			Behind:  step.moves,
		})
	}
	return layers, nil
}

// RefreshStackLinks moves ParentTip forward for a layer already built on its
// parent's tip, so what it's built on stays known however it got there, such as
// a rebase in a terminal or a finished conflict.
func RefreshStackLinks(ctx context.Context, repoPath string, links []StackLink) ([]StackLink, bool) {
	changed := false
	refreshed := make([]StackLink, len(links))
	for i, link := range links {
		refreshed[i] = link
		tip, err := ResolveCommit(ctx, repoPath, "refs/heads/"+link.Parent)
		if err != nil || tip == link.ParentTip || !isAncestor(ctx, repoPath, tip, "refs/heads/"+link.Branch) {
			continue
		}
		refreshed[i].ParentTip = tip
		changed = true
	}
	return refreshed, changed
}

type RestackPreviewLayer struct {
	Branch        string
	Onto          string
	Commits       []PreviewCommit
	DroppedMerges int
	Missing       bool
	Merged        bool
	Moves         bool
}

type RestackPreview struct {
	Layers    []RestackPreviewLayer
	Predicted bool
	Dirty     bool
	// Blocked is why it can't start, such as a layer in another worktree.
	Blocked string
}

// PreviewRestack predicts each layer on what the layer below is predicted to
// become. Nothing above a conflict can be predicted.
func PreviewRestack(ctx context.Context, repoPath string, links []StackLink) (RestackPreview, error) {
	preview := RestackPreview{Predicted: true}
	steps, err := planRestack(ctx, repoPath, links)
	if err != nil {
		return preview, err
	}
	predicted := map[string]string{}
	for _, step := range steps {
		layer := RestackPreviewLayer{Branch: step.link.Branch, Onto: step.onto, Missing: step.missing, Merged: step.merged, Moves: step.moves}
		if step.moves {
			revs := step.from + "..refs/heads/" + step.link.Branch
			result, err := runResult(ctx, repoPath, "log", "--reverse", "--no-merges", "--format=%H%x00%s", revs)
			if err != nil {
				return preview, err
			}
			var shas []string
			for _, line := range strings.Split(result.Stdout, "\n") {
				if sha, subject, ok := strings.Cut(line, "\x00"); ok {
					shas = append(shas, sha)
					layer.Commits = append(layer.Commits, PreviewCommit{SHA: sha, Subject: subject, Outcome: ReplayUnknown})
				}
			}
			if merges, err := CountCommits(ctx, repoPath, "--merges", revs); err == nil {
				layer.DroppedMerges = merges
			}
			onto, known := predicted[step.onto]
			if !known {
				onto = "refs/heads/" + step.onto
			}
			tip := onto
			if onto != "" && len(shas) > 0 {
				tip = ""
				replay, end, err := replayToTip(ctx, repoPath, onto, shas)
				if err != nil {
					preview.Predicted = false
				} else {
					tip = end
					for i, replayed := range replay {
						layer.Commits[i].Outcome = replayed.Outcome
						layer.Commits[i].Files = replayed.Files
					}
				}
			}
			predicted[step.link.Branch] = tip
		}
		preview.Layers = append(preview.Layers, layer)
	}
	preview.Blocked, preview.Dirty, err = restackBlocked(ctx, repoPath, steps)
	return preview, err
}

func restackBlocked(ctx context.Context, repoPath string, steps []restackStep) (string, bool, error) {
	state, err := DetectConflictState(ctx, repoPath)
	if err != nil {
		return "", false, err
	}
	if state.Operation != ConflictNone {
		return "Finish or abort the " + string(state.Operation) + " first.", false, nil
	}
	worktrees, err := ListWorktrees(ctx, repoPath)
	if err != nil {
		return "", false, err
	}
	for _, step := range steps {
		if !step.moves {
			continue
		}
		for _, worktree := range worktrees {
			if worktree.Branch == step.link.Branch && !worktree.Current {
				return step.link.Branch + " is checked out in the worktree at " + worktree.Path + ".", false, nil
			}
		}
	}
	dirty, err := rebaseBlockedByChanges(ctx, repoPath)
	return "", dirty, err
}

type RestackResult struct {
	Links []StackLink
	Moves []StackMove
	// Stopped is the layer with a conflict, which the conflict banner takes from
	// there. Restacking again carries on above it.
	Stopped string
	// Left is the layers that left the stack, being merged or gone.
	Left []string
}

// Restack rebases each layer whose parent has moved, using its own commits only,
// and goes back to the branch it started on. It moves one branch at a time, so it
// works from Git 2.23, where git rebase --update-refs needs 2.38.
func Restack(ctx context.Context, repoPath string, links []StackLink) (RestackResult, error) {
	var result RestackResult
	steps, err := planRestack(ctx, repoPath, links)
	if err != nil {
		return result, err
	}
	blocked, dirty, err := restackBlocked(ctx, repoPath, steps)
	if err != nil {
		return result, err
	}
	if blocked != "" {
		return result, &gitexec.AppError{Message: blocked}
	}

	var moving bool
	for _, step := range steps {
		if step.missing || step.merged {
			result.Left = append(result.Left, step.link.Branch)
		} else {
			result.Links = append(result.Links, StackLink{Branch: step.link.Branch, Parent: step.onto, ParentTip: step.from})
			moving = moving || step.moves
		}
	}
	if !moving {
		return result, nil
	}
	if dirty {
		return result, &gitexec.AppError{Message: "Commit or stash the changes in the working tree first."}
	}
	head, err := runResult(ctx, repoPath, "symbolic-ref", "-q", "--short", "HEAD")
	if err != nil {
		return result, &gitexec.AppError{Message: "Check out a branch to restack."}
	}
	start := strings.TrimSpace(head.Stdout)

	index := map[string]int{}
	for i, link := range result.Links {
		index[link.Branch] = i
	}
	for _, step := range steps {
		if step.missing || step.merged {
			continue
		}
		at := index[step.link.Branch]
		if step.moves {
			before, err := ResolveCommit(ctx, repoPath, "refs/heads/"+step.link.Branch)
			if err != nil {
				return result, err
			}
			args := []string{"rebase", "--onto", "refs/heads/" + step.onto, step.from, step.link.Branch}
			if _, err := runStoppable(ctx, gitexec.Command{Dir: repoPath, Args: args}); err != nil {
				_ = CheckoutBranch(ctx, repoPath, start)
				return result, err
			}
			state, err := DetectConflictState(ctx, repoPath)
			if err != nil {
				return result, err
			}
			if state.Operation != ConflictNone {
				result.Stopped = step.link.Branch
				result.Moves = append(result.Moves, StackMove{Branch: step.link.Branch, Before: before})
				return result, nil
			}
			after, err := ResolveCommit(ctx, repoPath, "refs/heads/"+step.link.Branch)
			if err != nil {
				return result, err
			}
			result.Moves = append(result.Moves, StackMove{Branch: step.link.Branch, Before: before, After: after})
		}
		tip, err := ResolveCommit(ctx, repoPath, "refs/heads/"+step.onto)
		if err != nil {
			return result, err
		}
		result.Links[at].ParentTip = tip
	}
	return result, CheckoutBranch(ctx, repoPath, start)
}

// StackMovedError is a restack that can no longer be undone, however long it waits.
type StackMovedError struct{ Message string }

func (e *StackMovedError) Error() string { return e.Message }

// UndoRestack puts each moved branch back, refusing if one has moved since, so it
// can't throw away later work.
func UndoRestack(ctx context.Context, repoPath string, moves []StackMove) error {
	state, err := DetectConflictState(ctx, repoPath)
	if err != nil {
		return err
	}
	if state.Operation != ConflictNone {
		return &gitexec.AppError{Message: "Finish or abort the " + string(state.Operation) + " first."}
	}
	current := ""
	if head, err := runResult(ctx, repoPath, "symbolic-ref", "-q", "--short", "HEAD"); err == nil {
		current = strings.TrimSpace(head.Stdout)
	}
	worktrees, err := ListWorktrees(ctx, repoPath)
	if err != nil {
		return err
	}
	tips := map[string]string{}
	for _, move := range moves {
		tip, err := ResolveCommit(ctx, repoPath, "refs/heads/"+move.Branch)
		if err != nil {
			return &StackMovedError{move.Branch + " is gone, so the restack can't be undone."}
		}
		if move.After != "" && tip != move.After {
			return &StackMovedError{move.Branch + " has changed since the restack, so it can't be undone."}
		}
		tips[move.Branch] = tip
		for _, worktree := range worktrees {
			if worktree.Branch == move.Branch && !worktree.Current {
				return &gitexec.AppError{Message: move.Branch + " is checked out in the worktree at " + worktree.Path + "."}
			}
		}
	}
	for _, move := range moves {
		if move.Branch == current {
			if err := Reset(ctx, repoPath, move.Before, ResetKeep); err != nil {
				return err
			}
			continue
		}
		if _, err := runResult(ctx, repoPath, "update-ref", "refs/heads/"+move.Branch, move.Before, tips[move.Branch]); err != nil {
			return err
		}
	}
	return nil
}

// PushStack pushes every layer with --force-with-lease, all or none where the
// server can do that, so a rejected layer can't leave the rest out of step.
func PushStack(ctx context.Context, repoPath, remote string, branches []string) error {
	args := []string{"push", "--force-with-lease", remote}
	for _, branch := range branches {
		args = append(args, "refs/heads/"+branch+":refs/heads/"+branch)
	}
	atomic := append([]string{args[0], "--atomic"}, args[1:]...)
	_, err := runResult(ctx, repoPath, atomic...)
	if err != nil && strings.Contains(err.Error(), "--atomic") {
		_, err = runResult(ctx, repoPath, args...)
	}
	if err != nil {
		return err
	}
	// Only where there's none: a layer tracking somewhere else keeps tracking it.
	for _, branch := range branches {
		if _, err := runResult(ctx, repoPath, "rev-parse", "--abbrev-ref", branch+"@{upstream}"); err != nil {
			if err := SetUpstream(ctx, repoPath, branch, remote+"/"+branch); err != nil {
				return err
			}
		}
	}
	return nil
}
