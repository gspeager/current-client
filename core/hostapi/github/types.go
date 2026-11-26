package github

import "time"

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
	MergeSHA  string    `json:"merge_commit_sha"`
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
