package app

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/gspeager/current-client/core/filekind"
	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
	"github.com/gspeager/current-client/core/readthrough"
)

type ExploreService struct{}

type MapFileInfo struct {
	Path   string `json:"path"`
	Lines  int    `json:"lines"`
	Binary bool   `json:"binary"`
	// Kind is "", "test", "generated" or "lockfile".
	Kind string `json:"kind"`
}

// Files is every file at rev, a tag or HEAD, with its lines. Before the first
// commit there are none.
func (s *ExploreService) Files(ctx context.Context, repoPath, rev string) ([]MapFileInfo, error) {
	if rev == "HEAD" && noCommits(ctx, repoPath) {
		return []MapFileInfo{}, nil
	}
	files, err := git.LineCounts(ctx, repoPath, rev)
	if err != nil {
		return nil, err
	}
	return mapSlice(files, func(f git.FileLines) MapFileInfo {
		return MapFileInfo{Path: f.Path, Lines: f.Lines, Binary: f.Binary, Kind: string(filekind.Of(f.Path))}
	}), nil
}

func noCommits(ctx context.Context, repoPath string) bool {
	_, err := git.ResolveCommit(ctx, repoPath, "HEAD")
	return err != nil
}

// The periods commits are counted over, in days; 0 counts every commit read.
var mapPeriods = []int{30, 90, 365, 0}

// mostMapCommits caps the commits Activity reads: reading every commit of a
// very long history takes minutes.
const mostMapCommits = 20000

type MapActivityInfo struct {
	// Commits is a count for each of mapPeriods, in its order.
	Commits []int  `json:"commits"`
	Last    string `json:"last"`
}

type MapActivity struct {
	Files map[string]MapActivityInfo `json:"files"`
	// Limit is the number of commits read when the history goes further back;
	// 0 when every commit was read.
	Limit int `json:"limit"`
}

func (s *ExploreService) Activity(ctx context.Context, repoPath string) (MapActivity, error) {
	if noCommits(ctx, repoPath) {
		return MapActivity{Files: map[string]MapActivityInfo{}}, nil
	}
	files, more, err := git.FileCommits(ctx, repoPath, "HEAD", mapPeriods, mostMapCommits)
	if err != nil {
		return MapActivity{}, err
	}
	activity := MapActivity{Files: make(map[string]MapActivityInfo, len(files))}
	for _, file := range files {
		activity.Files[file.Path] = MapActivityInfo{Commits: file.Commits, Last: file.Last.Format(time.RFC3339)}
	}
	if more {
		activity.Limit = mostMapCommits
	}
	return activity, nil
}

// ExploreComparison is what Explore's views compare.
type ExploreComparison struct {
	// Kind is "working" for uncommitted changes, "branch" for a branch since it
	// left its base, "release" for HEAD since a tag, or "commit" for HEAD since
	// any other point.
	Kind string `json:"kind"`
	// Ref is the branch, the tag or the point.
	Ref string `json:"ref"`
}

