package git

import (
	"context"
	"regexp"
	"strconv"
	"strings"
)

// ConventionalCommit.Type is lowercased, and empty when the message doesn't
// follow Conventional Commits.
type ConventionalCommit struct {
	Type         string
	Scope        string
	Breaking     bool
	BreakingNote string
	Description  string
}

const conventionalSampleSize = 100

var (
	conventionalSubject = regexp.MustCompile(`^([A-Za-z]+)(?:\(([^()]+)\))?(!)?: (\S.*)$`)
	breakingFooter      = regexp.MustCompile(`(?m)^BREAKING[ -]CHANGE: (.+)$`)
)

func ParseConventional(subject, body string) ConventionalCommit {
	m := conventionalSubject.FindStringSubmatch(strings.TrimSpace(subject))
	if m == nil {
		return ConventionalCommit{}
	}
	c := ConventionalCommit{
		Type:        strings.ToLower(m[1]),
		Scope:       strings.TrimSpace(m[2]),
		Breaking:    m[3] == "!",
		Description: strings.TrimSpace(m[4]),
	}
	if f := breakingFooter.FindStringSubmatch(body); f != nil {
		c.Breaking = true
		c.BreakingNote = strings.TrimSpace(f[1])
	}
	return c
}

// FollowsConventionalCommits reports whether more than half of HEAD's recent
// non-merge commits parse.
func FollowsConventionalCommits(ctx context.Context, repoPath string) (bool, error) {
	result, err := runResult(ctx, repoPath, "log", "--no-merges", "-z", "--format=%s", "-n", strconv.Itoa(conventionalSampleSize))
	if err != nil {
		if hasNoCommits(ctx, repoPath) {
			return false, nil
		}
		return false, err
	}
	return followsConventional(strings.Split(strings.TrimSuffix(result.Stdout, "\x00"), "\x00")), nil
}

func followsConventional(subjects []string) bool {
	parsed, total := 0, 0
	for _, s := range subjects {
		if s == "" {
			continue
		}
		total++
		if ParseConventional(s, "").Type != "" {
			parsed++
		}
	}
	return total > 0 && parsed*2 > total
}
