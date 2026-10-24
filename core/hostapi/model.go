package hostapi

import (
	"context"
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
	Reviews           []Review    `json:"reviews"`
	UpdatedAt         time.Time   `json:"updatedAt"`
	WebURL            string      `json:"webUrl"`
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

type Job struct {
	ID       int64         `json:"id"`
	Name     string        `json:"name"`
	Stage    string        `json:"stage"`
	Status   CIStatus      `json:"status"`
	Duration time.Duration `json:"duration"`
	WebURL   string        `json:"webUrl"`
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

type Issue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	WebURL string `json:"webUrl"`
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
	Issue(ctx context.Context, project string, number int) (Issue, error)
	CreateChange(ctx context.Context, project string, change NewChange) (Change, error)
	// BranchRules are the host's protection rules; branches names the ones
	// to ask about on a host whose endpoint takes a branch. A host that
	// won't show them to this token answers with none, not an error.
	BranchRules(ctx context.Context, project string, branches []string) ([]BranchRule, error)
	CurrentUser(ctx context.Context) (User, error)
}
