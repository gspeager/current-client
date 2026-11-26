package git

import (
	"context"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
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

// How far back RefAt looks in a branch ahead of the commit.
const mostAhead = 500

// RefAt is a ref at sha: one named branch before any other, and a local branch
// before a remote one before a fetched change head. Else it's the local branch
// of that name when it's ahead of sha, and "" when there's none. It never reads
// the commit, which a partial clone would fetch.
func RefAt(ctx context.Context, repoPath, sha, branch string) (string, error) {
	if sha == "" {
		return "", nil
	}
	result, err := gitexec.RunIn(ctx, repoPath, "for-each-ref", "--points-at="+sha, "--format=%(refname)", "refs/heads", "refs/remotes", hostRefPrefix)
	if err != nil {
		return "", err
	}
	at := slices.DeleteFunc(strings.Fields(result.Stdout), func(ref string) bool { return strings.HasSuffix(ref, "/HEAD") })
	named := func(ref string) bool { return strings.HasSuffix(ref, "/"+branch) }
	anyName := func(string) bool { return true }
	for _, accept := range []func(string) bool{named, anyName} {
		for _, prefix := range []string{"refs/heads/", "refs/remotes/", hostRefPrefix} {
			for _, ref := range at {
				if strings.HasPrefix(ref, prefix) && accept(ref) {
					return ref, nil
				}
			}
		}
	}
	local := "refs/heads/" + branch
	ahead, err := gitexec.RunIn(ctx, repoPath, "rev-list", "--max-count="+strconv.Itoa(mostAhead), local)
	if err == nil && slices.Contains(strings.Fields(ahead.Stdout), sha) {
		return local, nil
	}
	return "", nil
}

// FetchHostChangeHead fetches remoteRef (refs/merge-requests/N/head on GitLab,
// refs/pull/N/head on GitHub) into HostChangeRef(remote, number).
func FetchHostChangeHead(ctx context.Context, repoPath, remote, remoteRef string, number int) error {
	return FetchHostChangeHeads(ctx, repoPath, remote, map[int]string{number: remoteRef})
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
	_, err := gitexec.RunIn(ctx, repoPath, args...)
	return err
}

// DeleteHostRefs removes every fetched head for remote, or for all remotes
// when remote is empty. It also finds heads 0.5.0 and earlier fetched directly
// under the namespace, at refs/current-client/<remote>/mr/<number>.
func DeleteHostRefs(ctx context.Context, repoPath, remote string) error {
	result, err := gitexec.RunIn(ctx, repoPath, "for-each-ref", "--format=%(refname)", appRefNamespace)
	if err != nil {
		return err
	}
	for _, ref := range strings.Fields(result.Stdout) {
		if !isHostRef(ref, remote) {
			continue
		}
		if _, err := gitexec.RunIn(ctx, repoPath, "update-ref", "-d", ref); err != nil {
			return err
		}
	}
	return nil
}

var hostRefShape = regexp.MustCompile(`^(.+)/mr/\d+$`)

func isHostRef(ref, remote string) bool {
	match := hostRefShape.FindStringSubmatch(strings.TrimPrefix(ref, appRefNamespace))
	if match == nil {
		return false
	}
	return remote == "" || match[1] == remote || match[1] == "hosts/"+remote
}
