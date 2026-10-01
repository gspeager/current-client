package app

import (
	"context"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/graph"
)

type HistoryService struct{}

type CommitInfo struct {
	SHA         string       `json:"sha"`
	ParentSHAs  []string     `json:"parentShas"`
	AuthorName  string       `json:"authorName"`
	AuthorEmail string       `json:"authorEmail"`
	Date        string       `json:"date"`
	Subject     string       `json:"subject"`
	Body        string       `json:"body"`
	Signature   string       `json:"signature"` // "none", "verified", or "unverified"
	CoAuthors   []AuthorInfo `json:"coAuthors"`
}

type ChangedFile struct {
	Status   string `json:"status"`
	Path     string `json:"path"`
	OrigPath string `json:"origPath"`
	Added    int    `json:"added"`
	Removed  int    `json:"removed"`
	Binary   bool   `json:"binary"`
}

type HistoryFilterInfo struct {
	Ref    string `json:"ref"`
	Since  string `json:"since"`
	Until  string `json:"until"`
	Author string `json:"author"`
	Type   string `json:"type"`
}

func (s *HistoryService) GetHistory(repoPath string, limit, skip int, filter HistoryFilterInfo) ([]CommitInfo, error) {
	entries, err := git.History(context.Background(), repoPath, limit, skip, git.HistoryFilter(filter))
	if err != nil {
		return nil, err
	}
	return commitInfos(entries), nil
}

func (s *HistoryService) SearchHistory(repoPath, query string, limit int) ([]CommitInfo, error) {
	entries, err := git.SearchHistory(context.Background(), repoPath, query, limit)
	if err != nil {
		return nil, err
	}
	return commitInfos(entries), nil
}

func (s *HistoryService) FollowsConventionalCommits(repoPath string) (bool, error) {
	return git.FollowsConventionalCommits(context.Background(), repoPath)
}

func (s *HistoryService) GetCommitPosition(repoPath, sha string) (int, error) {
	return git.CommitPosition(context.Background(), repoPath, sha)
}

func (s *HistoryService) ResolveCommit(repoPath, ref string) (string, error) {
	return git.ResolveCommit(context.Background(), repoPath, ref)
}

func (s *HistoryService) GetFileHistory(repoPath, path string, limit, skip int) ([]CommitInfo, error) {
	entries, err := git.FileHistory(context.Background(), repoPath, path, limit, skip)
	if err != nil {
		return nil, err
	}
	return commitInfos(entries), nil
}

func commitInfos(entries []git.HistoryEntry) []CommitInfo {
	return mapSlice(entries, func(e git.HistoryEntry) CommitInfo {
		return CommitInfo{
			SHA:         e.SHA,
			ParentSHAs:  e.ParentSHAs,
			AuthorName:  e.AuthorName,
			AuthorEmail: e.AuthorEmail,
			Date:        e.Date.Format(time.RFC3339),
			Subject:     e.Subject,
			Body:        e.Body,
			Signature:   string(e.Signature),
			CoAuthors:   mapSlice(git.CoAuthors(e.Trailers), func(a git.Author) AuthorInfo { return AuthorInfo(a) }),
		}
	})
}

func (s *HistoryService) GetChangedFiles(repoPath, sha string) ([]ChangedFile, error) {
	ctx := context.Background()
	files, err := git.ChangedFiles(ctx, repoPath, sha)
	if err != nil {
		return nil, err
	}
	stats, err := git.CommitNumstat(ctx, repoPath, sha)
	if err != nil {
		return nil, err
	}
	return withLineCounts(files, stats), nil
}

func withLineCounts(files []git.ChangedFile, stats []git.NumstatEntry) []ChangedFile {
	statsByPath := numstatByPath(stats)
	return mapSlice(files, func(f git.ChangedFile) ChangedFile {
		n := statsByPath[f.Path]
		return ChangedFile{
			Status:   f.Status,
			Path:     f.Path,
			OrigPath: f.OrigPath,
			Added:    n.Added,
			Removed:  n.Removed,
			Binary:   n.Binary,
		}
	})
}

type CommitRef struct {
	SHA        string   `json:"sha"`
	ParentSHAs []string `json:"parentShas"`
}

type GraphNode struct {
	SHA             string `json:"sha"`
	Lane            int    `json:"lane"`
	ParentLanes     []int  `json:"parentLanes"`
	ConvergingLanes []int  `json:"convergingLanes"`
	PassThrough     []int  `json:"passThrough"`
}

func (s *HistoryService) ComputeGraphLayout(commits []CommitRef) []GraphNode {
	nodes := graph.Layout(mapSlice(commits, func(c CommitRef) graph.Commit { return graph.Commit(c) }))
	return mapSlice(nodes, func(n graph.Node) GraphNode { return GraphNode(n) })
}

type RefInfo struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	SHA  string `json:"sha"`
}

func (s *HistoryService) GetRefs(repoPath string) ([]RefInfo, error) {
	refs, err := git.ListRefs(context.Background(), repoPath)
	if err != nil {
		return nil, err
	}
	return mapSlice(refs, func(r git.Ref) RefInfo {
		return RefInfo{Kind: refKindString(r.Kind), Name: r.Name, SHA: r.SHA}
	}), nil
}

func refKindString(k git.RefKind) string {
	switch k {
	case git.RefBranch:
		return "branch"
	case git.RefRemoteBranch:
		return "remote"
	default:
		return "tag"
	}
}
