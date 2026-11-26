// Package gitlab is the hostapi.Provider for gitlab.com and GitLab
// Self-Managed, over the REST v4 API.
package gitlab

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gspeager/current-client/core/hostapi"
)

// MinimumVersion is the oldest GitLab the provider is tested against;
// every endpoint and field it reads exists from there on.
const MinimumVersion = "16.0"

const (
	readScope  = "read_api"
	writeScope = "api"
)

type Provider struct {
	hostapi.NumberedKeys
	client *hostapi.Client
	host   hostapi.Host

	mu           sync.Mutex
	projectPaths map[int64]string
}

func New(client *hostapi.Client, host hostapi.Host) *Provider {
	return &Provider{client: client, host: host, projectPaths: map[int64]string{}}
}

var _ hostapi.Provider = (*Provider)(nil)

func (p *Provider) OpenChanges(ctx context.Context, project string) ([]hostapi.Change, bool, error) {
	mrs, truncated, err := getAll[mergeRequest](ctx, p, readScope, projectPath(project, "merge_requests"),
		url.Values{"state": {"opened"}, "per_page": {"100"}})
	if err != nil {
		return nil, false, err
	}
	changes, err := p.changes(ctx, mrs)
	return changes, truncated, err
}

// ClosedChanges asks for merged and closed separately, 25 each, since
// GitLab lists one state per request, and interleaves them newest first.
func (p *Provider) ClosedChanges(ctx context.Context, project string) ([]hostapi.Change, bool, error) {
	var changes []hostapi.Change
	truncated := false
	for _, state := range []string{"merged", "closed"} {
		mrs, more, err := getPage[mergeRequest](ctx, p, readScope, projectPath(project, "merge_requests"),
			url.Values{"state": {state}, "order_by": {"updated_at"}, "sort": {"desc"}, "per_page": {"25"}})
		if err != nil {
			return nil, false, err
		}
		page, err := p.changes(ctx, mrs)
		if err != nil {
			return nil, false, err
		}
		changes = append(changes, page...)
		truncated = truncated || more
	}
	sort.SliceStable(changes, func(i, j int) bool { return changes[i].UpdatedAt.After(changes[j].UpdatedAt) })
	return changes, truncated, nil
}

func (p *Provider) ChangeFiles(ctx context.Context, project string, number int) ([]hostapi.ChangeFile, error) {
	diffs, err := mergeRequestItems[mergeRequestDiff](ctx, p, project, number, "diffs")
	if err != nil {
		return nil, err
	}
	var files []hostapi.ChangeFile
	for _, d := range diffs {
		files = append(files, toChangeFile(d))
	}
	return files, nil
}

func (p *Provider) ChangeCommits(ctx context.Context, project string, number int) ([]hostapi.ChangeCommit, error) {
	page, err := mergeRequestItems[mergeRequestCommit](ctx, p, project, number, "commits")
	if err != nil {
		return nil, err
	}
	var commits []hostapi.ChangeCommit
	for _, c := range page {
		commits = append(commits, hostapi.ChangeCommit{SHA: c.ID, Subject: c.Title, Author: c.AuthorName, Date: c.CreatedAt})
	}
	return commits, nil
}

// mergeRequestItems reads every page of one of a merge request's lists.
func mergeRequestItems[T any](ctx context.Context, p *Provider, project string, number int, list string) ([]T, error) {
	items, _, err := getAll[T](ctx, p, readScope, projectPath(project, "merge_requests", strconv.Itoa(number), list), url.Values{"per_page": {"100"}})
	return items, err
}

func (p *Provider) Change(ctx context.Context, project string, number int) (hostapi.Change, error) {
	var mr mergeRequest
	if err := p.get(ctx, readScope, projectPath(project, "merge_requests", strconv.Itoa(number)), nil, &mr); err != nil {
		return hostapi.Change{}, err
	}
	return p.change(ctx, mr)
}

func (p *Provider) Threads(ctx context.Context, project string, number int, headSHA string) ([]hostapi.Thread, error) {
	discussions, err := mergeRequestItems[discussion](ctx, p, project, number, "discussions")
	if err != nil {
		return nil, err
	}
	var threads []hostapi.Thread
	for _, d := range discussions {
		if thread, ok := toThread(d, headSHA); ok {
			threads = append(threads, thread)
		}
	}
	return threads, nil
}

