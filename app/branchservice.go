package app

import (
	"context"
	"strconv"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
)

type BranchService struct{}

type BranchInfo struct {
	Name           string `json:"name"`
	Current        bool   `json:"current"`
	Upstream       string `json:"upstream"`
	Ahead          int    `json:"ahead"`
	Behind         int    `json:"behind"`
	LastCommitDate string `json:"lastCommitDate"`
	WorktreePath   string `json:"worktreePath"`
	Tip            string `json:"tip"`
}

type BranchStatusInfo struct {
	Current    string `json:"current"`
	Upstream   string `json:"upstream"`
	Ahead      int    `json:"ahead"`
	Behind     int    `json:"behind"`
	DetachedAt string `json:"detachedAt"`
}

func (s *BranchService) ListLocal(repoPath string) ([]BranchInfo, error) {
	branches, err := git.ListBranches(context.Background(), repoPath)
	if err != nil {
		return nil, err
	}
	return mapSlice(branches, func(b git.Branch) BranchInfo { return BranchInfo(b) }), nil
}

func (s *BranchService) ListRemote(repoPath string) ([]string, error) {
	return git.ListRemote(context.Background(), repoPath)
}

func (s *BranchService) CreateBranch(repoPath, name string) error {
	return git.CreateBranch(context.Background(), repoPath, name)
}

func (s *BranchService) CheckoutCommit(repoPath, sha string) error {
	return git.CheckoutCommit(context.Background(), repoPath, sha)
}

func (s *BranchService) CheckoutBranch(repoPath, name string) error {
	snapshotBefore(repoPath, "checkout")
	return git.CheckoutBranch(context.Background(), repoPath, name)
}

func (s *BranchService) CreateBranchAt(repoPath, name, startPoint string) error {
	return git.CreateBranchAt(context.Background(), repoPath, name, startPoint)
}

func (s *BranchService) SetUpstream(repoPath, branch, remoteBranch string) error {
	return git.SetUpstream(context.Background(), repoPath, branch, remoteBranch)
}

func (s *BranchService) UnsetUpstream(repoPath, branch string) error {
	return git.UnsetUpstream(context.Background(), repoPath, branch)
}

func (s *BranchService) RenameBranch(repoPath, oldName, newName string) error {
	return git.RenameBranch(context.Background(), repoPath, oldName, newName)
}

// DeleteBranch returns the deleted branch's tip, so it can be recreated.
func (s *BranchService) DeleteBranch(repoPath, name string, force bool) (string, error) {
	ctx := context.Background()
	tip, err := git.ResolveCommit(ctx, repoPath, "refs/heads/"+name)
	if err != nil {
		return "", err
	}
	// Only when deleting it would leave its commits unreachable.
	if force && !git.CommitOnAnotherRef(ctx, repoPath, tip, "refs/heads/"+name) {
		snapshotTip(repoPath, "branch delete", name, tip)
	}
	return tip, git.DeleteBranch(ctx, repoPath, name, force)
}

// MergeBranch takes mode "" (fast-forward when possible), "no-ff" or "squash".
func (s *BranchService) MergeBranch(repoPath, branch, mode string) error {
	m := git.MergeMode(mode)
	if m != git.MergeFastForward && m != git.MergeCommit && m != git.MergeSquash {
		return &gitexec.AppError{Message: "Unknown merge option.", Detail: mode}
	}
	return git.MergeBranchMode(context.Background(), repoPath, branch, m)
}

func (s *BranchService) DefaultBranch(repoPath string) (string, error) {
	return git.DefaultBranch(context.Background(), repoPath)
}

type RebasePreviewCommitInfo struct {
	SHA     string   `json:"sha"`
	Subject string   `json:"subject"`
	Outcome string   `json:"outcome"`
	Files   []string `json:"files"`
}

type RebasePreviewInfo struct {
	Current       string                    `json:"current"`
	Onto          string                    `json:"onto"`
	Commits       []RebasePreviewCommitInfo `json:"commits"`
	Predicted     bool                      `json:"predicted"`
	DroppedMerges int                       `json:"droppedMerges"`
	ForcePush     bool                      `json:"forcePush"`
	Upstream      string                    `json:"upstream"`
	Dirty         bool                      `json:"dirty"`
	Command       string                    `json:"command"`
}

func (s *BranchService) PreviewRebase(repoPath, onto string) (RebasePreviewInfo, error) {
	preview, err := git.PreviewRebase(context.Background(), repoPath, onto)
	if err != nil {
		return RebasePreviewInfo{}, err
	}
	return RebasePreviewInfo{
		Current: preview.Current,
		Onto:    preview.Onto,
		Commits: mapSlice(preview.Commits, func(commit git.PreviewCommit) RebasePreviewCommitInfo {
			return RebasePreviewCommitInfo{commit.SHA, commit.Subject, string(commit.Outcome), append([]string{}, commit.Files...)}
		}),
		Predicted:     preview.Predicted,
		DroppedMerges: preview.DroppedMerges,
		ForcePush:     preview.ForcePush,
		Upstream:      preview.Upstream,
		Dirty:         preview.Dirty,
		Command:       preview.Command,
	}, nil
}

func (s *BranchService) RebaseOnto(repoPath, onto string) error {
	return git.RebaseOnto(context.Background(), repoPath, onto)
}

func (s *BranchService) CurrentBranchStatus(repoPath string) (BranchStatusInfo, error) {
	status, err := git.CurrentBranchStatus(context.Background(), repoPath)
	return BranchStatusInfo(status), err
}

