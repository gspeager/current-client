// Package github is the hostapi.Provider for github.com and GitHub
// Enterprise Server, over the REST API. The two differ only in address.
package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
}

type label struct {
	Name string `json:"name"`
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
	Number  int    `json:"number"`
	Title   string `json:"title"`
	State   string `json:"state"`
	HTMLURL string `json:"html_url"`
}

type rule struct {
	Type string `json:"type"`
}

func (p *Provider) OpenChanges(ctx context.Context, project string) ([]hostapi.Change, bool, error) {
	var changes []hostapi.Change
	truncated, err := p.client.GetPages(ctx, p.host, repoPath(project, "pulls"),
		url.Values{"state": {"open"}, "per_page": {"100"}}, func(resp *hostapi.Response) error {
			var page []pullRequest
			if err := decode(resp, &page); err != nil {
				return err
			}
			for _, pr := range page {
				changes = append(changes, toChange(pr))
			}
			return nil
		})
	return changes, truncated, scoped(err)
}

func (p *Provider) Change(ctx context.Context, project string, number int) (hostapi.Change, error) {
	var pr pullRequest
	if err := p.client.GetJSON(ctx, p.host, repoPath(project, "pulls", itoa(number)), nil, &pr); err != nil {
		return hostapi.Change{}, scoped(err)
	}
	return toChange(pr), nil
}

// Threads groups review comments by the comment they reply to. The REST
// API doesn't say whether a thread is resolved, so none is.
func (p *Provider) Threads(ctx context.Context, project string, number int, headSHA string) ([]hostapi.Thread, error) {
	var threads []hostapi.Thread
	index := map[int64]int{}
	_, err := p.client.GetPages(ctx, p.host, repoPath(project, "pulls", itoa(number), "comments"),
		url.Values{"per_page": {"100"}}, func(resp *hostapi.Response) error {
			var page []reviewComment
			if err := decode(resp, &page); err != nil {
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
	return threads, scoped(err)
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

func (p *Provider) Jobs(ctx context.Context, project string, pipelineID int64) ([]hostapi.Job, error) {
	return p.commitJobs(ctx, project, fmt.Sprintf("%015x", pipelineID))
}

func (p *Provider) commitJobs(ctx context.Context, project, ref string) ([]hostapi.Job, error) {
	var runs checkRuns
	if err := p.client.GetJSON(ctx, p.host, repoPath(project, "commits", ref, "check-runs"), url.Values{"per_page": {"100"}}, &runs); err != nil {
		return nil, scoped(err)
	}
	var status commitStatus
	if err := p.client.GetJSON(ctx, p.host, repoPath(project, "commits", ref, "status"), nil, &status); err != nil {
		return nil, scoped(err)
	}
	jobs := make([]hostapi.Job, 0, len(runs.CheckRuns)+len(status.Statuses))
	for _, run := range runs.CheckRuns {
		jobs = append(jobs, hostapi.Job{
			ID:       run.ID,
			Name:     run.Name,
			Stage:    run.App.Slug,
			Status:   checkStatus(run),
			Duration: duration(run.StartedAt, run.CompletedAt),
			WebURL:   run.HTMLURL,
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
		return "", false, scoped(err)
	}
	body := resp.Body
	if maxBytes > 0 && len(body) > maxBytes {
		cut := body[len(body)-maxBytes:]
		if i := strings.IndexByte(string(cut), '\n'); i >= 0 {
			cut = cut[i+1:]
		}
		return string(cut), true, nil
	}
	return string(body), false, nil
}

func (p *Provider) Issue(ctx context.Context, project string, number int) (hostapi.Issue, error) {
	var is issue
	if err := p.client.GetJSON(ctx, p.host, repoPath(project, "issues", itoa(number)), nil, &is); err != nil {
		return hostapi.Issue{}, scoped(err)
	}
	return hostapi.Issue{Number: is.Number, Title: is.Title, State: is.State, WebURL: is.HTMLURL}, nil
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
		return hostapi.Change{}, scoped(err)
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
		err := p.client.GetJSON(ctx, p.host, repoPath(project, "rules", "branches", escapeBranch(branch)), nil, &applied)
		var hostErr *hostapi.HostError
		if errors.As(err, &hostErr) && (hostErr.Status == http.StatusNotFound || hostErr.Status == http.StatusForbidden) {
			return rules, nil
		}
		if err != nil {
			return nil, scoped(err)
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
		return hostapi.User{}, scoped(err)
	}
	return toUser(u), nil
}

func toChange(pr pullRequest) hostapi.Change {
	c := hostapi.Change{
		Number:       pr.Number,
		Title:        pr.Title,
		Description:  pr.Body,
		State:        changeState(pr),
		Draft:        pr.Draft,
		SourceBranch: pr.Head.Ref,
		TargetBranch: pr.Base.Ref,
		Author:       toUser(pr.User),
		HeadSHA:      pr.Head.SHA,
		HasConflicts: pr.Mergeable != nil && !*pr.Mergeable,
		UpdatedAt:    pr.UpdatedAt,
		WebURL:       pr.HTMLURL,
	}
	for _, l := range pr.Labels {
		c.Labels = append(c.Labels, l.Name)
	}
	if pr.Head.Repo != nil && pr.Base.Repo != nil && pr.Head.Repo.FullName != pr.Base.Repo.FullName {
		c.SourceProject = pr.Head.Repo.FullName
	}
	return c
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
func pipelineID(sha string) (int64, error) {
	if len(sha) < 15 {
		return 0, fmt.Errorf("%q is not a full commit id", sha)
	}
	return strconv.ParseInt(sha[:15], 16, 64)
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

func escapeBranch(branch string) string {
	return url.PathEscape(branch)
}

func decode(resp *hostapi.Response, out any) error {
	return hostapi.DecodeJSON(resp, out)
}

func itoa(n int) string { return strconv.Itoa(n) }

func scoped(err error) error {
	var hostErr *hostapi.HostError
	if errors.As(err, &hostErr) && hostErr.Scope == "" {
		hostErr.Scope = scope
	}
	return err
}
