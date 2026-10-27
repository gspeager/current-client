package hostapi

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
)

// The normalized model every provider maps its host onto. Nothing in it is
// GitLab's or GitHub's, and nothing in it is an avatar: people are a name
// and a username, shown as initials badges.

type ChangeState string

const (
	ChangeOpen   ChangeState = "open"
	ChangeMerged ChangeState = "merged"
	ChangeClosed ChangeState = "closed"
)

type CIStatus string

const (
	CIUnknown  CIStatus = ""
	CISuccess  CIStatus = "success"
	CIFailed   CIStatus = "failed"
	CIRunning  CIStatus = "running"
	CIPending  CIStatus = "pending"
	CICanceled CIStatus = "canceled"
	CISkipped  CIStatus = "skipped"
)

type User struct {
	Name     string `json:"name"`
	Username string `json:"username"`
}

// Change is a merge request or pull request. SourceProject is the fork's
// project path when the change comes from one, else empty.
type Change struct {
	Number            int         `json:"number"`
	Title             string      `json:"title"`
	Description       string      `json:"description"`
	State             ChangeState `json:"state"`
	Draft             bool        `json:"draft"`
	SourceBranch      string      `json:"sourceBranch"`
	SourceProject     string      `json:"sourceProject"`
	TargetBranch      string      `json:"targetBranch"`
	Author            User        `json:"author"`
	Labels            []string    `json:"labels"`
	HeadSHA           string      `json:"headSha"`
	CI                CIStatus    `json:"ci"`
	UnresolvedThreads int         `json:"unresolvedThreads"`
	HasConflicts      bool        `json:"hasConflicts"`
	// ConflictsKnown says HasConflicts came from the host; GitHub's list
	// entries carry no answer.
	ConflictsKnown bool      `json:"conflictsKnown"`
	Reviews        []Review  `json:"reviews"`
	UpdatedAt      time.Time `json:"updatedAt"`
	WebURL         string    `json:"webUrl"`
}

type ReviewState string

const (
	ReviewRequested        ReviewState = "requested"
	ReviewApproved         ReviewState = "approved"
	ReviewChangesRequested ReviewState = "changes_requested"
	ReviewCommented        ReviewState = "commented"
)

// Review is where one reviewer stands on a change: asked and not yet done,
// or their latest verdict.
type Review struct {
	Reviewer User        `json:"reviewer"`
	State    ReviewState `json:"state"`
}

// SetReview puts a reviewer's state in place of what reviews had for them.
func SetReview(reviews []Review, review Review) []Review {
	for i := range reviews {
		if reviews[i].Reviewer.Username == review.Reviewer.Username {
			reviews[i].State = review.State
			return reviews
		}
	}
	return append(reviews, review)
}

// ChangeFile is one file a change touches; Status is added, deleted,
// modified or renamed, OldPath set only for a rename.
type ChangeFile struct {
	Path      string `json:"path"`
	OldPath   string `json:"oldPath"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

type ChangeCommit struct {
	SHA     string    `json:"sha"`
	Subject string    `json:"subject"`
	Author  string    `json:"author"`
	Date    time.Time `json:"date"`
}

type NewChange struct {
	SourceBranch string
	TargetBranch string
	Title        string
	Description  string
	Draft        bool
}

type Pipeline struct {
	ID     int64    `json:"id"`
	Status CIStatus `json:"status"`
	WebURL string   `json:"webUrl"`
}

// Job is one CI job. Retryable says the host's API can run it again: a
// failed or canceled GitLab job, or GitHub Actions job.
type Job struct {
	ID        int64         `json:"id"`
	Name      string        `json:"name"`
	Stage     string        `json:"stage"`
	Status    CIStatus      `json:"status"`
	Duration  time.Duration `json:"duration"`
	WebURL    string        `json:"webUrl"`
	Retryable bool          `json:"retryable"`
}

// CIRetrier runs CI again: one job, or every failed job of a pipeline. jobs
// is the pipeline's job list as last read, for a host that re-runs by
// workflow run rather than by pipeline.
type CIRetrier interface {
	RetryJob(ctx context.Context, project string, job Job) error
	RetryPipeline(ctx context.Context, project string, pipelineID int64, jobs []Job) error
}

// Thread is a review discussion. Path and Line are where it was left; Line
// is 0 for a thread on the change as a whole. Outdated means the code it
// was left on has changed since.
type Thread struct {
	ID       string `json:"id"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	OldLine  int    `json:"oldLine"`
	Resolved bool   `json:"resolved"`
	Outdated bool   `json:"outdated"`
	Notes    []Note `json:"notes"`
}

