// Package gitlab is the hostapi.Provider for gitlab.com and GitLab
// Self-Managed, over the REST v4 API.
package gitlab

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

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
	client *hostapi.Client
	host   hostapi.Host

	mu           sync.Mutex
	projectPaths map[int64]string
}

func New(client *hostapi.Client, host hostapi.Host) *Provider {
	return &Provider{client: client, host: host, projectPaths: map[int64]string{}}
}

var _ hostapi.Provider = (*Provider)(nil)

type mergeRequest struct {
	IID             int       `json:"iid"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	State           string    `json:"state"`
	Draft           bool      `json:"draft"`
	SourceBranch    string    `json:"source_branch"`
	TargetBranch    string    `json:"target_branch"`
	SourceProjectID int64     `json:"source_project_id"`
	TargetProjectID int64     `json:"target_project_id"`
	Author          user      `json:"author"`
	Labels          []string  `json:"labels"`
	SHA             string    `json:"sha"`
	HasConflicts    bool      `json:"has_conflicts"`
	UpdatedAt       time.Time `json:"updated_at"`
	WebURL          string    `json:"web_url"`
	HeadPipeline    *pipeline `json:"head_pipeline"`
	Reviewers       []user    `json:"reviewers"`
}

type reviewer struct {
	User  user   `json:"user"`
	State string `json:"state"`
}

type approvals struct {
	ApprovedBy []struct {
		User user `json:"user"`
	} `json:"approved_by"`
}

// user reads only what the model keeps; avatar_url is never decoded.
type user struct {
	Name     string `json:"name"`
	Username string `json:"username"`
}

type pipeline struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
	WebURL string `json:"web_url"`
}

type job struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Stage    string  `json:"stage"`
	Status   string  `json:"status"`
	Duration float64 `json:"duration"`
	WebURL   string  `json:"web_url"`
}

type discussion struct {
	ID    string `json:"id"`
	Notes []note `json:"notes"`
}

type note struct {
	Body      string    `json:"body"`
	Author    user      `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	System    bool      `json:"system"`
	Resolved  bool      `json:"resolved"`
	Position  *position `json:"position"`
}

type position struct {
	HeadSHA string `json:"head_sha"`
	NewPath string `json:"new_path"`
	OldPath string `json:"old_path"`
	NewLine int    `json:"new_line"`
	OldLine int    `json:"old_line"`
}

type mergeRequestDiff struct {
	OldPath     string `json:"old_path"`
	NewPath     string `json:"new_path"`
	NewFile     bool   `json:"new_file"`
	RenamedFile bool   `json:"renamed_file"`
	DeletedFile bool   `json:"deleted_file"`
	Diff        string `json:"diff"`
}

type mergeRequestCommit struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	AuthorName string    `json:"author_name"`
	CreatedAt  time.Time `json:"created_at"`
}

type issue struct {
	IID         int      `json:"iid"`
	Title       string   `json:"title"`
	State       string   `json:"state"`
	WebURL      string   `json:"web_url"`
	Description string   `json:"description"`
	Labels      []string `json:"labels"`
	Milestone   *struct {
		Title string `json:"title"`
	} `json:"milestone"`
	Author    user      `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type issueNote struct {
	Author    user      `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	System    bool      `json:"system"`
}

func (p *Provider) OpenChanges(ctx context.Context, project string) ([]hostapi.Change, bool, error) {
	var changes []hostapi.Change
	truncated, err := p.client.GetPages(ctx, p.host, projectPath(project, "merge_requests"),
		url.Values{"state": {"opened"}, "per_page": {"100"}}, func(resp *hostapi.Response) error {
			var page []mergeRequest
			if err := hostapi.DecodeJSON(resp, &page); err != nil {
				return err
			}
			for _, mr := range page {
				change, err := p.change(ctx, mr)
				if err != nil {
					return err
				}
				changes = append(changes, change)
			}
			return nil
		})
	return changes, truncated, hostapi.WithScope(err, readScope)
}

