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
	// "", "test", "generated" or "lockfile".
	Kind string `json:"kind"`
}

// Files is every file at rev, a tag or HEAD, with its lines. Before the first
// commit there are none.
func (s *ExploreService) Files(repoPath, rev string) ([]MapFileInfo, error) {
	ctx := context.Background()
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

// The periods commits are counted over, in days; 0 is all that's read.
var mapPeriods = []int{30, 90, 365, 0}

// Reading every commit of a very long history takes minutes.
const mostMapCommits = 20000

type MapActivityInfo struct {
	// In mapPeriods' order.
	Commits []int  `json:"commits"`
	Last    string `json:"last"`
}

type MapActivity struct {
	Files map[string]MapActivityInfo `json:"files"`
	// The commits read, when the history goes further back; 0 when it doesn't.
	Limit int `json:"limit"`
}

func (s *ExploreService) Activity(repoPath string) (MapActivity, error) {
	ctx := context.Background()
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

// ExploreComparison is what Explore's views compare: "working" for uncommitted
// changes, "branch" for a branch since it left its base, or "release" for HEAD since
// a tag. Ref is the branch or the tag.
type ExploreComparison struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref"`
}

type MapChangeInfo struct {
	Path     string `json:"path"`
	OrigPath string `json:"origPath"`
	// A, M, D, R or T.
	Status  string `json:"status"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
	Binary  bool   `json:"binary"`
	Kind    string `json:"kind"`
	// Predicted to conflict with the branch's target.
	Conflict bool `json:"conflict"`
}

// MapChangesInfo.Target is a branch's target, "" when there's nothing to compare
// with. Predicted is false when Git is too old to predict conflicts.
type MapChangesInfo struct {
	Target    string          `json:"target"`
	Files     []MapChangeInfo `json:"files"`
	Skipped   int             `json:"skipped"`
	Predicted bool            `json:"predicted"`
}

func (s *ExploreService) Changes(repoPath string, comparison ExploreComparison) (MapChangesInfo, error) {
	ctx := context.Background()
	changes := MapChangesInfo{Files: []MapChangeInfo{}, Predicted: true}
	if noCommits(ctx, repoPath) {
		return changes, nil
	}
	head, err := git.ResolveCommit(ctx, repoPath, "HEAD")
	if err != nil {
		return changes, err
	}
	r := readthrough.Range{Base: head, Head: head}
	var conflicts []string
	switch comparison.Kind {
	case "working":
		r.Uncommitted = true
	case "branch":
		info := ReadThroughInfo{Key: "refs/heads/" + comparison.Ref, Name: comparison.Ref}
		if info.Head, err = git.ResolveCommit(ctx, repoPath, info.Key); err != nil {
			return changes, err
		}
		if err := info.settleBase(ctx, repoPath); err != nil || info.Target == "" {
			return changes, err
		}
		r = info.changes()
		changes.Target = info.Target
		conflicts, err = git.MergeConflicts(ctx, repoPath, info.Target, info.Head)
		if errors.Is(err, git.ErrMergeTreeUnsupported) {
			changes.Predicted = false
		} else if err != nil {
			return changes, err
		}
	case "release":
		if r.Base, err = git.ResolveCommit(ctx, repoPath, "refs/tags/"+comparison.Ref); err != nil {
			return changes, err
		}
	default:
		return changes, &gitexec.AppError{Message: "Unknown comparison.", Detail: comparison.Kind}
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
