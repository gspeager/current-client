package app

import (
	"context"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/gitexec"
)

type TagService struct{}

type TagInfo struct {
	Name      string `json:"name"`
	SHA       string `json:"sha"`
	Annotated bool   `json:"annotated"`
	Date      string `json:"date"`
}

type TagMessageInfo struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func (s *TagService) ListTags(repoPath string) ([]TagInfo, error) {
	tags, err := git.ListTags(context.Background(), repoPath)
	if err != nil {
		return nil, err
	}
	return mapSlice(tags, func(t git.Tag) TagInfo { return TagInfo(t) }), nil
}

func (s *TagService) CreateTag(repoPath, name, message, target string) error {
	return git.CreateTag(context.Background(), repoPath, name, message, target)
}

func (s *TagService) DeleteTag(repoPath, name string) error {
	return git.DeleteTag(context.Background(), repoPath, name)
}

func (s *TagService) PushTag(ctx context.Context, repoPath, remote, name string, auth *gitexec.Credential) error {
	ctx, err := withAuth(ctx, auth)
	if err != nil {
		return err
	}
	return git.PushTag(ctx, repoPath, remote, name)
}

func (s *TagService) GetTagMessage(repoPath, name string) (TagMessageInfo, error) {
	subject, body, err := git.TagMessage(context.Background(), repoPath, name)
	return TagMessageInfo{Subject: subject, Body: body}, err
}
