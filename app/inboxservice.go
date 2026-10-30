package app

import (
	"context"
	"path/filepath"
	"sync"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/overlap"
)

// InboxService lists what waits on the user across the open repositories.
// The local rows come from Git; the host rows come from the cache a refresh
// filled, so opening the Inbox never contacts a host. It remembers which
// host rows the Inbox has shown, so a refresh that brings something new can
// say so.
type InboxService struct {
	mu   sync.Mutex
	seen map[string]map[string]bool
}

const staleAfter = 30 * 24 * time.Hour

// InboxRow is one thing to do. Kind says which: dirty, operation,
// unpushed, conflict, stale, merged (local); review, changes-requested,
// ci-failed, assigned, todo, undeployed (host). FetchedAt is set on host rows: when the
// cache learned it. TodoID is set on a row the host's to-do list has too,
// so it can be marked done there.
type InboxRow struct {
	// ID names a host row across refreshes; local rows have none.
	ID string `json:"id"`
	// New is a host row the Inbox hasn't shown yet.
	New       bool       `json:"new"`
	Kind      string     `json:"kind"`
	Branch    string     `json:"branch"`
	Number    int        `json:"number"`
	Prefix    string     `json:"prefix"`
	Key       string     `json:"key"`
	Title     string     `json:"title"`
	Detail    string     `json:"detail"`
	Count     int        `json:"count"`
	WebURL    string     `json:"webUrl"`
	FetchedAt *time.Time `json:"fetchedAt"`
	TodoID    string     `json:"todoId"`
}

// InboxRepo is one open repository's rows. HostEnabled with no FetchedAt
// means the host is on but nothing has refreshed it this session, so the
// host rows aren't known yet rather than empty. Notice says why the host's
// to-do list is missing, once per host.
type InboxRepo struct {
	Path        string     `json:"path"`
	Name        string     `json:"name"`
	Rows        []InboxRow `json:"rows"`
	Error       string     `json:"error"`
	HostEnabled bool       `json:"hostEnabled"`
	FetchedAt   *time.Time `json:"fetchedAt"`
	Notice      string     `json:"notice"`
}

// Rows is the Inbox's list. With markSeen, the host rows it returns count
// as shown from now on; the view passes true, a count for a badge doesn't.
func (s *InboxService) Rows(paths []string, markSeen bool) []InboxRepo {
	ctx := context.Background()
	repos := make([]InboxRepo, 0, len(paths))
	noticed := map[string]bool{}
	for _, path := range paths {
		repo := InboxRepo{Path: path, Name: filepath.Base(path), Rows: []InboxRow{}}
		rows, err := localInboxRows(ctx, path)
		if err != nil {
			repo.Error = err.Error()
		}
		repo.Rows = append(repo.Rows, rows...)
		hostRows, enabled, fetchedAt, notice := hostInboxRows(path)
		repo.HostEnabled, repo.FetchedAt = enabled, fetchedAt
		if notice != "" && !noticed[notice] {
			noticed[notice], repo.Notice = true, notice
		}
		repo.Rows = append(repo.Rows, s.flagNew(path, hostRows, markSeen)...)
		repos = append(repos, repo)
	}
	return repos
}

// Unseen counts the host rows the Inbox hasn't shown, for the header's
// badge after a refresh. Cache only.
func (s *InboxService) Unseen(paths []string) int {
	n := 0
	for _, path := range paths {
		hostRows, _, _, _ := hostInboxRows(path)
		for _, row := range s.flagNew(path, hostRows, false) {
			if row.New {
				n++
			}
		}
	}
	return n
}

// MarkDone takes a to-do off the host's list, the one write the Inbox makes.
func (s *InboxService) MarkDone(repoPath, todoID string) error {
	return markTodoDone(repoPath, todoID)
}

func (s *InboxService) flagNew(path string, rows []InboxRow, markSeen bool) []InboxRow {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := s.seen[path]
	if seen == nil {
		seen = map[string]bool{}
		if s.seen == nil {
			s.seen = map[string]map[string]bool{}
		}
		s.seen[path] = seen
	}
	for i := range rows {
		rows[i].New = !seen[rows[i].ID]
		if markSeen {
			seen[rows[i].ID] = true
		}
	}
	return rows
}

// localInboxRows needs the branch list; everything else is best effort, so
// one failing check doesn't empty the repository's rows.
func localInboxRows(ctx context.Context, repoPath string) ([]InboxRow, error) {
	branches, err := git.ListBranches(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	var rows []InboxRow
	if state, err := git.DetectConflictState(ctx, repoPath); err == nil && state.Operation != "" {
		rows = append(rows, InboxRow{Kind: "operation", Detail: string(state.Operation), Count: len(state.ConflictedPaths)})
	}
	if files, err := git.GetStatus(ctx, repoPath); err == nil && len(files) > 0 {
		rows = append(rows, InboxRow{Kind: "dirty", Count: len(files)})
	}
	merged := map[string]bool{}
	if report, err := git.MergedBranches(ctx, repoPath); err == nil {
		for _, b := range report.Branches {
			if b.Newer {
				continue
			}
			merged[b.Name] = true
			rows = append(rows, InboxRow{Kind: "merged", Branch: b.Name, Detail: report.Base})
		}
	}
	now := time.Now()
	for _, b := range branches {
		if b.Ahead > 0 {
			rows = append(rows, InboxRow{Kind: "unpushed", Branch: b.Name, Count: b.Ahead})
		}
		if last, err := time.Parse(time.RFC3339, b.LastCommitDate); err == nil && !b.Current && !merged[b.Name] && now.Sub(last) >= staleAfter {
			rows = append(rows, InboxRow{Kind: "stale", Branch: b.Name, Count: int(now.Sub(last).Hours() / 24)})
		}
	}
	// Local branches only: a remote-tracking ref that would conflict isn't
	// the user's to act on, and the sidebar still flags it.
	if overlaps, err := overlap.Predict(ctx, repoPath); err == nil {
		for _, o := range overlaps {
			if !o.Remote {
				rows = append(rows, InboxRow{Kind: "conflict", Branch: o.Branch, Count: len(o.Files), Detail: joinFiles(o.Files)})
			}
		}
	}
	return rows, nil
}

func joinFiles(files []string) string {
	const limit = 3
	out := ""
	for i, f := range files {
		if i == limit {
			return out + ", …"
		}
		if i > 0 {
			out += ", "
		}
		out += f
	}
	return out
}
