package git

import (
	"context"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

type ReplayOutcome string

const (
	// ReplayClean applies without a conflict.
	ReplayClean ReplayOutcome = "clean"
	// ReplayEmpty applies but changes nothing, so a rebase drops the commit.
	ReplayEmpty ReplayOutcome = "empty"
	// ReplayConflict stops the replay on this commit.
	ReplayConflict ReplayOutcome = "conflict"
	// ReplayUnknown is a commit after a conflict: what happens depends on how
	// the conflict is resolved.
	ReplayUnknown ReplayOutcome = "unknown"
)

type ReplayStep struct {
	SHA     string
	Outcome ReplayOutcome
	Files   []string
}

// ReplayCommits predicts putting commits, in this order, on top of onto. It
// merges each with its own parent as the base, as a rebase does, which finds
// conflicts a merge of the branch tips can't: a commit that changes a line and a
// later one that changes it back merge cleanly, but stop a rebase. Nothing in the
// working tree, index or refs changes; only unreferenced objects are written.
//
// It returns ErrMergeTreeUnsupported on Git before 2.40, which added --merge-base.
func ReplayCommits(ctx context.Context, repoPath, onto string, commits []string) ([]ReplayStep, error) {
	current, err := ResolveCommit(ctx, repoPath, onto)
	if err != nil {
		return nil, err
	}
	currentTree, err := treeOf(ctx, repoPath, current)
	if err != nil {
		return nil, err
	}
	steps := make([]ReplayStep, 0, len(commits))
	for i, sha := range commits {
		merge, err := replayOne(ctx, repoPath, current, sha)
		if err != nil {
			return nil, err
		}
		if !merge.Clean {
			steps = append(steps, ReplayStep{SHA: sha, Outcome: ReplayConflict, Files: merge.Files})
			for _, rest := range commits[i+1:] {
				steps = append(steps, ReplayStep{SHA: rest, Outcome: ReplayUnknown})
			}
			return steps, nil
		}
		if merge.Tree == currentTree {
			steps = append(steps, ReplayStep{SHA: sha, Outcome: ReplayEmpty})
			continue
		}
		current, err = commitTree(ctx, repoPath, merge.Tree, current)
		if err != nil {
			return nil, err
		}
		currentTree = merge.Tree
		steps = append(steps, ReplayStep{SHA: sha, Outcome: ReplayClean})
	}
	return steps, nil
}

func replayOne(ctx context.Context, repoPath, onto, sha string) (MergeResult, error) {
	result, err := gitexec.NewExecutor("").Run(ctx, gitexec.Command{
		Dir: repoPath,
		Args: []string{
			"merge-tree", "--write-tree", "--name-only", "--no-messages", "-z",
			"--merge-base=" + sha + "^", onto, sha,
		},
	})
	if err != nil {
		return MergeResult{}, gitexec.WrapRunError(err)
	}
	fields := strings.Split(strings.TrimRight(result.Stdout, "\x00\n"), "\x00")
	switch result.ExitCode {
	case 0:
		return MergeResult{Tree: fields[0], Clean: true}, nil
	case 1:
		return MergeResult{Tree: fields[0], Files: fields[1:]}, nil
	}
	return MergeResult{}, mergeTreeError(result)
}

func treeOf(ctx context.Context, repoPath, rev string) (string, error) {
	result, err := runResult(ctx, repoPath, "rev-parse", "--verify", rev+"^{tree}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

// The identity is fixed so this works before the user has set one. The commit is
// a stepping stone for the next merge, and nothing refers to it.
func commitTree(ctx context.Context, repoPath, tree, parent string) (string, error) {
	result, err := gitexec.NewExecutor("").RunChecked(ctx, gitexec.Command{
		Dir:  repoPath,
		Args: []string{"commit-tree", tree, "-p", parent, "-m", "replay"},
		Env: []string{
			"GIT_AUTHOR_NAME=replay", "GIT_AUTHOR_EMAIL=replay@localhost",
			"GIT_COMMITTER_NAME=replay", "GIT_COMMITTER_EMAIL=replay@localhost",
		},
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

type PreviewCommit struct {
	SHA     string
	Subject string
	Outcome ReplayOutcome
	Files   []string
}

// RebasePreview says what RebaseOnto would do. Predicted is false when Git is too
// old to replay commits in memory, so a commit's Outcome says nothing.
type RebasePreview struct {
	Current       string
	Onto          string
	Commits       []PreviewCommit
	Predicted     bool
	DroppedMerges int
	ForcePush     bool
	Upstream      string
	// Rebase would refuse: tracked changes with no rebase.autoStash.
	Dirty   bool
	Command string
}

// PreviewRebase lists the commits as git rebase would pick them: not merges, and
// not ones already applied on onto by another name.
func PreviewRebase(ctx context.Context, repoPath, onto string) (RebasePreview, error) {
	preview := RebasePreview{Onto: onto, Command: "git rebase " + onto, Predicted: true}
	list, err := listBranches(ctx, repoPath)
	if err != nil {
		return preview, err
	}
	for _, b := range list.branches {
		if b.Current {
			preview.Current = b.Name
			if !list.goneUpstream[b.Name] {
				preview.Upstream = b.Upstream
			}
		}
	}
	result, err := runResult(ctx, repoPath, "log", "--reverse", "--no-merges", "--cherry-pick", "--right-only", "--format=%H%x00%s", onto+"...HEAD")
	if err != nil {
		return preview, err
	}
	var shas []string
	for _, line := range strings.Split(result.Stdout, "\n") {
		if sha, subject, ok := strings.Cut(line, "\x00"); ok {
			shas = append(shas, sha)
			preview.Commits = append(preview.Commits, PreviewCommit{SHA: sha, Subject: subject, Outcome: ReplayUnknown})
		}
	}
	if merges, err := CountCommits(ctx, repoPath, "--merges", onto+"..HEAD"); err == nil {
		preview.DroppedMerges = merges
	}
	// git rebase has nothing to do on a line already built on onto.
	if preview.DroppedMerges == 0 && isAncestor(ctx, repoPath, onto, "HEAD") {
		shas, preview.Commits = nil, nil
	}

	if len(shas) > 0 {
		// The rebase may still work, such as with a root commit in the range,
		// which has no parent to replay from.
		steps, err := ReplayCommits(ctx, repoPath, onto, shas)
		if err != nil {
			preview.Predicted = false
		} else {
			for i, step := range steps {
				preview.Commits[i].Outcome = step.Outcome
				preview.Commits[i].Files = step.Files
			}
		}
		if preview.Upstream != "" && isAncestor(ctx, repoPath, shas[0], preview.Upstream) && !isAncestor(ctx, repoPath, onto, shas[0]+"^") {
			preview.ForcePush = true
		}
	}

	statuses, err := GetStatus(ctx, repoPath)
	if err != nil {
		return preview, err
	}
	dirty := false
	for _, status := range statuses {
		if status.WorktreeStatus != '?' {
			dirty = true
		}
	}
	preview.Dirty = dirty && configValue(ctx, repoPath, "--type=bool", "rebase.autostash") != "true"
	return preview, nil
}

func isAncestor(ctx context.Context, repoPath, ancestor, of string) bool {
	_, err := runResult(ctx, repoPath, "merge-base", "--is-ancestor", ancestor, of)
	return err == nil
}