// Reviews reads the reviewers' states, then the approvals, since anyone may
// approve and an approval doesn't always reach the approver's reviewer state.
func (p *Provider) Reviews(ctx context.Context, project string, change hostapi.Change) ([]hostapi.Review, error) {
	number := strconv.Itoa(change.Number)
	var reviewers []reviewer
	if err := p.get(ctx, readScope, projectPath(project, "merge_requests", number, "reviewers"), nil, &reviewers); err != nil {
		return nil, err
	}
	var approved approvals
	if err := p.get(ctx, readScope, projectPath(project, "merge_requests", number, "approvals"), nil, &approved); err != nil {
		return nil, err
	}
	reviews := make([]hostapi.Review, 0, len(reviewers))
	for _, r := range reviewers {
		reviews = append(reviews, hostapi.Review{Reviewer: toUser(r.User), State: reviewState(r.State)})
	}
	for _, a := range approved.ApprovedBy {
		reviews = hostapi.SetReview(reviews, hostapi.Review{Reviewer: toUser(a.User), State: hostapi.ReviewApproved})
	}
	return reviews, nil
}

// reviewState reads a withdrawn approval as having reviewed.
func reviewState(state string) hostapi.ReviewState {
	switch state {
	case "approved":
		return hostapi.ReviewApproved
	case "requested_changes":
		return hostapi.ReviewChangesRequested
	case "reviewed", "unapproved":
		return hostapi.ReviewCommented
	}
	return hostapi.ReviewRequested
}

var (
	_ hostapi.IssueTracker = (*Provider)(nil)
	_ hostapi.IssueLister  = (*Provider)(nil)
)

func (p *Provider) CreateChange(ctx context.Context, project string, change hostapi.NewChange) (hostapi.Change, error) {
	title := change.Title
	if change.Draft && !strings.HasPrefix(strings.ToLower(title), "draft:") {
		title = "Draft: " + title
	}
	body := map[string]string{
		"source_branch": change.SourceBranch,
		"target_branch": change.TargetBranch,
		"title":         title,
		"description":   change.Description,
	}
	var mr mergeRequest
	if err := p.post(ctx, writeScope, projectPath(project, "merge_requests"), body, &mr); err != nil {
		return hostapi.Change{}, err
	}
	return p.change(ctx, mr)
}

