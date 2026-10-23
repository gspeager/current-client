package git

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

// MergedBranch is a local branch whose work is already in Base. Newer means
// the branch has commits past its upstream, which is what was merged.
type MergedBranch struct {
	Branch
	Newer bool
}

// MergedReport lists the branches with an upstream that add nothing to Base,
// the default branch's upstream when it has one. A squash or a rebase merge
// is found as well as a merge commit: merging the branch in again would
// change nothing.
type MergedReport struct {
	Base     string
	Branches []MergedBranch
}

func MergedBranches(ctx context.Context, repoPath string) (MergedReport, error) {
	list, err := listBranches(ctx, repoPath)
	if err != nil {
		return MergedReport{}, err
	}
	def := defaultBranch(list.branches, originHead(ctx, repoPath))
	if def == "" {
		return MergedReport{}, nil
	}
	report := MergedReport{Base: def}
	var candidates []Branch
	var tips []string
	for _, b := range list.branches {
		if b.Name == def {
			if b.Upstream != "" && !list.goneUpstream[b.Name] {
				report.Base = b.Upstream
			}
			continue
		}
		if b.Upstream == "" || (b.WorktreePath != "" && !b.Current) {
			continue
		}
		candidates = append(candidates, b)
		tips = append(tips, b.Tip)
		if b.Ahead > 0 && !list.goneUpstream[b.Name] {
			tips = append(tips, b.Upstream)
		}
	}
	merged, err := MergedInto(ctx, repoPath, report.Base, tips)
	if err != nil {
		return MergedReport{}, err
	}
	i := 0
	for _, b := range candidates {
		local := merged[i]
		i++
		upstream := false
		if b.Ahead > 0 && !list.goneUpstream[b.Name] {
			upstream = merged[i]
			i++
		}
		if local || upstream {
			report.Branches = append(report.Branches, MergedBranch{Branch: b, Newer: !local})
		}
	}
	return report, nil
}

// MergedInto reports for each tip whether merging it into base would leave
// base's tree as it is. Nothing in the working tree or index is touched.
func MergedInto(ctx context.Context, repoPath, base string, tips []string) ([]bool, error) {
	if len(tips) == 0 {
		return nil, nil
	}
	baseTree, err := runResult(ctx, repoPath, "rev-parse", "--verify", base+"^{tree}")
	if err != nil {
		return nil, err
	}
	trees, err := mergedTrees(ctx, repoPath, base, tips)
	if err != nil {
		trees, err = mergedTreesOneByOne(ctx, repoPath, base, tips)
	}
	if err != nil {
		return nil, err
	}
	merged := make([]bool, len(tips))
	for i, tree := range trees {
		merged[i] = tree == strings.TrimSpace(baseTree.Stdout)
	}
	return merged, nil
}

// The tree each merge would produce, "" for one that conflicts.
func mergedTrees(ctx context.Context, repoPath, base string, tips []string) ([]string, error) {
	var input strings.Builder
	for _, tip := range tips {
		fmt.Fprintf(&input, "%s %s\n", base, tip)
	}
	result, err := gitexec.NewExecutor("").Run(ctx, gitexec.Command{
		Dir:   repoPath,
		Args:  []string{"merge-tree", "--write-tree", "--stdin", "--name-only", "--no-messages", "-z"},
		Stdin: strings.NewReader(input.String()),
	})
	if err != nil {
		return nil, gitexec.WrapRunError(err)
	}
	if result.ExitCode != 0 {
		return nil, mergeTreeError(result)
	}
	merges, err := parseMergeTreeStdin(result.Stdout, len(tips))
	if err != nil {
		return nil, err
	}
	trees := make([]string, len(tips))
	for i, merge := range merges {
		if merge.clean {
			trees[i] = merge.tree
		}
	}
	return trees, nil
}

// For Git before 2.40, and for a batch merge-tree gave up on, such as one
// with a branch sharing no history; that one is simply not merged.
func mergedTreesOneByOne(ctx context.Context, repoPath, base string, tips []string) ([]string, error) {
	trees := make([]string, len(tips))
	for i, tip := range tips {
		result, err := gitexec.NewExecutor("").Run(ctx, gitexec.Command{
			Dir:  repoPath,
			Args: []string{"merge-tree", "--write-tree", "--no-messages", base, tip},
		})
		if err != nil {
			return nil, gitexec.WrapRunError(err)
		}
		switch err := mergeTreeError(result); {
		case result.ExitCode == 0:
			trees[i] = strings.TrimSpace(result.Stdout)
		case result.ExitCode == 1, errors.Is(err, ErrUnrelatedHistories):
		default:
			return nil, err
		}
	}
	return trees, nil
}
