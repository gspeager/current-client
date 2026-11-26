package gitlab

import (
	"time"
)

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
	MergeCommitSHA  string    `json:"merge_commit_sha"`
	SquashCommitSHA string    `json:"squash_commit_sha"`
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

type protectedBranch struct {
	Name             string `json:"name"`
	AllowForcePush   bool   `json:"allow_force_push"`
	PushAccessLevels []struct {
		AccessLevel int `json:"access_level"`
	} `json:"push_access_levels"`
}
