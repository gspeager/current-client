// Package github is the hostapi.Provider for github.com and GitHub
// Enterprise Server, over the REST API. The two differ only in address.
package github

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gspeager/current-client/core/hostapi"
)

// MinimumEnterpriseVersion is the oldest GitHub Enterprise Server the
// provider is tested against: repository rulesets, which the protected-branch
// warning reads, arrived in 3.11. Earlier versions get everything else.
const MinimumEnterpriseVersion = "3.11"

// A classic token needs the repo scope; a fine-grained one, read access to
// pull requests, checks, contents and issues, and write access to pull
// requests for creating one.
const scope = "repo"

// ruleBranchCap bounds the per-branch rules requests a refresh makes.
const ruleBranchCap = 25

type Provider struct {
	hostapi.NumberedKeys
	client *hostapi.Client
	host   hostapi.Host
}

func New(client *hostapi.Client, host hostapi.Host) *Provider {
	return &Provider{client: client, host: host}
}

var _ hostapi.Provider = (*Provider)(nil)

func (p *Provider) OpenChanges(ctx context.Context, project string) ([]hostapi.Change, bool, error) {
	prs, truncated, err := getAll[pullRequest](ctx, p, scope, repoPath(project, "pulls"),
		url.Values{"state": {"open"}, "per_page": {"100"}})
	if err != nil {
		return nil, false, err
	}
	return toChanges(prs), truncated, nil
}

// ClosedChanges is one page of 50, closed and merged alike, since GitHub
// lists both under one state.
func (p *Provider) ClosedChanges(ctx context.Context, project string) ([]hostapi.Change, bool, error) {
	prs, more, err := getPage[pullRequest](ctx, p, scope, repoPath(project, "pulls"),
		url.Values{"state": {"closed"}, "sort": {"updated"}, "direction": {"desc"}, "per_page": {"50"}})
	if err != nil {
		return nil, false, err
	}
	return toChanges(prs), more, nil
}

func toChanges(prs []pullRequest) []hostapi.Change {
	var changes []hostapi.Change
	for _, pr := range prs {
		changes = append(changes, toChange(pr))
	}
	return changes
}

func (p *Provider) ChangeFiles(ctx context.Context, project string, number int) ([]hostapi.ChangeFile, error) {
	page, err := pullItems[pullFile](ctx, p, project, number, "files")
	if err != nil {
		return nil, err
	}
	var files []hostapi.ChangeFile
	for _, f := range page {
		files = append(files, hostapi.ChangeFile{
			Path: f.Filename, OldPath: f.PreviousFilename, Status: fileStatus(f.Status),
			Additions: f.Additions, Deletions: f.Deletions,
		})
	}
	return files, nil
}

func (p *Provider) ChangeCommits(ctx context.Context, project string, number int) ([]hostapi.ChangeCommit, error) {
	page, err := pullItems[pullCommit](ctx, p, project, number, "commits")
	if err != nil {
		return nil, err
	}
	var commits []hostapi.ChangeCommit
	for _, c := range page {
		subject, _, _ := strings.Cut(c.Commit.Message, "\n")
		commits = append(commits, hostapi.ChangeCommit{SHA: c.SHA, Subject: subject, Author: c.Commit.Author.Name, Date: c.Commit.Author.Date})
	}
	return commits, nil
}

// pullItems reads every page of one of a pull request's lists.
func pullItems[T any](ctx context.Context, p *Provider, project string, number int, list string) ([]T, error) {
	items, _, err := getAll[T](ctx, p, scope, repoPath(project, "pulls", strconv.Itoa(number), list), url.Values{"per_page": {"100"}})
	return items, err
}

func (p *Provider) Change(ctx context.Context, project string, number int) (hostapi.Change, error) {
	var pr pullRequest
	if err := p.get(ctx, scope, repoPath(project, "pulls", strconv.Itoa(number)), nil, &pr); err != nil {
		return hostapi.Change{}, err
	}
	return toChange(pr), nil
}

