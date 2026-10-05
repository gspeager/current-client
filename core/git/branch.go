package git

import (
	"context"
	"regexp"
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
}

func ListBranches(ctx context.Context, repoPath string) ([]Branch, error) {
	result, err := runResult(ctx, repoPath, "branch",
		"--format=%(HEAD)%09%(refname:short)%09%(upstream:short)%09%(upstream:track)%09%(committerdate:iso-strict)")
	if err != nil {
		return nil, err
	}
	return parseLocalBranches(result.Stdout), nil
}

func parseLocalBranches(output string) []Branch {
	var branches []Branch
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "\t", 5)
		if len(fields) != 5 {
			continue
		}
		ahead, behind := parseUpstreamTrack(fields[3])
		branches = append(branches, Branch{
			Name:           fields[1],
			Current:        fields[0] == "*",
			Upstream:       fields[2],
			Ahead:          ahead,
			Behind:         behind,
			LastCommitDate: fields[4],
		})
	}
	return branches
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

func CurrentBranchStatus(ctx context.Context, repoPath string) (BranchStatus, error) {
	current, err := currentBranchName(ctx, repoPath)
	if err != nil {
		return BranchStatus{}, err
	}
	status := BranchStatus{Current: current}
	if current == "HEAD" {
		sha, err := runResult(ctx, repoPath, "rev-parse", "--short", "HEAD")
		if err != nil {
			return status, err
		}
		status.DetachedAt = strings.TrimSpace(sha.Stdout)
		return status, nil
	}

	upstream, err := runResult(ctx, repoPath, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	if err != nil {
		return status, nil
	}
	status.Upstream = strings.TrimSpace(upstream.Stdout)

	counts, err := runResult(ctx, repoPath, "rev-list", "--left-right", "--count", "@{upstream}...HEAD")
	if err != nil {
		return status, err
	}
	fields := strings.Fields(counts.Stdout)
	if len(fields) == 2 {
		status.Behind, _ = strconv.Atoi(fields[0])
		status.Ahead, _ = strconv.Atoi(fields[1])
	}
	return status, nil
}

func ListRemote(ctx context.Context, repoPath string) ([]string, error) {
	result, err := runResult(ctx, repoPath, "branch", "-r", "--format=%(refname)%09%(refname:short)")
	if err != nil {
		return nil, err
	}
	return parseRemoteBranches(result.Stdout), nil
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
	result, err := runResult(ctx, repoPath, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.TrimSpace(result.Stdout), "origin/")
}

func defaultBranch(branches []Branch, remoteHead string) string {
	for _, candidate := range []string{remoteHead, "main", "master"} {
		if candidate == "" {
			continue
		}
		for _, b := range branches {
			if b.Name == candidate {
				return candidate
			}
		}
	}
	return ""
}
