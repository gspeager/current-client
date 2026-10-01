package git

import (
	"context"
	"strings"
)

type Tag struct {
	Name      string
	SHA       string
	Annotated bool
	Date      string
}

func ListTags(ctx context.Context, repoPath string) ([]Tag, error) {
	result, err := runResult(ctx, repoPath, "for-each-ref",
		"--format=%(refname:short)%09%(objectname)%09%(*objectname)%09%(creatordate:iso-strict)", "refs/tags")
	if err != nil {
		return nil, err
	}
	return parseTags(result.Stdout), nil
}

func parseTags(output string) []Tag {
	var tags []Tag
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "\t", 4)
		if len(fields) != 4 {
			continue
		}
		annotated := fields[2] != ""
		sha := fields[1]
		if annotated {
			sha = fields[2]
		}
		tags = append(tags, Tag{Name: fields[0], SHA: sha, Annotated: annotated, Date: fields[3]})
	}
	return tags
}

// CreateTag annotates the tag when message is set; an empty target means HEAD.
func CreateTag(ctx context.Context, repoPath, name, message, target string) error {
	args := []string{"tag"}
	if message != "" {
		args = append(args, "-a", "-m", message)
	}
	args = append(args, name)
	if target != "" {
		args = append(args, target)
	}
	_, err := runResult(ctx, repoPath, args...)
	return err
}

func DeleteTag(ctx context.Context, repoPath, name string) error {
	_, err := runResult(ctx, repoPath, "tag", "-d", name)
	return err
}

func PushTag(ctx context.Context, repoPath, remote, name string) error {
	_, err := runResult(ctx, repoPath, "push", remote, name)
	return err
}

func TagMessage(ctx context.Context, repoPath, name string) (subject, body string, err error) {
	result, err := runResult(ctx, repoPath, "for-each-ref", "--format=%(contents:subject)%1e%(contents:body)", "refs/tags/"+name)
	if err != nil {
		return "", "", err
	}
	parts := strings.SplitN(result.Stdout, "\x1e", 2)
	subject = parts[0]
	if len(parts) == 2 {
		body = strings.TrimRight(parts[1], "\n")
	}
	return subject, body, nil
}

// LatestTag is the newest tag reachable from ref, or "" when there is none.
func LatestTag(ctx context.Context, repoPath, ref string) (string, error) {
	sha, err := ResolveCommit(ctx, repoPath, ref)
	if err != nil {
		return "", err
	}
	result, err := runResult(ctx, repoPath, "describe", "--tags", "--abbrev=0", sha)
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(result.Stdout), nil
}

type DatedTag struct {
	Name string
	SHA  string // the tagged commit
	Date string // YYYY-MM-DD: the tagger date, or the commit date for a lightweight tag
}

// TagsInRange lists the tags reachable from to but not from from, newest
// first in history order (dates can tie or run backwards). An empty from means
// every tag reachable from to.
func TagsInRange(ctx context.Context, repoPath, from, to string) ([]DatedTag, error) {
	toSHA, err := ResolveCommit(ctx, repoPath, to)
	if err != nil {
		return nil, err
	}
	rev := toSHA
	if from != "" {
		fromSHA, err := ResolveCommit(ctx, repoPath, from)
		if err != nil {
			return nil, err
		}
		rev = fromSHA + ".." + toSHA
	}
	ordered, err := runResult(ctx, repoPath, "log", "--topo-order", "--decorate-refs=refs/tags/", "--format=%H %D", rev)
	if err != nil {
		return nil, err
	}
	dates, err := runResult(ctx, repoPath, "for-each-ref", "--format=%(refname:short)%09%(creatordate:short)", "refs/tags")
	if err != nil {
		return nil, err
	}
	dateOf := map[string]string{}
	for _, line := range strings.Split(dates.Stdout, "\n") {
		if name, date, ok := strings.Cut(line, "\t"); ok {
			dateOf[name] = date
		}
	}
	var tags []DatedTag
	for _, line := range strings.Split(ordered.Stdout, "\n") {
		sha, refs, _ := strings.Cut(line, " ")
		for _, ref := range strings.Split(refs, ", ") {
			if name, ok := strings.CutPrefix(ref, "tag: "); ok {
				tags = append(tags, DatedTag{Name: name, SHA: sha, Date: dateOf[name]})
			}
		}
	}
	return tags, nil
}
