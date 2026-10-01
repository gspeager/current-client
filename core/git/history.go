package git

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type HistoryEntry struct {
	SHA         string
	ParentSHAs  []string
	AuthorName  string
	AuthorEmail string
	Date        time.Time
	Subject     string
	Body        string
	Signature   SignatureStatus
	Trailers    []Trailer
}

// SignatureStatus collapses git's %G? codes: anything short of a good,
// trusted signature counts as unverified.
type SignatureStatus string

const (
	SignatureNone       SignatureStatus = "none"
	SignatureVerified   SignatureStatus = "verified"
	SignatureUnverified SignatureStatus = "unverified"
)

func parseSignatureStatus(code string) SignatureStatus {
	switch code {
	case "N", "":
		return SignatureNone
	case "G":
		return SignatureVerified
	default:
		return SignatureUnverified
	}
}

// %aN/%aE apply .mailmap, matching shortlog and blame.
const historyFormat = "%H\x1f%P\x1f%aN\x1f%aE\x1f%at\x1f%s\x1f%b\x1f%G?"

// HistoryFilter.Ref is empty for HEAD, "--all" for every ref, or a branch name.
// Type is empty for every commit, a Conventional Commits type, or
// BreakingType for breaking changes of any type.
type HistoryFilter struct {
	Ref    string
	Since  string
	Until  string
	Author string
	Type   string
}

const BreakingType = "!"

func History(ctx context.Context, repoPath string, limit, skip int, filter HistoryFilter) ([]HistoryEntry, error) {
	if filter.Type != "" {
		return typedHistory(ctx, repoPath, limit, skip, filter)
	}
	args := append([]string{"log", "-z", "--pretty=format:" + historyFormat}, pageArgs(limit, skip)...)
	result, err := runResult(ctx, repoPath, append(args, filterArgs(filter)...)...)
	if err != nil {
		if hasNoCommits(ctx, repoPath) {
			return nil, nil
		}
		return nil, err
	}
	return parseHistory(result.Stdout)
}

// SearchHistory is limited to HEAD's ancestry so every match has a
// well-defined CommitPosition.
func SearchHistory(ctx context.Context, repoPath, query string, limit int) ([]HistoryEntry, error) {
	args := append([]string{"log", "-z", "--pretty=format:" + historyFormat, "-i", "--grep=" + query}, pageArgs(limit, 0)...)
	result, err := runResult(ctx, repoPath, args...)
	if err != nil {
		if hasNoCommits(ctx, repoPath) {
			return nil, nil
		}
		return nil, err
	}
	return parseHistory(result.Stdout)
}

// CommitPosition returns how many commits on HEAD's ancestry are newer than sha.
func CommitPosition(ctx context.Context, repoPath, sha string) (int, error) {
	return CountCommits(ctx, repoPath, sha+"..HEAD")
}

// CountCommits takes rev-list arguments, such as "a..b" or "b", "^a", "^c".
func CountCommits(ctx context.Context, repoPath string, revs ...string) (int, error) {
	result, err := runResult(ctx, repoPath, append([]string{"rev-list", "--count"}, revs...)...)
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(result.Stdout))
	if err != nil {
		return 0, fmt.Errorf("malformed rev-list count %q: %w", result.Stdout, err)
	}
	return n, nil
}

func FileHistory(ctx context.Context, repoPath, path string, limit, skip int) ([]HistoryEntry, error) {
	args := append([]string{"log", "-z", "--follow", "--pretty=format:" + historyFormat}, pageArgs(limit, skip)...)
	args = append(args, "--", path)
	result, err := runResult(ctx, repoPath, args...)
	if err != nil {
		if hasNoCommits(ctx, repoPath) {
			return nil, nil
		}
		return nil, err
	}
	return parseHistory(result.Stdout)
}

func pageArgs(limit, skip int) []string {
	var args []string
	if limit > 0 {
		args = append(args, "-n", strconv.Itoa(limit))
	}
	if skip > 0 {
		args = append(args, "--skip", strconv.Itoa(skip))
	}
	return args
}

