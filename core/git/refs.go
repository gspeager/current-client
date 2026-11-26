package git

import (
	"context"
	"strconv"
	"strings"

	"github.com/gspeager/current-client/core/gitexec"
)

type RefKind int

const (
	RefBranch RefKind = iota
	RefRemoteBranch
	RefTag
)

type Ref struct {
	Kind RefKind
	Name string
	SHA  string
	// Unix seconds, when the commit was made; 0 for an annotated tag.
	CommittedAt int64
}

// ListRefs resolves annotated tags to the commit they point at.
func ListRefs(ctx context.Context, repoPath string) ([]Ref, error) {
	result, err := gitexec.RunIn(ctx, repoPath, "for-each-ref",
		"--format=%(objectname)%09%(*objectname)%09%(committerdate:unix)%09%(refname)",
		"refs/heads", "refs/remotes", "refs/tags")
	if err != nil {
		return nil, err
	}
	return parseRefs(result.Stdout), nil
}

func parseRefs(output string) []Ref {
	var refs []Ref
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "\t", 4)
		if len(fields) != 4 {
			continue
		}
		sha := fields[0]
		if fields[1] != "" {
			sha = fields[1]
		}
		committedAt, _ := strconv.ParseInt(fields[2], 10, 64)
		refname := fields[3]

		switch {
		case strings.HasPrefix(refname, "refs/heads/"):
			refs = append(refs, Ref{Kind: RefBranch, Name: strings.TrimPrefix(refname, "refs/heads/"), SHA: sha, CommittedAt: committedAt})
		case strings.HasPrefix(refname, "refs/remotes/"):
			name := strings.TrimPrefix(refname, "refs/remotes/")
			if strings.HasSuffix(name, "/HEAD") {
				continue
			}
			refs = append(refs, Ref{Kind: RefRemoteBranch, Name: name, SHA: sha, CommittedAt: committedAt})
		case strings.HasPrefix(refname, "refs/tags/"):
			refs = append(refs, Ref{Kind: RefTag, Name: strings.TrimPrefix(refname, "refs/tags/"), SHA: sha, CommittedAt: committedAt})
		}
	}
	return refs
}
