package git

import (
	"context"
	"errors"
	"strconv"
	"strings"
)

// BranchReviewRow is a local branch with what's needed to judge it. Ahead and
// Behind in Branch are against its upstream; the Base counts against the base.
type BranchReviewRow struct {
	Branch
	AuthorName  string
	AuthorEmail string
	AheadBase   int
	BehindBase  int
	Merged      bool
}

// BranchReview holds every local branch, compared with Base: the default
// branch's upstream when it has one, or the default branch, which is "" when
// the repository has none.
type BranchReview struct {
	Default string
	Base    string
	Rows    []BranchReviewRow
}

func ReviewBranches(ctx context.Context, repoPath string) (BranchReview, error) {
	list, err := listBranches(ctx, repoPath)
	if err != nil {
		return BranchReview{}, err
	}
	review := BranchReview{Default: defaultBranch(list.branches, originHead(ctx, repoPath))}
	review.Base = list.base(review.Default)
	for _, b := range list.branches {
		review.Rows = append(review.Rows, BranchReviewRow{Branch: b})
	}
	authors, err := branchAuthors(ctx, repoPath)
	if err != nil {
		return BranchReview{}, err
	}
	for i := range review.Rows {
		author := authors[review.Rows[i].Name]
		review.Rows[i].AuthorName, review.Rows[i].AuthorEmail = author[0], author[1]
	}
	if review.Base == "" {
		return review, nil
	}
	if err := fillBaseCounts(ctx, repoPath, review.Base, review.Rows); err != nil {
		return BranchReview{}, err
	}
	if err := fillMerged(ctx, repoPath, review.Base, review.Rows); err != nil {
		return BranchReview{}, err
	}
	return review, nil
}

func branchAuthors(ctx context.Context, repoPath string) (map[string][2]string, error) {
	result, err := runResult(ctx, repoPath, "for-each-ref", "refs/heads", "--format=%(refname:short)%00%(authorname)%00%(authoremail:trim)")
	if err != nil {
		return nil, err
	}
	authors := map[string][2]string{}
	for _, line := range strings.Split(result.Stdout, "\n") {
		if fields := strings.Split(line, "\x00"); len(fields) == 3 {
			authors[fields[0]] = [2]string{fields[1], fields[2]}
		}
	}
	return authors, nil
}

// fillBaseCounts asks for all of them at once on Git 2.41 or newer, which added
// ahead-behind to for-each-ref, and one branch at a time before it.
func fillBaseCounts(ctx context.Context, repoPath, base string, rows []BranchReviewRow) error {
	result, err := runResult(ctx, repoPath, "for-each-ref", "refs/heads", "--format=%(refname:short)%00%(ahead-behind:"+base+")")
	if err == nil {
		counts := map[string][2]int{}
		for _, line := range strings.Split(result.Stdout, "\n") {
			if name, pair, ok := strings.Cut(line, "\x00"); ok {
				counts[name] = parseCountPair(pair)
			}
		}
		for i := range rows {
			count := counts[rows[i].Name]
			rows[i].AheadBase, rows[i].BehindBase = count[0], count[1]
		}
		return nil
	}
	for i := range rows {
		result, err := runResult(ctx, repoPath, "rev-list", "--left-right", "--count", rows[i].Tip+"..."+base)
		if err != nil {
			return err
		}
		count := parseCountPair(result.Stdout)
		rows[i].AheadBase, rows[i].BehindBase = count[0], count[1]
	}
	return nil
}

func parseCountPair(text string) [2]int {
	fields := strings.Fields(text)
	if len(fields) != 2 {
		return [2]int{}
	}
	ahead, _ := strconv.Atoi(fields[0])
	behind, _ := strconv.Atoi(fields[1])
	return [2]int{ahead, behind}
}

// fillMerged finds squash and rebase merges as well as merge commits, on Git
// 2.38 or newer. Before it, only a branch whose commits are all in base counts.
func fillMerged(ctx context.Context, repoPath, base string, rows []BranchReviewRow) error {
	tips := make([]string, len(rows))
	for i := range rows {
		tips[i] = rows[i].Tip
	}
	merged, err := MergedInto(ctx, repoPath, base, tips)
	if errors.Is(err, ErrMergeTreeUnsupported) {
		for i := range rows {
			rows[i].Merged = rows[i].AheadBase == 0
		}
		return nil
	}
	if err != nil {
		return err
	}
	for i := range rows {
		rows[i].Merged = merged[i]
	}
	return nil
}
