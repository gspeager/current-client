package app

import (
	"context"

	"github.com/gspeager/current-client/core/git"
)

type HookService struct{}

type HooksInfo struct {
	Dir               string     `json:"dir"`
	Scope             string     `json:"scope"`
	Tracked           bool       `json:"tracked"`
	Worktrees         int        `json:"worktrees"`
	ConfigUnsupported bool       `json:"configUnsupported"`
	Hooks             []HookInfo `json:"hooks"`
}

type HookInfo struct {
	Event   string `json:"event"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	Command string `json:"command"`
	Scope   string `json:"scope"`
	On      bool   `json:"on"`
	Manager string `json:"manager"`
}

func (s *HookService) Hooks(repoPath string) (HooksInfo, error) {
	hooks, err := git.ReadHooks(context.Background(), repoPath)
	if err != nil {
		return HooksInfo{}, err
	}
	return HooksInfo{
		Dir:               hooks.Dir,
		Scope:             hooks.Scope,
		Tracked:           hooks.Tracked,
		Worktrees:         hooks.Worktrees,
		ConfigUnsupported: hooks.ConfigUnsupported,
		Hooks:             mapSlice(hooks.List, func(h git.Hook) HookInfo { return HookInfo(h) }),
	}, nil
}

func (s *HookService) SetHookOn(repoPath string, hook HookInfo, on bool) error {
	return git.SetHookOn(context.Background(), repoPath, git.Hook(hook), on)
}