type Note struct {
	Author    User      `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

// Issue is one issue on a tracker. Key is how its references name it:
// "123" for a host's #123, "PROJ-123" for Jira. Number is the host's
// number, 0 on a tracker whose keys aren't numbers.
type Issue struct {
	Key    string `json:"key"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	WebURL string `json:"webUrl"`
}

// IssueDetails is what an issue's panel shows beyond the summary. It is a
// type of its own so Issue stays comparable.
type IssueDetails struct {
	Issue
	Description string    `json:"description"`
	Labels      []string  `json:"labels"`
	Milestone   string    `json:"milestone"`
	Author      User      `json:"author"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// IssueTracker is where a repository's issues live: the code host itself,
// or a tracker beside it such as Jira. Every issue feature reads through
// it, so a tracker that isn't the host has somewhere to go. It adds no
// trigger for requests: callers keep their cache and refresh rules.
type IssueTracker interface {
	// IssueByKey reads one issue by the key its references use.
	IssueByKey(ctx context.Context, project, key string) (Issue, error)
	// AssignedIssues is one page of the open issues assigned to username,
	// most recently updated first.
	AssignedIssues(ctx context.Context, project, username string) ([]Issue, error)
	// KeyPattern matches one reference in text, with the key in group 1,
	// so the app can find issues in commit messages and change titles.
	KeyPattern() *regexp.Regexp
	// KeyInBranch is the key a branch is named after ("123-fix-login",
	// "PROJ-123-fix-login"), or "".
	KeyInBranch(branch string) string
	// Reference is how text names the key: "#123", "PROJ-123".
	Reference(key string) string
	// IssueDetails is the issue with its description, labels, milestone
	// and author, for its panel.
	IssueDetails(ctx context.Context, project, key string) (IssueDetails, error)
	// IssueComments is the discussion on an issue, oldest first, without
	// the tracker's own system notes.
	IssueComments(ctx context.Context, project, key string) ([]Note, error)
}

type NewIssue struct {
	Title       string
	Description string
	Labels      []string
}

// IssueWriter is a tracker the app can write to: GitLab and GitHub. Jira
// isn't one, since teams move Jira issues through their host's own Jira
// integration.
type IssueWriter interface {
	CreateIssue(ctx context.Context, project string, issue NewIssue) (Issue, error)
	CommentOnIssue(ctx context.Context, project, key, body string) (Note, error)
	SetIssueState(ctx context.Context, project, key string, open bool) (IssueDetails, error)
}

// BranchRule is what the host forbids on branches matching Pattern, which
// may end in a wildcard ("release/*"). NoPush means nobody pushes directly;
// RestrictedPush that only some roles or people may.
type BranchRule struct {
	Pattern        string `json:"pattern"`
	NoPush         bool   `json:"noPush"`
	RestrictedPush bool   `json:"restrictedPush"`
	NoForcePush    bool   `json:"noForcePush"`
	NoDelete       bool   `json:"noDelete"`
}

// Matches follows the hosts: * stands for any run of characters.
func (r BranchRule) Matches(branch string) bool {
	return matchWildcard(r.Pattern, branch)
}

func matchWildcard(pattern, name string) bool {
	before, after, found := strings.Cut(pattern, "*")
	if !found {
		return pattern == name
	}
	rest, ok := strings.CutPrefix(name, before)
	if !ok {
		return false
	}
	for i := 0; i <= len(rest); i++ {
		if matchWildcard(after, rest[i:]) {
			return true
		}
	}
	return false
}

// MatchRule picks the rule for a branch: an exact one before a wildcard.
func MatchRule(rules []BranchRule, branch string) (BranchRule, bool) {
	var matched BranchRule
	found := false
	for _, rule := range rules {
		if !rule.Matches(branch) {
			continue
		}
		if rule.Pattern == branch {
			return rule, true
		}
		if !found {
			matched, found = rule, true
		}
	}
	return matched, found
}

// Todo is one entry on the host's to-do list for the user, GitLab's To-Do
// List or GitHub's notifications: why the host put it there, and the change
// or issue it is about. Project is the repository's path on the host.
// Target is "change", "issue" or "" for anything else, such as a commit.
type Todo struct {
	ID        string
	Project   string
	Reason    string
	Target    string
	Number    int
	Title     string
	Author    User
	WebURL    string
	UpdatedAt time.Time
}

// Reasons a host gives for a to-do that the Inbox merges with the rows it
// already has; the rest are shown as the host words them.
const (
	TodoAssigned        = "assigned"
	TodoMentioned       = "mentioned"
	TodoReviewRequested = "review requested"
)

// TodoList is a host that keeps a to-do list for the user, which a Provider
// may also be. The list is account-wide: Todos is one request however many
// repositories it covers, and the caller keeps only what it needs.
type TodoList interface {
	// Todos is the user's pending to-dos, one page, newest first.
	Todos(ctx context.Context) ([]Todo, error)
	// MarkTodoDone takes a to-do off the list; it needs write scope.
	MarkTodoDone(ctx context.Context, id string) error
}

// ErrTodosUnavailable means the host won't list to-dos for this token, which
// is the case for GitHub's notifications with a fine-grained token.
var ErrTodosUnavailable = errors.New("the host won't list to-dos for this token")

// Merge methods a host may allow.
const (
	MergeMethodMerge  = "merge"
	MergeMethodSquash = "squash"
	MergeMethodRebase = "rebase"
)

// MergeOptions is how a host merges: the methods it allows, the one it
// prefers first, and whether it deletes the source branch after.
// DeleteBranchFixed means the host decides that for every merge, as GitHub
// does; otherwise each merge may say.
type MergeOptions struct {
	Methods           []string `json:"methods"`
	DeleteBranch      bool     `json:"deleteBranch"`
	DeleteBranchFixed bool     `json:"deleteBranchFixed"`
}

// MergeRequest is one merge: the method, whether to delete the source
// branch where the host lets a merge say, and the head the merge is meant
// for, so a change that moved since is refused rather than merged blind.
type MergeRequest struct {
	Method       string
	DeleteBranch bool
	HeadSHA      string
}

// MergeRefusedError is a merge the host declined, with its reason in plain
// words from the status it gave; the response body is never read.
type MergeRefusedError struct {
	Host   string
	Status int
	Reason string
}

func (e *MergeRefusedError) Error() string {
	return e.Host + " refused the merge: " + e.Reason + "."
}

// Merger is a host that merges changes, which a Provider may also be.
type Merger interface {
	// MergeOptions is one request for the project's settings.
	MergeOptions(ctx context.Context, project string) (MergeOptions, error)
	// Merge needs write scope.
	Merge(ctx context.Context, project string, number int, how MergeRequest) error
}

// ReviewCapabilities says which review writes a host's API offers; what
// it lacks is left out of the app rather than faked. GitHub's REST API
// can't resolve a thread; GitLab's has no request-changes verdict.
type ReviewCapabilities struct {
	Reply          bool `json:"reply"`
	Resolve        bool `json:"resolve"`
	LineComments   bool `json:"lineComments"`
	Approve        bool `json:"approve"`
	RequestChanges bool `json:"requestChanges"`
}

// NewComment is a comment on a line of a change's diff: Line on the new
// side, or OldLine on the old side for a removed line.
type NewComment struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	OldLine int    `json:"oldLine"`
	Body    string `json:"body"`
}

// Review verdicts.
const (
	ReviewApprove        = "approve"
	ReviewRequestChanges = "request-changes"
	ReviewComment        = "comment"
)

// ReviewSubmission is a review submitted together: a verdict, an optional
// summary, and the line comments drafted for it.
type ReviewSubmission struct {
	Verdict  string       `json:"verdict"`
	Summary  string       `json:"summary"`
	Comments []NewComment `json:"comments"`
}

// ErrReviewUnsupported means the host's API has no way to do this; the
// message says what.
var ErrReviewUnsupported = errors.New("the host's API can't do this")

// Reviewer is a host that takes review writes, which a Provider may also
// be. Every method needs write scope and is sent only on the user's click.
// headSHA is the head the comments were written against.
type Reviewer interface {
	ReviewCapabilities() ReviewCapabilities
	Reply(ctx context.Context, project string, number int, threadID, body string) error
	ResolveThread(ctx context.Context, project string, number int, threadID string, resolved bool) error
	CommentOnLine(ctx context.Context, project string, number int, headSHA string, comment NewComment) error
	SubmitReview(ctx context.Context, project string, number int, headSHA string, review ReviewSubmission) error
}

// Provider is what a host offers this release's features. Every method
// goes through the one Client, so the enabled set, Git's trust settings,
// rate limits and the page cap apply to all of them. project is the path
// on the host, such as "group/sub/repo".
type Provider interface {
	// OpenChanges lists open changes; truncated says the page cap cut it.
	OpenChanges(ctx context.Context, project string) (changes []Change, truncated bool, err error)
	// ClosedChanges is one page of merged and closed changes, newest
	// updated first; truncated says there were more.
	ClosedChanges(ctx context.Context, project string) (changes []Change, truncated bool, err error)
	Change(ctx context.Context, project string, number int) (Change, error)
	ChangeFiles(ctx context.Context, project string, number int) ([]ChangeFile, error)
	ChangeCommits(ctx context.Context, project string, number int) ([]ChangeCommit, error)
	// Threads needs the change's head SHA to tell which threads are outdated.
	Threads(ctx context.Context, project string, number int, headSHA string) ([]Thread, error)
	// Reviews is the change's reviewers, starting from the requested ones
	// its list entry carried.
	Reviews(ctx context.Context, project string, change Change) ([]Review, error)
	// LatestPipeline is the newest pipeline for a commit; ok is false when
	// the host has none for it.
	LatestPipeline(ctx context.Context, project, sha string) (pipeline Pipeline, ok bool, err error)
	Jobs(ctx context.Context, project string, pipelineID int64) ([]Job, error)
	// JobLog returns up to maxBytes from the end of a job's log.
	JobLog(ctx context.Context, project string, jobID int64, maxBytes int) (log string, truncated bool, err error)
	// Issue reads one issue by number. Deprecated: IssueTracker.IssueByKey
	// takes the key a reference carries and works for every tracker.
	Issue(ctx context.Context, project string, number int) (Issue, error)
	// AssignedIssues is IssueTracker's, kept here so a Provider is one.
	AssignedIssues(ctx context.Context, project, username string) ([]Issue, error)
	CreateChange(ctx context.Context, project string, change NewChange) (Change, error)
	// BranchRules are the host's protection rules; branches names the ones
	// to ask about on a host whose endpoint takes a branch. A host that
	// won't show them to this token answers with none, not an error.
	BranchRules(ctx context.Context, project string, branches []string) ([]BranchRule, error)
	CurrentUser(ctx context.Context) (User, error)
}