type MapChangeInfo struct {
	Path     string `json:"path"`
	OrigPath string `json:"origPath"`
	// Status is A, M, D, R or T.
	Status  string `json:"status"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
	Binary  bool   `json:"binary"`
	Kind    string `json:"kind"`
	// Conflict is predicted against the branch's target.
	Conflict bool `json:"conflict"`
}

type MapChangesInfo struct {
	// Target is a branch's target, "" when there's nothing to compare with.
	Target  string          `json:"target"`
	Files   []MapChangeInfo `json:"files"`
	Skipped int             `json:"skipped"`
	// Predicted is false when Git is too old to predict conflicts.
	Predicted bool `json:"predicted"`
}

func (s *ExploreService) Changes(ctx context.Context, repoPath string, comparison ExploreComparison) (MapChangesInfo, error) {
	changes := MapChangesInfo{Files: []MapChangeInfo{}, Predicted: true}
	if noCommits(ctx, repoPath) {
		return changes, nil
	}
	r, target, err := comparedRange(ctx, repoPath, comparison)
	if err != nil || (comparison.Kind == "branch" && target == "") {
		return changes, err
	}
	changes.Target = target
	var conflicts []string
	if target != "" {
		conflicts, err = git.MergeConflicts(ctx, repoPath, target, r.Head)
		if errors.Is(err, git.ErrMergeTreeUnsupported) {
			changes.Predicted = false
		} else if err != nil {
			return changes, err
		}
	}
	files, skipped, err := readthrough.Files(ctx, repoPath, r)
	if err != nil {
		return changes, err
	}
	changes.Skipped = skipped
	for _, f := range files {
		changes.Files = append(changes.Files, MapChangeInfo{
			Path:     f.Path,
			OrigPath: f.OrigPath,
			Status:   f.Status[:1],
			Added:    f.Added,
			Removed:  f.Removed,
			Binary:   f.Binary,
			Kind:     string(f.Kind),
			Conflict: slices.Contains(conflicts, f.Path),
		})
	}
	return changes, nil
}

// comparedRange is what a comparison is from and to. target is a branch's, "" when
// there's nothing to compare it with.
func comparedRange(ctx context.Context, repoPath string, comparison ExploreComparison) (r readthrough.Range, target string, err error) {
	head, err := git.ResolveCommit(ctx, repoPath, "HEAD")
	if err != nil {
		return r, "", err
	}
	r = readthrough.Range{Base: head, Head: head}
	switch comparison.Kind {
	case "working":
		r.Uncommitted = true
	case "branch":
		info := ReadThroughInfo{Key: "refs/heads/" + comparison.Ref, Name: comparison.Ref}
		if info.Head, err = git.ResolveCommit(ctx, repoPath, info.Key); err != nil {
			return r, "", err
		}
		if err := info.settleBase(ctx, repoPath); err != nil || info.Target == "" {
			return r, "", err
		}
		return info.changes(), info.Target, nil
	case "release":
		r.Base, err = git.ResolveCommit(ctx, repoPath, "refs/tags/"+comparison.Ref)
	case "commit":
		r.Base, err = git.ResolveCommit(ctx, repoPath, comparison.Ref)
	default:
		err = &gitexec.AppError{Message: "Unknown comparison.", Detail: comparison.Kind}
	}
	return r, "", err
}

// mostTimelineCommits caps the commits a timeline reads, the newest kept.
const mostTimelineCommits = 5000

type TimelineFileInfo struct {
	Path  string `json:"path"`
	Lines int    `json:"lines"`
	Kind  string `json:"kind"`
}

type TimelineCommitInfo struct {
	SHA     string             `json:"sha"`
	Time    string             `json:"time"`
	Subject string             `json:"subject"`
	Files   []TimelineFileInfo `json:"files"`
}

type TimelineInfo struct {
	// Newest first.
	Commits []TimelineCommitInfo `json:"commits"`
	// Now is whether the range ends at HEAD, and so at its uncommitted changes, Working.
	Now     bool               `json:"now"`
	Working []TimelineFileInfo `json:"working"`
	// Limit is the commits read when the range goes further back; 0 when it doesn't.
	Limit  int    `json:"limit"`
	Target string `json:"target"`
}

// Timeline is the comparison's commits with each file's lines changed. Only it
// takes a comparison of Kind "since", HEAD's commits since Ref, an RFC 3339 time.
func (s *ExploreService) Timeline(ctx context.Context, repoPath string, comparison ExploreComparison) (TimelineInfo, error) {
	timeline := TimelineInfo{Commits: []TimelineCommitInfo{}, Working: []TimelineFileInfo{}}
	if noCommits(ctx, repoPath) {
		return timeline, nil
	}
	head, err := git.ResolveCommit(ctx, repoPath, "HEAD")
	if err != nil {
		return timeline, err
	}
	r := readthrough.Range{Head: head}
	var since time.Time
	if comparison.Kind == "since" {
		if since, err = time.Parse(time.RFC3339, comparison.Ref); err != nil {
			return timeline, err
		}
	} else if r, timeline.Target, err = comparedRange(ctx, repoPath, comparison); err != nil || (comparison.Kind == "branch" && timeline.Target == "") {
		return timeline, err
	}
	if comparison.Kind != "working" {
		commits, more, err := git.RangeCommitLines(ctx, repoPath, r.Base, r.Head, since, mostTimelineCommits)
		if err != nil {
			return timeline, err
		}
		if more {
			timeline.Limit = mostTimelineCommits
		}
		for _, c := range commits {
			timeline.Commits = append(timeline.Commits, TimelineCommitInfo{
				SHA:     c.SHA,
				Time:    c.Time.Format(time.RFC3339),
				Subject: c.Subject,
				Files: mapSlice(c.Files, func(f git.NumstatEntry) TimelineFileInfo {
					return TimelineFileInfo{Path: f.Path, Lines: f.Added + f.Removed, Kind: string(filekind.Of(f.Path))}
				}),
			})
		}
	}
	if r.Head != head {
		return timeline, nil
	}
	files, _, err := readthrough.Files(ctx, repoPath, readthrough.Range{Base: head, Head: head, Uncommitted: true})
	if err != nil {
		return timeline, err
	}
	timeline.Now = true
	timeline.Working = mapSlice(files, func(f readthrough.File) TimelineFileInfo {
		return TimelineFileInfo{Path: f.Path, Lines: f.Added + f.Removed, Kind: string(f.Kind)}
	})
	return timeline, nil
}