// ClosedChanges asks for merged and closed separately, 25 each, since
// GitLab lists one state per request, and interleaves them newest first.
func (p *Provider) ClosedChanges(ctx context.Context, project string) ([]hostapi.Change, bool, error) {
	var changes []hostapi.Change
	truncated := false
	for _, state := range []string{"merged", "closed"} {
		resp, err := p.client.Get(ctx, p.host, projectPath(project, "merge_requests"),
			url.Values{"state": {state}, "order_by": {"updated_at"}, "sort": {"desc"}, "per_page": {"25"}})
		if err != nil {
			return nil, false, hostapi.WithScope(err, readScope)
		}
		var page []mergeRequest
		if err := hostapi.DecodeJSON(resp, &page); err != nil {
			return nil, false, err
		}
		for _, mr := range page {
			change, err := p.change(ctx, mr)
			if err != nil {
				return nil, false, err
			}
			changes = append(changes, change)
		}
		truncated = truncated || hostapi.HasNextPage(resp)
	}
	sort.SliceStable(changes, func(i, j int) bool { return changes[i].UpdatedAt.After(changes[j].UpdatedAt) })
	return changes, truncated, nil
}

func (p *Provider) ChangeFiles(ctx context.Context, project string, number int) ([]hostapi.ChangeFile, error) {
	var files []hostapi.ChangeFile
	_, err := p.client.GetPages(ctx, p.host, projectPath(project, "merge_requests", strconv.Itoa(number), "diffs"),
		url.Values{"per_page": {"100"}}, func(resp *hostapi.Response) error {
			var page []mergeRequestDiff
			if err := hostapi.DecodeJSON(resp, &page); err != nil {
				return err
			}
			for _, d := range page {
				files = append(files, toChangeFile(d))
			}
			return nil
		})
	return files, hostapi.WithScope(err, readScope)
}

func (p *Provider) ChangeCommits(ctx context.Context, project string, number int) ([]hostapi.ChangeCommit, error) {
	var commits []hostapi.ChangeCommit
	_, err := p.client.GetPages(ctx, p.host, projectPath(project, "merge_requests", strconv.Itoa(number), "commits"),
		url.Values{"per_page": {"100"}}, func(resp *hostapi.Response) error {
			var page []mergeRequestCommit
			if err := hostapi.DecodeJSON(resp, &page); err != nil {
				return err
			}
			for _, c := range page {
				commits = append(commits, hostapi.ChangeCommit{SHA: c.ID, Subject: c.Title, Author: c.AuthorName, Date: c.CreatedAt})
			}
			return nil
		})
	return commits, hostapi.WithScope(err, readScope)
}

func (p *Provider) Change(ctx context.Context, project string, number int) (hostapi.Change, error) {
	var mr mergeRequest
	if err := p.client.GetJSON(ctx, p.host, projectPath(project, "merge_requests", strconv.Itoa(number)), nil, &mr); err != nil {
		return hostapi.Change{}, hostapi.WithScope(err, readScope)
	}
	return p.change(ctx, mr)
}

func (p *Provider) Threads(ctx context.Context, project string, number int, headSHA string) ([]hostapi.Thread, error) {
	var threads []hostapi.Thread
	_, err := p.client.GetPages(ctx, p.host, projectPath(project, "merge_requests", strconv.Itoa(number), "discussions"),
		url.Values{"per_page": {"100"}}, func(resp *hostapi.Response) error {
			var page []discussion
			if err := hostapi.DecodeJSON(resp, &page); err != nil {
				return err
			}
			for _, d := range page {
				if thread, ok := toThread(d, headSHA); ok {
					threads = append(threads, thread)
				}
			}
			return nil
		})
	return threads, hostapi.WithScope(err, readScope)
}

