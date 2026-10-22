package git

import (
	"context"
	"strconv"
	"strings"
)

// Heads of merge requests / pull requests fetched from a host live under
// refs/current-client/, never refs/remotes/, so ListRefs and everything built
// on it never sees them.
const hostRefPrefix = "refs/current-client/"

func HostChangeRef(remote string, number int) string {
	return hostRefPrefix + remote + "/mr/" + strconv.Itoa(number)
}

// FetchHostChangeHead fetches remoteRef (refs/merge-requests/N/head on GitLab,
// refs/pull/N/head on GitHub) into HostChangeRef(remote, number).
func FetchHostChangeHead(ctx context.Context, repoPath, remote, remoteRef string, number int) error {
	_, err := runResult(ctx, repoPath, "fetch", "--no-tags", remote, "+"+remoteRef+":"+HostChangeRef(remote, number))
	return err
}

// DeleteHostRefs removes every fetched head for remote, or for all remotes
// when remote is empty.
func DeleteHostRefs(ctx context.Context, repoPath, remote string) error {
	result, err := runResult(ctx, repoPath, "for-each-ref", "--format=%(refname)", hostRefPrefix+remote)
	if err != nil {
		return err
	}
	for _, ref := range strings.Fields(result.Stdout) {
		if _, err := runResult(ctx, repoPath, "update-ref", "-d", ref); err != nil {
			return err
		}
	}
	return nil
}