// BranchRules lists every protected branch. The endpoint needs the
// Maintainer role; a 403 means no rules to warn about. A protected branch
// can't be deleted with a push on GitLab.
func (p *Provider) BranchRules(ctx context.Context, project string, _ []string) ([]hostapi.BranchRule, error) {
	branches, _, err := hostapi.GetAll[protectedBranch](ctx, p.client, p.host, projectPath(project, "protected_branches"), url.Values{"per_page": {"100"}})
	if hostapi.StatusOf(err) == http.StatusForbidden {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rules []hostapi.BranchRule
	for _, b := range branches {
		rules = append(rules, branchRule(b))
	}
	return rules, nil
}

func branchRule(b protectedBranch) hostapi.BranchRule {
	rule := hostapi.BranchRule{Pattern: b.Name, NoPush: true, NoForcePush: !b.AllowForcePush, NoDelete: true}
	for _, level := range b.PushAccessLevels {
		if level.AccessLevel > 0 {
			rule.NoPush, rule.RestrictedPush = false, true
		}
	}
	return rule
}

func (p *Provider) CurrentUser(ctx context.Context) (hostapi.User, error) {
	var u user
	if err := p.get(ctx, readScope, "user", nil, &u); err != nil {
		return hostapi.User{}, err
	}
	return toUser(u), nil
}

func toUser(u user) hostapi.User {
	return hostapi.User{Name: u.Name, Username: u.Username}
}

func (p *Provider) changes(ctx context.Context, mrs []mergeRequest) ([]hostapi.Change, error) {
	var changes []hostapi.Change
	for _, mr := range mrs {
		change, err := p.change(ctx, mr)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	return changes, nil
}

func (p *Provider) change(ctx context.Context, mr mergeRequest) (hostapi.Change, error) {
	change := hostapi.Change{
		Number:         mr.IID,
		Title:          mr.Title,
		Description:    mr.Description,
		State:          changeState(mr.State),
		Draft:          mr.Draft,
		SourceBranch:   mr.SourceBranch,
		TargetBranch:   mr.TargetBranch,
		Author:         toUser(mr.Author),
		Labels:         mr.Labels,
		HeadSHA:        mr.SHA,
		MergeCommitSHA: landedAs(mr),
		HasConflicts:   mr.HasConflicts,
		ConflictsKnown: true,
		UpdatedAt:      mr.UpdatedAt,
		WebURL:         mr.WebURL,
	}
	for _, r := range mr.Reviewers {
		change.Reviews = append(change.Reviews, hostapi.Review{Reviewer: toUser(r), State: hostapi.ReviewRequested})
	}
	if mr.HeadPipeline != nil {
		change.CI = ciStatus(mr.HeadPipeline.Status)
	}
	if mr.SourceProjectID != 0 && mr.SourceProjectID != mr.TargetProjectID {
		path, err := p.projectPath(ctx, mr.SourceProjectID)
		if err != nil {
			return hostapi.Change{}, err
		}
		change.SourceProject = path
	}
	return change, nil
}

// projectPath names a fork by its path; the merge request carries only its
// id. One lookup per fork, remembered for the provider's lifetime.
func (p *Provider) projectPath(ctx context.Context, id int64) (string, error) {
	p.mu.Lock()
	path, ok := p.projectPaths[id]
	p.mu.Unlock()
	if ok {
		return path, nil
	}
	var project struct {
		Path string `json:"path_with_namespace"`
	}
	if err := p.get(ctx, readScope, "projects/"+strconv.FormatInt(id, 10), nil, &project); err != nil {
		return "", err
	}
	p.mu.Lock()
	p.projectPaths[id] = project.Path
	p.mu.Unlock()
	return project.Path, nil
}

// toThread drops discussions made only of system notes ("added 1 commit").
func toThread(d discussion, headSHA string) (hostapi.Thread, bool) {
	thread := hostapi.Thread{ID: d.ID}
	for i, n := range d.Notes {
		if n.System {
			continue
		}
		if i == 0 || len(thread.Notes) == 0 {
			thread.Resolved = n.Resolved
			if n.Position != nil {
				thread.Path = n.Position.NewPath
				if thread.Path == "" {
					thread.Path = n.Position.OldPath
				}
				thread.Line, thread.OldLine = n.Position.NewLine, n.Position.OldLine
				thread.Outdated = headSHA != "" && n.Position.HeadSHA != headSHA
			}
		}
		thread.Notes = append(thread.Notes, hostapi.Note{Author: toUser(n.Author), Body: n.Body, CreatedAt: n.CreatedAt})
	}
	return thread, len(thread.Notes) > 0
}

// toChangeFile counts the diff's lines itself; the diffs endpoint sends
// no totals. Only lines after the first hunk header count, so a file
// header or a content line starting with --- is told apart by position.
func toChangeFile(d mergeRequestDiff) hostapi.ChangeFile {
	file := hostapi.ChangeFile{Path: d.NewPath, Status: "modified"}
	switch {
	case d.NewFile:
		file.Status = "added"
	case d.DeletedFile:
		file.Status = "deleted"
	case d.RenamedFile:
		file.Status, file.OldPath = "renamed", d.OldPath
	}
	inHunk := false
	for _, line := range strings.Split(d.Diff, "\n") {
		switch {
		case strings.HasPrefix(line, "@@"):
			inHunk = true
		case !inHunk:
		case strings.HasPrefix(line, "+"):
			file.Additions++
		case strings.HasPrefix(line, "-"):
			file.Deletions++
		}
	}
	return file
}

func changeState(state string) hostapi.ChangeState {
	switch state {
	case "opened":
		return hostapi.ChangeOpen
	case "merged":
		return hostapi.ChangeMerged
	}
	return hostapi.ChangeClosed
}

func projectPath(project string, parts ...string) string {
	return strings.Join(append([]string{"projects", url.PathEscape(project)}, parts...), "/")
}

// landedAs is the commit a merged merge request became on its target: the
// squash commit, else the merge commit, else (a fast-forward) its head.
func landedAs(mr mergeRequest) string {
	if mr.State != "merged" {
		return ""
	}
	for _, sha := range []string{mr.SquashCommitSHA, mr.MergeCommitSHA, mr.SHA} {
		if sha != "" {
			return sha
		}
	}
	return ""
}
