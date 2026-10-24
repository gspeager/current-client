package git

import (
	"context"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type Branch struct {
	Name           string
	Current        bool
	Upstream       string
	Ahead          int
	Behind         int
	LastCommitDate string
	// WorktreePath is the worktree the branch is checked out in, if any.
	WorktreePath string
	Tip          string
}

func ListBranches(ctx context.Context, repoPath string) ([]Branch, error) {
	list, err := listBranches(ctx, repoPath)
	return slices.Clone(list.branches), err
}

type branchList struct {
	branches []Branch
	// goneUpstream holds the branches whose upstream no longer exists.
	goneUpstream map[string]bool
	// head is the current branch's commit, "" when HEAD isn't on a branch.
	head string
}

func listBranches(ctx context.Context, repoPath string) (branchList, error) {
	return cachedByRefs(ctx, repoPath, "branches", func() (branchList, error) {
		result, err := runResult(ctx, repoPath, "branch",
			"--format=%(HEAD)%09%(refname:short)%09%(upstream:short)%09%(upstream:track)%09%(committerdate:iso-strict)%09%(objectname)%09%(worktreepath)")
		if err != nil {
			return branchList{}, err
		}
		return parseLocalBranches(result.Stdout), nil
	})
}

func parseLocalBranches(output string) branchList {
	list := branchList{goneUpstream: map[string]bool{}}
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "\t", 7)
		if len(fields) < 5 {
			continue
		}
		ahead, behind := parseUpstreamTrack(fields[3])
		if fields[3] == "[gone]" {
			list.goneUpstream[fields[1]] = true
		}
		list.branches = append(list.branches, Branch{
			Name:           fields[1],
			Current:        fields[0] == "*",
			Upstream:       fields[2],
			Ahead:          ahead,
			Behind:         behind,
			LastCommitDate: fields[4],
		})
		if len(fields) >= 6 {
			list.branches[len(list.branches)-1].Tip = fields[5]
			if fields[0] == "*" && !strings.HasPrefix(fields[1], "(") {
				list.head = fields[5]
			}
		}
		if len(fields) == 7 && fields[6] != "" {
			list.branches[len(list.branches)-1].WorktreePath = filepath.FromSlash(fields[6])
		}
	}
	return list
}

// Matches either half of %(upstream:track)'s "[ahead N, behind M]".
var upstreamTrackPattern = regexp.MustCompile(`ahead (\d+)|behind (\d+)`)

func parseUpstreamTrack(track string) (ahead, behind int) {
	for _, m := range upstreamTrackPattern.FindAllStringSubmatch(track, -1) {
		if m[1] != "" {
			ahead, _ = strconv.Atoi(m[1])
		}
		if m[2] != "" {
			behind, _ = strconv.Atoi(m[2])
		}
	}
	return ahead, behind
}

func CreateBranch(ctx context.Context, repoPath, name string) error {
	_, err := runResult(ctx, repoPath, "branch", name)
	return err
}

func CreateBranchAt(ctx context.Context, repoPath, name, startPoint string) error {
	_, err := runResult(ctx, repoPath, "branch", name, startPoint)
	return err
}

// CheckoutCommit detaches HEAD at sha, for looking at an old version.
func CheckoutCommit(ctx context.Context, repoPath, sha string) error {
	_, err := runResult(ctx, repoPath, "switch", "--detach", sha)
	return err
}

func CheckoutBranch(ctx context.Context, repoPath, name string) error {
	_, err := runResult(ctx, repoPath, "checkout", name)
	return err
}

// SetUpstream makes branch track remoteBranch, a remote-tracking branch such
// as "origin/feature". The full ref keeps a local branch of the same name from
// being picked instead.
func SetUpstream(ctx context.Context, repoPath, branch, remoteBranch string) error {
	_, err := runResult(ctx, repoPath, "branch", "--set-upstream-to=refs/remotes/"+remoteBranch, branch)
	return err
}

func UnsetUpstream(ctx context.Context, repoPath, branch string) error {
	_, err := runResult(ctx, repoPath, "branch", "--unset-upstream", branch)
	return err
}

