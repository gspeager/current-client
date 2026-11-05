package git

import (
	"context"
	"strconv"
	"strings"
)

const appRefNamespace = "refs/current-client/"

// Heads of merge requests / pull requests fetched from a host live under
// hosts/ in the app's namespace, never refs/remotes/, so ListRefs and
// everything built on it never sees them, and turning a host off can't reach
// the app's local refs beside them.
const hostRefPrefix = appRefNamespace + "hosts/"

// allRefs is --all without what isn't history: stash entries and the app's own
// refs. An --exclude only applies to the --all after it, and it needs the * to
// reach nested refs.
var allRefs = []string{"--exclude=refs/stash", "--exclude=" + appRefNamespace + "*", "--all"}

func HostChangeRef(remote string, number int) string {
	return hostRefPrefix + remote + "/mr/" + strconv.Itoa(number)
}

// FetchHostChangeHead fetches remoteRef (refs/merge-requests/N/head on GitLab,
// refs/pull/N/head on GitHub) into HostChangeRef(remote, number).
func FetchHostChangeHead(ctx context.Context, repoPath, remote, remoteRef string, number int) error {
	_, err := runResult(ctx, repoPath, "fetch", "--no-tags", remote, "+"+remoteRef+":"+HostChangeRef(remote, number))
	return err
}

// FetchHostChangeHeads fetches several changes in one go; refs maps each
// number to its ref on the remote.
func FetchHostChangeHeads(ctx context.Context, repoPath, remote string, refs map[int]string) error {
	if len(refs) == 0 {
		return nil
	}
	args := []string{"fetch", "--no-tags", remote}
	for number, remoteRef := range refs {
		args = append(args, "+"+remoteRef+":"+HostChangeRef(remote, number))
	}
	_, err := runResult(ctx, repoPath, args...)
	return err
}

// DeleteHostRefs removes every fetched head for remote, or for all remotes
// when remote is empty. It also finds heads 0.5.0 and earlier fetched directly
// under the namespace, at refs/current-client/<remote>/mr/<number>.
func DeleteHostRefs(ctx context.Context, repoPath, remote string) error {
	result, err := runResult(ctx, repoPath, "for-each-ref", "--format=%(refname)", appRefNamespace)
	if err != nil {
		return err
	}
	for _, ref := range strings.Fields(result.Stdout) {
		if !isHostRef(ref, remote) {
			continue
		}
		if _, err := runResult(ctx, repoPath, "update-ref", "-d", ref); err != nil {
			return err
		}
	}
	return nil
}

func isHostRef(ref, remote string) bool {
	name := strings.TrimPrefix(strings.TrimPrefix(ref, appRefNamespace), "hosts/")
	if remote == "" {
		return strings.Contains(name, "/mr/")
	}
	return strings.HasPrefix(name, remote+"/mr/")
}