func parseHistory(output string) ([]HistoryEntry, error) {
	var entries []HistoryEntry
	for _, record := range strings.Split(output, "\x00") {
		if record == "" {
			continue
		}
		fields := strings.Split(record, "\x1f")
		if len(fields) != 8 {
			return nil, fmt.Errorf("malformed commit entry: got %d fields, want 8", len(fields))
		}
		var parents []string
		if fields[1] != "" {
			parents = strings.Split(fields[1], " ")
		}
		ts, err := strconv.ParseInt(fields[4], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("malformed commit timestamp %q: %w", fields[4], err)
		}
		entries = append(entries, HistoryEntry{
			SHA:         fields[0],
			ParentSHAs:  parents,
			AuthorName:  fields[2],
			AuthorEmail: fields[3],
			Date:        time.Unix(ts, 0),
			Subject:     fields[5],
			Body:        strings.TrimRight(fields[6], "\n"),
			Signature:   parseSignatureStatus(fields[7]),
			Trailers:    ParseTrailers(fields[6]),
		})
	}
	return entries, nil
}

func filterArgs(filter HistoryFilter) []string {
	var args []string
	if filter.Since != "" {
		args = append(args, "--since="+filter.Since)
	}
	if filter.Until != "" {
		args = append(args, "--until="+filter.Until)
	}
	if filter.Author != "" {
		// --use-mailmap so the canonical name also finds an alias's commits.
		args = append(args, "--use-mailmap", "--author="+filter.Author)
	}
	if filter.Ref == "--all" {
		args = append(args, "--all")
	} else if filter.Ref != "" {
		args = append(args, filter.Ref)
	}
	return args
}

// typedHistory pages in Go rather than with git's -n/--skip: --grep matches
// any message line, so a body line like "fix: …" has to be re-checked by the
// parser, and dropping it after git paged would leave pages short.
func typedHistory(ctx context.Context, repoPath string, limit, skip int, filter HistoryFilter) ([]HistoryEntry, error) {
	args := []string{"log", "-z", "-E", "-i", "--format=%H\x1f%s\x1f%b", "--grep=" + typeGrep(filter.Type)}
	result, err := runResult(ctx, repoPath, append(args, filterArgs(filter)...)...)
	if err != nil {
		if hasNoCommits(ctx, repoPath) {
			return nil, nil
		}
		return nil, err
	}
	var shas []string
	for _, record := range strings.Split(result.Stdout, "\x00") {
		fields := strings.SplitN(record, "\x1f", 3)
		if len(fields) != 3 {
			continue
		}
		c := ParseConventional(fields[1], fields[2])
		if (filter.Type == BreakingType && c.Breaking) || (filter.Type != BreakingType && c.Type == strings.ToLower(filter.Type)) {
			shas = append(shas, fields[0])
		}
	}
	shas = shas[min(skip, len(shas)):]
	if limit > 0 {
		shas = shas[:min(limit, len(shas))]
	}
	if len(shas) == 0 {
		return nil, nil
	}
	page, err := runResult(ctx, repoPath, append([]string{"log", "-z", "--no-walk=unsorted", "--pretty=format:" + historyFormat}, shas...)...)
	if err != nil {
		return nil, err
	}
	return parseHistory(page.Stdout)
}

func typeGrep(commitType string) string {
	if commitType == BreakingType {
		return `^[a-z]+(\([^()]+\))?!: |^BREAKING[ -]CHANGE: `
	}
	return "^" + regexp.QuoteMeta(commitType) + `(\([^()]+\))?!?: `
}

// RangeHistory lists the commits reachable from to but not from from, merges
// included, newest first. An empty from means all of to's history; since,
// when set, is passed to --since.
func RangeHistory(ctx context.Context, repoPath, from, to, since string) ([]HistoryEntry, error) {
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
	args := []string{"log", "-z", "--pretty=format:" + historyFormat}
	if since != "" {
		args = append(args, "--since="+since)
	}
	result, err := runResult(ctx, repoPath, append(args, rev)...)
	if err != nil {
		return nil, err
	}
	return parseHistory(result.Stdout)
}

// ResolveCommit returns ref's commit SHA. The ^{commit} suffix also keeps a ref
// that starts with "-" from reading as an option.
func ResolveCommit(ctx context.Context, repoPath, ref string) (string, error) {
	result, err := runResult(ctx, repoPath, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("%s is not a commit", ref)
	}
	return strings.TrimSpace(result.Stdout), nil
}