func RenameBranch(ctx context.Context, repoPath, oldName, newName string) error {
	_, err := runResult(ctx, repoPath, "branch", "-m", oldName, newName)
	return err
}

func DeleteBranch(ctx context.Context, repoPath, name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := runResult(ctx, repoPath, "branch", flag, name)
	return err
}

type BranchStatus struct {
	Current  string
	Upstream string
	Ahead    int
	Behind   int
	// DetachedAt is HEAD's short SHA when HEAD is detached; Current is then "HEAD".
	DetachedAt string
}

// CurrentBranchStatus reads the current branch's row of the cached branch
// list; only a detached or unborn HEAD needs its own git process.
func CurrentBranchStatus(ctx context.Context, repoPath string) (BranchStatus, error) {
	list, err := listBranches(ctx, repoPath)
	if err != nil {
		return BranchStatus{}, err
	}
	for _, b := range list.branches {
		if !b.Current {
			continue
		}
		// git branch lists a detached HEAD, or one mid-rebase, as "(HEAD detached at …)".
		if strings.HasPrefix(b.Name, "(") {
			sha, err := runResult(ctx, repoPath, "rev-parse", "--short", "HEAD")
			if err != nil {
				return BranchStatus{Current: "HEAD"}, err
			}
			return BranchStatus{Current: "HEAD", DetachedAt: strings.TrimSpace(sha.Stdout)}, nil
		}
		if list.goneUpstream[b.Name] {
			return BranchStatus{Current: b.Name}, nil
		}
		return BranchStatus{Current: b.Name, Upstream: b.Upstream, Ahead: b.Ahead, Behind: b.Behind}, nil
	}
	// An unborn branch has no row yet.
	unborn, err := runResult(ctx, repoPath, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return BranchStatus{}, err
	}
	return BranchStatus{Current: strings.TrimSpace(unborn.Stdout)}, nil
}

func ListRemote(ctx context.Context, repoPath string) ([]string, error) {
	branches, err := cachedByRefs(ctx, repoPath, "remote-branches", func() ([]string, error) {
		result, err := runResult(ctx, repoPath, "branch", "-r", "--format=%(refname)%09%(refname:short)")
		if err != nil {
			return nil, err
		}
		return parseRemoteBranches(result.Stdout), nil
	})
	return slices.Clone(branches), err
}

// BranchesContaining lists the local branches whose history includes sha.
// for-each-ref, not branch, so a detached HEAD isn't listed as "(HEAD detached at …)".
func BranchesContaining(ctx context.Context, repoPath, sha string) ([]string, error) {
	result, err := runResult(ctx, repoPath, "for-each-ref", "refs/heads", "--contains", sha, "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}
	return strings.Fields(result.Stdout), nil
}

func parseRemoteBranches(output string) []string {
	var branches []string
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "\t", 2)
		if len(fields) != 2 || strings.HasSuffix(fields[0], "/HEAD") {
			continue
		}
		branches = append(branches, fields[1])
	}
	return branches
}

// DefaultBranch prefers the branch origin/HEAD names, then a local main or
// master, and returns "" when none of those exists locally.
func DefaultBranch(ctx context.Context, repoPath string) (string, error) {
	branches, err := ListBranches(ctx, repoPath)
	if err != nil {
		return "", err
	}
	return defaultBranch(branches, originHead(ctx, repoPath)), nil
}

// originHead returns "" when origin/HEAD isn't set, e.g. no origin remote or
// a repo that was never cloned.
func originHead(ctx context.Context, repoPath string) string {
	head, _ := cachedByRefs(ctx, repoPath, "origin-head", func() (string, error) {
		result, err := runResult(ctx, repoPath, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD")
		if err != nil {
			return "", nil
		}
		return strings.TrimPrefix(strings.TrimSpace(result.Stdout), "origin/"), nil
	})
	return head
}

func defaultBranch(branches []Branch, remoteHead string) string {
	for _, candidate := range []string{remoteHead, "main", "master"} {
		if candidate != "" && hasBranch(branches, candidate) {
			return candidate
		}
	}
	return ""
}