// MergedBranchInfo is a local branch whose work is already in Base, found
// with Git alone, or merged on the host as change Number when the cache
// knows that. Newer means the branch has commits past what was merged.
type MergedBranchInfo struct {
	Name     string `json:"name"`
	Current  bool   `json:"current"`
	Upstream string `json:"upstream"`
	Tip      string `json:"tip"`
	Newer    bool   `json:"newer"`
	Number   int    `json:"number"`
	Prefix   string `json:"prefix"`
}

type MergedBranches struct {
	Default  string             `json:"default"`
	Base     string             `json:"base"`
	Branches []MergedBranchInfo `json:"branches"`
}

// MergedBranches makes no request: the host's part comes from the cache.
func (s *BranchService) MergedBranches(repoPath string) (MergedBranches, error) {
	ctx := context.Background()
	report, err := git.MergedBranches(ctx, repoPath)
	if err != nil {
		return MergedBranches{}, err
	}
	def, err := git.DefaultBranch(ctx, repoPath)
	if err != nil {
		return MergedBranches{}, err
	}
	result := MergedBranches{Default: def, Base: report.Base}
	listed := map[string]int{}
	for _, b := range report.Branches {
		listed[b.Name] = len(result.Branches)
		result.Branches = append(result.Branches, MergedBranchInfo{
			Name: b.Name, Current: b.Current, Upstream: b.Upstream, Tip: b.Tip, Newer: b.Newer,
		})
	}
	onHost := mergedOnHost(repoPath)
	if len(onHost) == 0 {
		return result, nil
	}
	branches, err := git.ListBranches(ctx, repoPath)
	if err != nil {
		return MergedBranches{}, err
	}
	for _, b := range branches {
		change, ok := onHost[b.Upstream]
		if !ok || b.Name == def {
			continue
		}
		if i, ok := listed[b.Name]; ok {
			result.Branches[i].Number, result.Branches[i].Prefix = change.Number, change.Prefix
			continue
		}
		result.Branches = append(result.Branches, MergedBranchInfo{
			Name: b.Name, Current: b.Current, Upstream: b.Upstream, Tip: b.Tip,
			Newer: b.Tip != change.HeadSHA, Number: change.Number, Prefix: change.Prefix,
		})
	}
	return result, nil
}

// BranchReviewRowInfo.Deletable is false for the current branch, the default
// branch and a branch checked out in a worktree.
type BranchReviewRowInfo struct {
	Name           string       `json:"name"`
	Current        bool         `json:"current"`
	Upstream       string       `json:"upstream"`
	Ahead          int          `json:"ahead"`
	Behind         int          `json:"behind"`
	AheadBase      int          `json:"aheadBase"`
	BehindBase     int          `json:"behindBase"`
	Merged         bool         `json:"merged"`
	MergedChange   string       `json:"mergedChange"`
	OpenChange     string       `json:"openChange"`
	Stale          bool         `json:"stale"`
	Unpushed       bool         `json:"unpushed"`
	LastCommitDate string       `json:"lastCommitDate"`
	AuthorName     string       `json:"authorName"`
	AuthorEmail    string       `json:"authorEmail"`
	Author         IdentityInfo `json:"author"`
	Deletable      bool         `json:"deletable"`
}

type BranchReviewInfo struct {
	Base string                `json:"base"`
	Rows []BranchReviewRowInfo `json:"rows"`
}

// ReviewBranches makes no request: the host's part comes from the cache.
func (s *BranchService) ReviewBranches(repoPath string) (BranchReviewInfo, error) {
	review, err := git.ReviewBranches(context.Background(), repoPath)
	if err != nil {
		return BranchReviewInfo{}, err
	}
	info := BranchReviewInfo{Base: review.Base, Rows: []BranchReviewRowInfo{}}
	merged, open := mergedOnHost(repoPath), openChangesByBranch(repoPath)
	cfgIdentities := (&IdentityService{}).GetIdentities(mapSlice(review.Rows, func(row git.BranchReviewRow) IdentityRef {
		return IdentityRef{Name: row.AuthorName, Email: row.AuthorEmail}
	}))
	now := time.Now()
	for i, row := range review.Rows {
		last, err := time.Parse(time.RFC3339, row.LastCommitDate)
		mergedChange := ""
		// Commits added since it merged aren't merged, so they aren't called so.
		if change, ok := merged[row.Upstream]; ok && row.Tip == change.HeadSHA {
			mergedChange = change.Prefix + strconv.Itoa(change.Number)
		}
		info.Rows = append(info.Rows, BranchReviewRowInfo{
			Name:           row.Name,
			Current:        row.Current,
			Upstream:       row.Upstream,
			Ahead:          row.Ahead,
			Behind:         row.Behind,
			AheadBase:      row.AheadBase,
			BehindBase:     row.BehindBase,
			Merged:         row.Name != review.Default && (row.Merged || mergedChange != ""),
			MergedChange:   mergedChange,
			OpenChange:     open[row.Name],
			Stale:          err == nil && !row.Current && now.Sub(last) > staleAfter,
			Unpushed:       row.Upstream == "" || row.Ahead > 0,
			LastCommitDate: row.LastCommitDate,
			AuthorName:     row.AuthorName,
			AuthorEmail:    row.AuthorEmail,
			Author:         cfgIdentities[i],
			Deletable:      !row.Current && row.Name != review.Default && row.WorktreePath == "",
		})
	}
	return info, nil
}

// hostMergedChange is what the cache knows of a merged change, by its
// source branch's remote-tracking name.
type hostMergedChange struct {
	Number  int
	Prefix  string
	HeadSHA string
}