// Reviews reads the reviewers' states, then the approvals, since anyone may
// approve and an approval doesn't always reach the approver's reviewer state.
func (p *Provider) Reviews(ctx context.Context, project string, change hostapi.Change) ([]hostapi.Review, error) {
	number := strconv.Itoa(change.Number)
	var reviewers []reviewer
	if err := p.client.GetJSON(ctx, p.host, projectPath(project, "merge_requests", number, "reviewers"), nil, &reviewers); err != nil {
		return nil, hostapi.WithScope(err, readScope)
	}
	var approved approvals
	if err := p.client.GetJSON(ctx, p.host, projectPath(project, "merge_requests", number, "approvals"), nil, &approved); err != nil {
		return nil, hostapi.WithScope(err, readScope)
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

func (p *Provider) LatestPipeline(ctx context.Context, project, sha string) (hostapi.Pipeline, bool, error) {
	var pipelines []pipeline
	err := p.client.GetJSON(ctx, p.host, projectPath(project, "pipelines"), url.Values{"sha": {sha}, "per_page": {"1"}}, &pipelines)
	if err != nil || len(pipelines) == 0 {
		return hostapi.Pipeline{}, false, hostapi.WithScope(err, readScope)
	}
	return toPipeline(pipelines[0]), true, nil
}

func (p *Provider) Jobs(ctx context.Context, project string, pipelineID int64) ([]hostapi.Job, error) {
	var jobs []job
	err := p.client.GetJSON(ctx, p.host, projectPath(project, "pipelines", strconv.FormatInt(pipelineID, 10), "jobs"), url.Values{"per_page": {"100"}}, &jobs)
	if err != nil {
		return nil, hostapi.WithScope(err, readScope)
	}
	out := make([]hostapi.Job, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, hostapi.Job{
			ID:       j.ID,
			Name:     j.Name,
			Stage:    j.Stage,
			Status:   ciStatus(j.Status),
			Duration: time.Duration(j.Duration * float64(time.Second)),
			WebURL:   j.WebURL,
		})
	}
	return out, nil
}

func (p *Provider) JobLog(ctx context.Context, project string, jobID int64, maxBytes int) (string, bool, error) {
	resp, err := p.client.Get(ctx, p.host, projectPath(project, "jobs", strconv.FormatInt(jobID, 10), "trace"), nil)
	if err != nil {
		return "", false, hostapi.WithScope(err, readScope)
	}
	log, truncated := hostapi.TailLog(resp.Body, maxBytes)
	return log, truncated, nil
}

func (p *Provider) Issue(ctx context.Context, project string, number int) (hostapi.Issue, error) {
	var is issue
	if err := p.client.GetJSON(ctx, p.host, projectPath(project, "issues", strconv.Itoa(number)), nil, &is); err != nil {
		return hostapi.Issue{}, hostapi.WithScope(err, readScope)
	}
	return toIssue(is), nil
}

func (p *Provider) AssignedIssues(ctx context.Context, project, username string) ([]hostapi.Issue, error) {
	var page []issue
	query := url.Values{"assignee_username": {username}, "state": {"opened"}, "order_by": {"updated_at"}, "per_page": {"50"}}
	if err := p.client.GetJSON(ctx, p.host, projectPath(project, "issues"), query, &page); err != nil {
		return nil, hostapi.WithScope(err, readScope)
	}
	issues := make([]hostapi.Issue, 0, len(page))
	for _, is := range page {
		issues = append(issues, toIssue(is))
	}
	return issues, nil
}

var _ hostapi.IssueTracker = (*Provider)(nil)

func (p *Provider) IssueByKey(ctx context.Context, project, key string) (hostapi.Issue, error) {
	number, err := hostapi.IssueNumber(key)
	if err != nil {
		return hostapi.Issue{}, err
	}
	return p.Issue(ctx, project, number)
}

func (p *Provider) KeyPattern() *regexp.Regexp {
	return hostapi.NumberedKeys
}

func (p *Provider) KeyInBranch(branch string) string {
	return hostapi.NumberedBranchKey(branch)
}

func (p *Provider) Reference(key string) string {
	return hostapi.NumberedReference(key)
}

func (p *Provider) IssueDetails(ctx context.Context, project, key string) (hostapi.IssueDetails, error) {
	number, err := hostapi.IssueNumber(key)
	if err != nil {
		return hostapi.IssueDetails{}, err
	}
	var is issue
	if err := p.client.GetJSON(ctx, p.host, projectPath(project, "issues", strconv.Itoa(number)), nil, &is); err != nil {
		return hostapi.IssueDetails{}, hostapi.WithScope(err, readScope)
	}
	return toIssueDetails(is), nil
}

// IssueComments leaves out system notes, GitLab's own record of label and
// state changes.
func (p *Provider) IssueComments(ctx context.Context, project, key string) ([]hostapi.Note, error) {
	number, err := hostapi.IssueNumber(key)
	if err != nil {
		return nil, err
	}
	var page []issueNote
	query := url.Values{"sort": {"asc"}, "order_by": {"created_at"}, "per_page": {"100"}}
	if err := p.client.GetJSON(ctx, p.host, projectPath(project, "issues", strconv.Itoa(number), "notes"), query, &page); err != nil {
		return nil, hostapi.WithScope(err, readScope)
	}
	notes := make([]hostapi.Note, 0, len(page))
	for _, n := range page {
		if !n.System {
			notes = append(notes, hostapi.Note{Author: toUser(n.Author), Body: n.Body, CreatedAt: n.CreatedAt})
		}
	}
	return notes, nil
}

func toIssue(is issue) hostapi.Issue {
	state := "open"
	if is.State == "closed" {
		state = "closed"
	}
	return hostapi.Issue{Key: strconv.Itoa(is.IID), Number: is.IID, Title: is.Title, State: state, WebURL: is.WebURL}
}

func toIssueDetails(is issue) hostapi.IssueDetails {
	labels := is.Labels
	if labels == nil {
		labels = []string{}
	}
	out := hostapi.IssueDetails{Issue: toIssue(is), Description: is.Description, Labels: labels, Author: toUser(is.Author), CreatedAt: is.CreatedAt, UpdatedAt: is.UpdatedAt}
	if is.Milestone != nil {
		out.Milestone = is.Milestone.Title
	}
	return out
}

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
	if err := p.client.PostJSON(ctx, p.host, projectPath(project, "merge_requests"), body, &mr); err != nil {
		return hostapi.Change{}, hostapi.WithScope(err, writeScope)
	}
	return p.change(ctx, mr)
}

