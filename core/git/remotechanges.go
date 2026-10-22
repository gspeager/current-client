package git

import (
	"context"
	"strconv"
	"strings"
)

// ChangeRefKind is the namespace a host publishes pull request / merge
// request heads under, which also identifies the kind of host without
// its API.
type ChangeRefKind string

const (
	// PullRefs is refs/pull/N/head: GitHub, Gitea and Forgejo.
	PullRefs ChangeRefKind = "pull"
	// MergeRequestRefs is refs/merge-requests/N/head: GitLab.
	MergeRequestRefs ChangeRefKind = "merge-requests"
)

// RemoteChanges maps each remote branch to the number of its pull request /
// merge request. On PullRefs hosts that is the open one; on GitLab, Git
// alone can't tell a closed merge request whose branch was kept from an
// open one, so it is the newest whose head is still the branch tip and the
// state has to come from the API. Kind is empty, and ByBranch empty, for a
// host that publishes no such refs (Bitbucket).
type RemoteChanges struct {
	Kind     ChangeRefKind
	ByBranch map[string]int
}

// ListRemoteChanges runs one ls-remote against remote. A change's head must
// still be a branch tip, which drops every change whose branch was deleted
// on merge. GitHub publishes refs/pull/N/merge only while a change is open,
// so where any merge ref exists one is required too; GitLab keeps merge refs
// after closing, so they are ignored there.
func ListRemoteChanges(ctx context.Context, repoPath, remote string) (RemoteChanges, error) {
	result, err := runResult(ctx, repoPath, "ls-remote", "--refs", remote,
		"refs/heads/*", "refs/pull/*", "refs/merge-requests/*")
	if err != nil {
		return RemoteChanges{}, err
	}
	return parseRemoteChanges(result.Stdout), nil
}

func parseRemoteChanges(output string) RemoteChanges {
	tips := map[string]string{}
	heads := map[ChangeRefKind]map[int]string{PullRefs: {}, MergeRequestRefs: {}}
	merges := map[ChangeRefKind]map[int]bool{PullRefs: {}, MergeRequestRefs: {}}
	for _, line := range strings.Split(output, "\n") {
		sha, ref, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		if branch, ok := strings.CutPrefix(ref, "refs/heads/"); ok {
			tips[sha] = branch
			continue
		}
		for kind := range heads {
			rest, ok := strings.CutPrefix(ref, "refs/"+string(kind)+"/")
			if !ok {
				continue
			}
			numberText, suffix, _ := strings.Cut(rest, "/")
			number, err := strconv.Atoi(numberText)
			if err != nil {
				continue
			}
			switch suffix {
			case "head":
				heads[kind][number] = sha
			case "merge":
				merges[kind][number] = true
			}
		}
	}

	changes := RemoteChanges{ByBranch: map[string]int{}}
	for _, kind := range []ChangeRefKind{PullRefs, MergeRequestRefs} {
		if len(heads[kind]) == 0 {
			continue
		}
		changes.Kind = kind
		requireMerge := kind == PullRefs && len(merges[kind]) > 0
		for number, sha := range heads[kind] {
			branch, ok := tips[sha]
			if !ok || (requireMerge && !merges[kind][number]) {
				continue
			}
			if number > changes.ByBranch[branch] {
				changes.ByBranch[branch] = number
			}
		}
		break
	}
	return changes
}
