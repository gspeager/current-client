// Package github is the hostapi.Provider for github.com and GitHub
// Enterprise Server, over the REST API. The two differ only in address.
package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

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
	client *hostapi.Client
	host   hostapi.Host
}

func New(client *hostapi.Client, host hostapi.Host) *Provider {
	return &Provider{client: client, host: host}
}

var _ hostapi.Provider = (*Provider)(nil)

// user reads only what the model keeps; avatar_url is never decoded.
type user struct {
	Login string `json:"login"`
	Name  string `json:"name"`
}

type repoRef struct {
	Ref  string `json:"ref"`
	SHA  string `json:"sha"`
	Repo *struct {
		FullName string `json:"full_name"`
	} `json:"repo"`
}

type pullRequest struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	State     string    `json:"state"`
	Draft     bool      `json:"draft"`
	MergedAt  *string   `json:"merged_at"`
	Mergeable *bool     `json:"mergeable"`
	Head      repoRef   `json:"head"`
	Base      repoRef   `json:"base"`
	User      user      `json:"user"`
	Labels    []label   `json:"labels"`
	UpdatedAt time.Time `json:"updated_at"`
	HTMLURL   string    `json:"html_url"`
	Reviewers []user    `json:"requested_reviewers"`
}

type review struct {
	User  user   `json:"user"`
	State string `json:"state"`
}

type label struct {
	Name string `json:"name"`
}

type pullFile struct {
	Filename         string `json:"filename"`
	PreviousFilename string `json:"previous_filename"`
	Status           string `json:"status"`
	Additions        int    `json:"additions"`
	Deletions        int    `json:"deletions"`
}

type pullCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
		Author  struct {
			Name string    `json:"name"`
			Date time.Time `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

type reviewComment struct {
	ID           int64     `json:"id"`
	InReplyTo    int64     `json:"in_reply_to_id"`
	Path         string    `json:"path"`
	Line         *int      `json:"line"`
	OriginalLine *int      `json:"original_line"`
	Side         string    `json:"side"`
	Body         string    `json:"body"`
	User         user      `json:"user"`
	CreatedAt    time.Time `json:"created_at"`
	CommitID     string    `json:"commit_id"`
	Position     *int      `json:"position"`
}

type checkRun struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Status      string  `json:"status"`
	Conclusion  string  `json:"conclusion"`
	HTMLURL     string  `json:"html_url"`
	StartedAt   *string `json:"started_at"`
	CompletedAt *string `json:"completed_at"`
	App         struct {
		Slug string `json:"slug"`
	} `json:"app"`
}

type checkRuns struct {
	CheckRuns []checkRun `json:"check_runs"`
}

type commitStatus struct {
	State    string `json:"state"`
	Statuses []struct {
		ID        int64  `json:"id"`
		Context   string `json:"context"`
		State     string `json:"state"`
		TargetURL string `json:"target_url"`
	} `json:"statuses"`
}

type issue struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	State       string `json:"state"`
	HTMLURL     string `json:"html_url"`
	PullRequest *struct {
		URL string `json:"url"`
	} `json:"pull_request"`
	Body   string `json:"body"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Milestone *struct {
		Title string `json:"title"`
	} `json:"milestone"`
	User      user      `json:"user"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type issueComment struct {
	User      user      `json:"user"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type rule struct {
	Type string `json:"type"`
}

func (p *Provider) OpenChanges(ctx context.Context, project string) ([]hostapi.Change, bool, error) {
	var changes []hostapi.Change
	truncated, err := p.client.GetPages(ctx, p.host, repoPath(project, "pulls"),
		url.Values{"state": {"open"}, "per_page": {"100"}}, func(resp *hostapi.Response) error {
			var page []pullRequest
			if err := hostapi.DecodeJSON(resp, &page); err != nil {
				return err
			}
			for _, pr := range page {
				changes = append(changes, toChange(pr))
			}
			return nil
		})
	return changes, truncated, hostapi.WithScope(err, scope)
}

// ClosedChanges is one page of 50, closed and merged alike, since GitHub
// lists both under one state.
func (p *Provider) ClosedChanges(ctx context.Context, project string) ([]hostapi.Change, bool, error) {
	resp, err := p.client.Get(ctx, p.host, repoPath(project, "pulls"),
		url.Values{"state": {"closed"}, "sort": {"updated"}, "direction": {"desc"}, "per_page": {"50"}})
	if err != nil {
		return nil, false, hostapi.WithScope(err, scope)
	}
	var page []pullRequest
	if err := hostapi.DecodeJSON(resp, &page); err != nil {
		return nil, false, err
	}
	changes := make([]hostapi.Change, 0, len(page))
	for _, pr := range page {
		changes = append(changes, toChange(pr))
	}
	return changes, hostapi.HasNextPage(resp), nil
}

func (p *Provider) ChangeFiles(ctx context.Context, project string, number int) ([]hostapi.ChangeFile, error) {
	var files []hostapi.ChangeFile
	_, err := p.client.GetPages(ctx, p.host, repoPath(project, "pulls", strconv.Itoa(number), "files"),
		url.Values{"per_page": {"100"}}, func(resp *hostapi.Response) error {
			var page []pullFile
			if err := hostapi.DecodeJSON(resp, &page); err != nil {
				return err
			}
			for _, f := range page {
				files = append(files, hostapi.ChangeFile{
					Path: f.Filename, OldPath: f.PreviousFilename, Status: fileStatus(f.Status),
					Additions: f.Additions, Deletions: f.Deletions,
				})
			}
			return nil
		})
	return files, hostapi.WithScope(err, scope)
}

func (p *Provider) ChangeCommits(ctx context.Context, project string, number int) ([]hostapi.ChangeCommit, error) {
	var commits []hostapi.ChangeCommit
	_, err := p.client.GetPages(ctx, p.host, repoPath(project, "pulls", strconv.Itoa(number), "commits"),
		url.Values{"per_page": {"100"}}, func(resp *hostapi.Response) error {
			var page []pullCommit
			if err := hostapi.DecodeJSON(resp, &page); err != nil {
				return err
			}
			for _, c := range page {
				subject, _, _ := strings.Cut(c.Commit.Message, "\n")
				commits = append(commits, hostapi.ChangeCommit{SHA: c.SHA, Subject: subject, Author: c.Commit.Author.Name, Date: c.Commit.Author.Date})
			}
			return nil
		})
	return commits, hostapi.WithScope(err, scope)
}

func (p *Provider) Change(ctx context.Context, project string, number int) (hostapi.Change, error) {
	var pr pullRequest
	if err := p.client.GetJSON(ctx, p.host, repoPath(project, "pulls", strconv.Itoa(number)), nil, &pr); err != nil {
		return hostapi.Change{}, hostapi.WithScope(err, scope)
	}
	return toChange(pr), nil
}

// Threads groups review comments by the comment they reply to. The REST
// API doesn't say whether a thread is resolved, so none is.
func (p *Provider) Threads(ctx context.Context, project string, number int, headSHA string) ([]hostapi.Thread, error) {
	var threads []hostapi.Thread
	index := map[int64]int{}
	_, err := p.client.GetPages(ctx, p.host, repoPath(project, "pulls", strconv.Itoa(number), "comments"),
		url.Values{"per_page": {"100"}}, func(resp *hostapi.Response) error {
			var page []reviewComment
			if err := hostapi.DecodeJSON(resp, &page); err != nil {
				return err
			}
			for _, c := range page {
				note := hostapi.Note{Author: toUser(c.User), Body: c.Body, CreatedAt: c.CreatedAt}
				if i, ok := index[c.InReplyTo]; ok && c.InReplyTo != 0 {
					threads[i].Notes = append(threads[i].Notes, note)
					continue
				}
				index[c.ID] = len(threads)
				threads = append(threads, toThread(c, headSHA, note))
			}
			return nil
		})
	return threads, hostapi.WithScope(err, scope)
}

// Reviews keeps each reviewer's latest verdict, where a later comment
// doesn't undo an approval or a request for changes. A reviewer asked again
// is shown as requested, as the host does.
func (p *Provider) Reviews(ctx context.Context, project string, change hostapi.Change) ([]hostapi.Review, error) {
	var reviews []hostapi.Review
	_, err := p.client.GetPages(ctx, p.host, repoPath(project, "pulls", strconv.Itoa(change.Number), "reviews"),
		url.Values{"per_page": {"100"}}, func(resp *hostapi.Response) error {
			var page []review
			if err := hostapi.DecodeJSON(resp, &page); err != nil {
				return err
			}
			for _, r := range page {
				state, ok := reviewState(r.State)
				if !ok || (state == hostapi.ReviewCommented && hasVerdict(reviews, r.User.Login)) {
					continue
				}
				reviews = hostapi.SetReview(reviews, hostapi.Review{Reviewer: toUser(r.User), State: state})
			}
			return nil
		})
	if err != nil {
		return nil, hostapi.WithScope(err, scope)
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

// LatestPipeline is the commit's check runs and commit statuses read as one
// result. GitHub has no pipeline id, so the id is the start of the SHA,
// which Jobs turns back into a commit reference.
func (p *Provider) LatestPipeline(ctx context.Context, project, sha string) (hostapi.Pipeline, bool, error) {
	jobs, err := p.commitJobs(ctx, project, sha)
	if err != nil || len(jobs) == 0 {
		return hostapi.Pipeline{}, false, err
	}
	statuses := make([]hostapi.CIStatus, 0, len(jobs))
	for _, job := range jobs {
		statuses = append(statuses, job.Status)
	}
	id, err := pipelineID(sha)
	if err != nil {
		return hostapi.Pipeline{}, false, err
	}
	return hostapi.Pipeline{ID: id, Status: combine(statuses), WebURL: p.webRoot() + project + "/commit/" + sha + "/checks"}, true, nil
}

// Jobs gets back the abbreviated commit the pipeline id was made from; the
// host resolves it like any short SHA.
func (p *Provider) Jobs(ctx context.Context, project string, pipelineID int64) ([]hostapi.Job, error) {
	return p.commitJobs(ctx, project, fmt.Sprintf("%0*x", pipelineIDChars, pipelineID))
}

func (p *Provider) commitJobs(ctx context.Context, project, ref string) ([]hostapi.Job, error) {
	var runs checkRuns
	if err := p.client.GetJSON(ctx, p.host, repoPath(project, "commits", ref, "check-runs"), url.Values{"per_page": {"100"}}, &runs); err != nil {
		return nil, hostapi.WithScope(err, scope)
	}
	var status commitStatus
	if err := p.client.GetJSON(ctx, p.host, repoPath(project, "commits", ref, "status"), nil, &status); err != nil {
		return nil, hostapi.WithScope(err, scope)
	}
	jobs := make([]hostapi.Job, 0, len(runs.CheckRuns)+len(status.Statuses))
	for _, run := range runs.CheckRuns {
		jobs = append(jobs, hostapi.Job{
			ID:        run.ID,
			Name:      run.Name,
			Stage:     run.App.Slug,
			Status:    checkStatus(run),
			Duration:  duration(run.StartedAt, run.CompletedAt),
			WebURL:    run.HTMLURL,
			Retryable: run.App.Slug == actionsApp && hostapi.Retryable(checkStatus(run)),
		})
	}
	for _, s := range status.Statuses {
		jobs = append(jobs, hostapi.Job{ID: s.ID, Name: s.Context, Stage: "status", Status: statusState(s.State), WebURL: s.TargetURL})
	}
	return jobs, nil
}

// JobLog reads an Actions job's log; a check run made by Actions has the
// job's id. The host answers with a redirect to the log's storage, which the
// client follows without the token.
func (p *Provider) JobLog(ctx context.Context, project string, jobID int64, maxBytes int) (string, bool, error) {
	resp, err := p.client.Get(ctx, p.host, repoPath(project, "actions", "jobs", strconv.FormatInt(jobID, 10), "logs"), nil)
	if err != nil {
		return "", false, hostapi.WithScope(err, scope)
	}
	log, truncated := hostapi.TailLog(resp.Body, maxBytes)
	return log, truncated, nil
}

func (p *Provider) Issue(ctx context.Context, project string, number int) (hostapi.Issue, error) {
	var is issue
	if err := p.client.GetJSON(ctx, p.host, repoPath(project, "issues", strconv.Itoa(number)), nil, &is); err != nil {
		return hostapi.Issue{}, hostapi.WithScope(err, scope)
	}
	return toIssue(is), nil
}

// AssignedIssues leaves out pull requests, which GitHub lists as issues too.
func (p *Provider) AssignedIssues(ctx context.Context, project, username string) ([]hostapi.Issue, error) {
	var page []issue
	query := url.Values{"assignee": {username}, "state": {"open"}, "sort": {"updated"}, "per_page": {"50"}}
	if err := p.client.GetJSON(ctx, p.host, repoPath(project, "issues"), query, &page); err != nil {
		return nil, hostapi.WithScope(err, scope)
	}
	issues := make([]hostapi.Issue, 0, len(page))
	for _, is := range page {
		if is.PullRequest == nil {
			issues = append(issues, toIssue(is))
		}
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
	if err := p.client.GetJSON(ctx, p.host, repoPath(project, "issues", strconv.Itoa(number)), nil, &is); err != nil {
		return hostapi.IssueDetails{}, hostapi.WithScope(err, scope)
	}
	return toIssueDetails(is), nil
}

func (p *Provider) IssueComments(ctx context.Context, project, key string) ([]hostapi.Note, error) {
	number, err := hostapi.IssueNumber(key)
	if err != nil {
		return nil, err
	}
	var page []issueComment
	query := url.Values{"per_page": {"100"}}
	if err := p.client.GetJSON(ctx, p.host, repoPath(project, "issues", strconv.Itoa(number), "comments"), query, &page); err != nil {
		return nil, hostapi.WithScope(err, scope)
	}
	notes := make([]hostapi.Note, 0, len(page))
	for _, c := range page {
		notes = append(notes, hostapi.Note{Author: toUser(c.User), Body: c.Body, CreatedAt: c.CreatedAt})
	}
	return notes, nil
}

func toIssue(is issue) hostapi.Issue {
	return hostapi.Issue{Key: strconv.Itoa(is.Number), Number: is.Number, Title: is.Title, State: is.State, WebURL: is.HTMLURL}
}

func toIssueDetails(is issue) hostapi.IssueDetails {
	labels := make([]string, 0, len(is.Labels))
	for _, l := range is.Labels {
		labels = append(labels, l.Name)
	}
	out := hostapi.IssueDetails{Issue: toIssue(is), Description: is.Body, Labels: labels, Author: toUser(is.User), CreatedAt: is.CreatedAt, UpdatedAt: is.UpdatedAt}
	if is.Milestone != nil {
		out.Milestone = is.Milestone.Title
	}
	return out
}

func (p *Provider) CreateChange(ctx context.Context, project string, change hostapi.NewChange) (hostapi.Change, error) {
	body := map[string]any{
		"head":  change.SourceBranch,
		"base":  change.TargetBranch,
		"title": change.Title,
		"body":  change.Description,
		"draft": change.Draft,
	}
	var pr pullRequest
	if err := p.client.PostJSON(ctx, p.host, repoPath(project, "pulls"), body, &pr); err != nil {
		return hostapi.Change{}, hostapi.WithScope(err, scope)
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
		err := p.client.GetJSON(ctx, p.host, repoPath(project, "rules", "branches", url.PathEscape(branch)), nil, &applied)
		var hostErr *hostapi.HostError
		if errors.As(err, &hostErr) && (hostErr.Status == http.StatusNotFound || hostErr.Status == http.StatusForbidden) {
			return rules, nil
		}
		if err != nil {
			return nil, hostapi.WithScope(err, scope)
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
	if err := p.client.GetJSON(ctx, p.host, "user", nil, &u); err != nil {
		return hostapi.User{}, hostapi.WithScope(err, scope)
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

// A comment whose position is gone was left on code that has changed since.
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

func checkStatus(run checkRun) hostapi.CIStatus {
	if run.Status != "completed" {
		if run.Status == "in_progress" {
			return hostapi.CIRunning
		}
		return hostapi.CIPending
	}
	switch run.Conclusion {
	case "success":
		return hostapi.CISuccess
	case "failure", "timed_out", "action_required", "startup_failure":
		return hostapi.CIFailed
	case "cancelled":
		return hostapi.CICanceled
	case "skipped", "neutral", "stale":
		return hostapi.CISkipped
	}
	return hostapi.CIUnknown
}

func statusState(state string) hostapi.CIStatus {
	switch state {
	case "success":
		return hostapi.CISuccess
	case "failure", "error":
		return hostapi.CIFailed
	case "pending":
		return hostapi.CIPending
	}
	return hostapi.CIUnknown
}

// combine reads several results as one, the way the host's own merge box
// does: a failure first, then anything still going, then a cancellation.
func combine(statuses []hostapi.CIStatus) hostapi.CIStatus {
	seen := map[hostapi.CIStatus]bool{}
	for _, s := range statuses {
		seen[s] = true
	}
	for _, s := range []hostapi.CIStatus{hostapi.CIFailed, hostapi.CIRunning, hostapi.CIPending, hostapi.CICanceled, hostapi.CISuccess, hostapi.CISkipped} {
		if seen[s] {
			return s
		}
	}
	return hostapi.CIUnknown
}

func duration(started, completed *string) time.Duration {
	if started == nil || completed == nil {
		return 0
	}
	from, err1 := time.Parse(time.RFC3339, *started)
	to, err2 := time.Parse(time.RFC3339, *completed)
	if err1 != nil || err2 != nil || to.Before(from) {
		return 0
	}
	return to.Sub(from)
}

// pipelineID is the first 15 hex digits of the SHA, which fit an int64 and
// still name the commit to the host.
// GitHub has no pipeline id, so a commit's is its SHA's first hex digits:
// 13 of them, 52 bits, so the id survives being a JavaScript number in the
// frontend (a 53-bit mantissa) and comes back exact.
const pipelineIDChars = 13

func pipelineID(sha string) (int64, error) {
	if len(sha) < pipelineIDChars {
		return 0, fmt.Errorf("%q is not a full commit id", sha)
	}
	return strconv.ParseInt(sha[:pipelineIDChars], 16, 64)
}

// webRoot is the pages' address: github.com for its own API host, else
// the Enterprise Server itself, which serves the API under /api/v3.
func (p *Provider) webRoot() string {
	if p.host.Name == "github.com" || p.host.Name == "api.github.com" {
		return "https://github.com/"
	}
	return "https://" + p.host.Name + "/"
}

// repoPath builds "repos/<owner>/<repo>/<parts…>".
func repoPath(project string, parts ...string) string {
	return strings.Join(append([]string{"repos", project}, parts...), "/")
}