// Threads groups review comments by the comment they reply to. The REST
// API doesn't say whether a thread is resolved, so none is.
func (p *Provider) Threads(ctx context.Context, project string, number int, headSHA string) ([]hostapi.Thread, error) {
	comments, err := pullItems[reviewComment](ctx, p, project, number, "comments")
	if err != nil {
		return nil, err
	}
	var threads []hostapi.Thread
	index := map[int64]int{}
	for _, c := range comments {
		note := hostapi.Note{Author: toUser(c.User), Body: c.Body, CreatedAt: c.CreatedAt}
		if i, ok := index[c.InReplyTo]; ok && c.InReplyTo != 0 {
			threads[i].Notes = append(threads[i].Notes, note)
			continue
		}
		index[c.ID] = len(threads)
		threads = append(threads, toThread(c, headSHA, note))
	}
	return threads, nil
}

// Reviews keeps each reviewer's latest verdict, where a later comment
// doesn't undo an approval or a request for changes. A reviewer asked again
// is shown as requested, as the host does.
func (p *Provider) Reviews(ctx context.Context, project string, change hostapi.Change) ([]hostapi.Review, error) {
	page, err := pullItems[review](ctx, p, project, change.Number, "reviews")
	if err != nil {
		return nil, err
	}
	var reviews []hostapi.Review
	for _, r := range page {
		state, ok := reviewState(r.State)
		if !ok || (state == hostapi.ReviewCommented && hasVerdict(reviews, r.User.Login)) {
			continue
		}
		reviews = hostapi.SetReview(reviews, hostapi.Review{Reviewer: toUser(r.User), State: state})
	}
	for _, requested := range change.Reviews {
		reviews = hostapi.SetReview(reviews, requested)
	}
	return reviews, nil
}

// reviewState reads a dismissed review as a comment; a pending one is a
// draft nobody else sees.
func reviewState(state string) (hostapi.ReviewState, bool) {
	switch state {
	case "APPROVED":
		return hostapi.ReviewApproved, true
	case "CHANGES_REQUESTED":
		return hostapi.ReviewChangesRequested, true
	case "COMMENTED", "DISMISSED":
		return hostapi.ReviewCommented, true
	}
	return "", false
}

func hasVerdict(reviews []hostapi.Review, login string) bool {
	for _, r := range reviews {
		if r.Reviewer.Username == login {
			return r.State == hostapi.ReviewApproved || r.State == hostapi.ReviewChangesRequested
		}
	}
	return false
}

var (
	_ hostapi.IssueTracker = (*Provider)(nil)
	_ hostapi.IssueLister  = (*Provider)(nil)
)

func (p *Provider) CreateChange(ctx context.Context, project string, change hostapi.NewChange) (hostapi.Change, error) {
	body := map[string]any{
		"head":  change.SourceBranch,
		"base":  change.TargetBranch,
		"title": change.Title,
		"body":  change.Description,
		"draft": change.Draft,
	}
	var pr pullRequest
	if err := p.post(ctx, scope, repoPath(project, "pulls"), body, &pr); err != nil {
		return hostapi.Change{}, err
	}
	return toChange(pr), nil
}

// BranchRules asks the rules that apply to each named branch, which needs no
// administration permission, unlike classic branch protection. A host
// without rulesets (Enterprise Server before 3.11) answers 404: no rules.
func (p *Provider) BranchRules(ctx context.Context, project string, branches []string) ([]hostapi.BranchRule, error) {
	var rules []hostapi.BranchRule
	if len(branches) > ruleBranchCap {
		branches = branches[:ruleBranchCap]
	}
	for _, branch := range branches {
		var applied []rule
		err := p.get(ctx, scope, repoPath(project, "rules", "branches", url.PathEscape(branch)), nil, &applied)
		if status := hostapi.StatusOf(err); status == http.StatusNotFound || status == http.StatusForbidden {
			return rules, nil
		}
		if err != nil {
			return nil, err
		}
		if r, ok := branchRule(branch, applied); ok {
			rules = append(rules, r)
		}
	}
	return rules, nil
}

