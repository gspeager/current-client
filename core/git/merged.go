package git

import (
	"context"
	"strings"
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

// DefaultBase is the default branch, and what merged is judged against: its
// upstream when it has one. Both are "" when there's no default branch.
func DefaultBase(ctx context.Context, repoPath string) (def, base string, err error) {
	list, err := listBranches(ctx, repoPath)
	if err != nil {
		return "", "", err
	}
	def, base = list.defaultBase(ctx, repoPath)
	return def, base, nil
}

func (list branchList) defaultBase(ctx context.Context, repoPath string) (def, base string) {
	if def = defaultBranch(list.branches, originHead(ctx, repoPath)); def == "" {
		return "", ""
	}
	return def, list.base(def)
}

func MergedBranches(ctx context.Context, repoPath string) (MergedReport, error) {
	list, err := listBranches(ctx, repoPath)
	if err != nil {
		return MergedReport{}, err
	}
	def, base := list.defaultBase(ctx, repoPath)
	if def == "" {
		return MergedReport{}, nil
	}
	report := MergedReport{Base: base}
	pastUpstream := func(b Branch) bool { return b.Ahead > 0 && list.upstream(b) != "" }
	var candidates []Branch
	var tips []string
	for _, b := range list.branches {
		if b.Name == def || b.Upstream == "" || (b.WorktreePath != "" && !b.Current) {
			continue
		}
		candidates = append(candidates, b)
		tips = append(tips, b.Tip)
		if pastUpstream(b) {
			tips = append(tips, b.Upstream)
		}
	}
	merged, err := mergedWithoutFetching(ctx, repoPath, report.Base, tips)
	if err != nil {
		return MergedReport{}, err
	}
	i := 0
	for _, b := range candidates {
		local := merged[i]
		i++
		upstream := false
		if pastUpstream(b) {
			upstream = merged[i]
			i++
		}
		if local || upstream {
			report.Branches = append(report.Branches, MergedBranch{Branch: b, Newer: !local})
		}
	}
	return report, nil
}

// mergedWithoutFetching is MergedInto where nobody started a Git operation. In a partial
// clone a merge would fetch, so there only a tip in base's history counts, and a
// squash or rebase merge isn't found. Tips are branches, or their commits.
func mergedWithoutFetching(ctx context.Context, repoPath, base string, tips []string) ([]bool, error) {
	if !PartialClone(ctx, repoPath) {
		return MergedInto(ctx, repoPath, base, tips)
	}
	result, err := runResult(ctx, repoPath, "for-each-ref", "--merged="+base, "--format=%(objectname) %(refname:short)",
		"refs/heads", "refs/remotes")
	if err != nil {
		return nil, err
	}
	inBase := map[string]bool{}
	for _, name := range strings.Fields(result.Stdout) {
		inBase[name] = true
	}
	merged := make([]bool, len(tips))
	for i, tip := range tips {
		merged[i] = inBase[tip]
	}
	return merged, nil
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
	pairs := make([][2]string, len(tips))
	for i, tip := range tips {
		pairs[i] = [2]string{base, tip}
	}
	merges, err := mergePairsDiscarding(ctx, repoPath, pairs)
	if err != nil {
		return nil, err
	}
	merged := make([]bool, len(tips))
	for i, merge := range merges {
		merged[i] = merge.Clean && merge.Tree == strings.TrimSpace(baseTree.Stdout)
	}
	return merged, nil
}