type protectedBranch struct {
	Name             string `json:"name"`
	AllowForcePush   bool   `json:"allow_force_push"`
	PushAccessLevels []struct {
		AccessLevel int `json:"access_level"`
	} `json:"push_access_levels"`
}

// BranchRules lists every protected branch. The endpoint needs the
// Maintainer role; a 403 means no rules to warn about. A protected branch
// can't be deleted with a push on GitLab.
func (p *Provider) BranchRules(ctx context.Context, project string, _ []string) ([]hostapi.BranchRule, error) {
	var rules []hostapi.BranchRule
	_, err := p.client.GetPages(ctx, p.host, projectPath(project, "protected_branches"), url.Values{"per_page": {"100"}},
		func(resp *hostapi.Response) error {
			var page []protectedBranch
			if err := hostapi.DecodeJSON(resp, &page); err != nil {
				return err
			}
			for _, b := range page {
				rules = append(rules, branchRule(b))
			}
			return nil
		})
	var hostErr *hostapi.HostError
	if errors.As(err, &hostErr) && hostErr.Status == http.StatusForbidden {
		return nil, nil
	}
	return rules, hostapi.WithScope(err, readScope)
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
	if err := p.client.GetJSON(ctx, p.host, "user", nil, &u); err != nil {
		return hostapi.User{}, hostapi.WithScope(err, readScope)
	}
	return toUser(u), nil
}

func toUser(u user) hostapi.User {
	return hostapi.User{Name: u.Name, Username: u.Username}
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
	if err := p.client.GetJSON(ctx, p.host, "projects/"+strconv.FormatInt(id, 10), nil, &project); err != nil {
		return "", hostapi.WithScope(err, readScope)
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

func toPipeline(pl pipeline) hostapi.Pipeline {
	return hostapi.Pipeline{ID: pl.ID, Status: ciStatus(pl.Status), WebURL: pl.WebURL}
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

func ciStatus(status string) hostapi.CIStatus {
	switch status {
	case "success":
		return hostapi.CISuccess
	case "failed":
		return hostapi.CIFailed
	case "running":
		return hostapi.CIRunning
	case "canceled", "canceling":
		return hostapi.CICanceled
	case "skipped":
		return hostapi.CISkipped
	case "pending", "created", "waiting_for_resource", "preparing", "scheduled", "manual":
		return hostapi.CIPending
	}
	return hostapi.CIUnknown
}

// projectPath builds "projects/<group%2Fproject>/<parts…>".
func projectPath(project string, parts ...string) string {
	return strings.Join(append([]string{"projects", url.PathEscape(project)}, parts...), "/")
}