func branchRule(branch string, applied []rule) (hostapi.BranchRule, bool) {
	r := hostapi.BranchRule{Pattern: branch}
	for _, a := range applied {
		switch a.Type {
		case "pull_request":
			r.NoPush = true
		case "update":
			r.RestrictedPush = true
		case "non_fast_forward":
			r.NoForcePush = true
		case "deletion":
			r.NoDelete = true
		}
	}
	return r, r.NoPush || r.RestrictedPush || r.NoForcePush || r.NoDelete
}

func (p *Provider) CurrentUser(ctx context.Context) (hostapi.User, error) {
	var u user
	if err := p.get(ctx, scope, "user", nil, &u); err != nil {
		return hostapi.User{}, err
	}
	return toUser(u), nil
}

func toChange(pr pullRequest) hostapi.Change {
	c := hostapi.Change{
		Number:         pr.Number,
		Title:          pr.Title,
		Description:    pr.Body,
		State:          changeState(pr),
		Draft:          pr.Draft,
		SourceBranch:   pr.Head.Ref,
		TargetBranch:   pr.Base.Ref,
		Author:         toUser(pr.User),
		HeadSHA:        pr.Head.SHA,
		MergeCommitSHA: mergedSHA(pr),
		HasConflicts:   pr.Mergeable != nil && !*pr.Mergeable,
		ConflictsKnown: pr.Mergeable != nil,
		UpdatedAt:      pr.UpdatedAt,
		WebURL:         pr.HTMLURL,
	}
	for _, l := range pr.Labels {
		c.Labels = append(c.Labels, l.Name)
	}
	for _, r := range pr.Reviewers {
		c.Reviews = append(c.Reviews, hostapi.Review{Reviewer: toUser(r), State: hostapi.ReviewRequested})
	}
	if pr.Head.Repo != nil && pr.Base.Repo != nil && pr.Head.Repo.FullName != pr.Base.Repo.FullName {
		c.SourceProject = pr.Head.Repo.FullName
	}
	return c
}

func fileStatus(status string) string {
	switch status {
	case "added", "copied":
		return "added"
	case "removed":
		return "deleted"
	case "renamed":
		return "renamed"
	}
	return "modified"
}

func changeState(pr pullRequest) hostapi.ChangeState {
	switch {
	case pr.State == "open":
		return hostapi.ChangeOpen
	case pr.MergedAt != nil:
		return hostapi.ChangeMerged
	}
	return hostapi.ChangeClosed
}

// toUser keeps the login as the name where the host sends none, as in a
// pull request's author.
func toUser(u user) hostapi.User {
	name := u.Name
	if name == "" {
		name = u.Login
	}
	return hostapi.User{Name: name, Username: u.Login}
}

// toThread marks a comment outdated when its position is gone: it was left on
// code that has changed since.
func toThread(c reviewComment, headSHA string, first hostapi.Note) hostapi.Thread {
	thread := hostapi.Thread{ID: strconv.FormatInt(c.ID, 10), Path: c.Path, Notes: []hostapi.Note{first}}
	line := 0
	if c.Line != nil {
		line = *c.Line
	} else if c.OriginalLine != nil {
		line = *c.OriginalLine
	}
	if c.Side == "LEFT" {
		thread.OldLine = line
	} else {
		thread.Line = line
	}
	thread.Outdated = c.Position == nil || (headSHA != "" && c.CommitID != headSHA)
	return thread
}

// GitHub has no pipeline id, so a commit's is its SHA's first hex digits:
// 13 of them, 52 bits, so the id survives being a JavaScript number in the
// frontend (a 53-bit mantissa) and comes back exact.
const pipelineIDChars = 13

// webRoot is the web pages' address: github.com for its own API host, else
// the Enterprise Server itself, which serves the API under /api/v3.
func (p *Provider) webRoot() string {
	if p.host.Name == "github.com" || p.host.Name == "api.github.com" {
		return "https://github.com/"
	}
	return "https://" + p.host.Name + "/"
}

func repoPath(project string, parts ...string) string {
	return strings.Join(append([]string{"repos", project}, parts...), "/")
}

// mergedSHA is only meaningful once merged: while open, GitHub's
// merge_commit_sha is a test merge.
func mergedSHA(pr pullRequest) string {
	if pr.MergedAt == nil {
		return ""
	}
	return pr.MergeSHA
}
