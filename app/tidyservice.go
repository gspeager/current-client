package app

import (
	"context"

	"github.com/gspeager/current-client/core/git"
)

type TidyService struct{}

type TidyCommitInfo struct {
	SHA       string `json:"sha"`
	Subject   string `json:"subject"`
	Message   string `json:"message"`
	Suggested string `json:"suggested"`
}

type TidyPlanInfo struct {
	Head      string           `json:"head"`
	BaseName  string           `json:"baseName"`
	Commits   []TidyCommitInfo `json:"commits"`
	Dirty     bool             `json:"dirty"`
	HasMerges bool             `json:"hasMerges"`
}

type TidyStepInfo struct {
	SHA     string `json:"sha"`
	Action  string `json:"action"`
	Message string `json:"message"`
}

type TidyResultInfo struct {
	Subject string `json:"subject"`
	Folded  int    `json:"folded"`
}

type TidyOutcomeInfo struct {
	SHA     string   `json:"sha"`
	Outcome string   `json:"outcome"`
	Files   []string `json:"files"`
}

type TidyPreviewInfo struct {
	Result    []TidyResultInfo  `json:"result"`
	Outcomes  []TidyOutcomeInfo `json:"outcomes"`
	Predicted bool              `json:"predicted"`
	ForcePush bool              `json:"forcePush"`
	Upstream  string            `json:"upstream"`
}

func (s *TidyService) Plan(repoPath string) (TidyPlanInfo, error) {
	plan, err := git.PlanTidy(context.Background(), repoPath)
	if err != nil {
		return TidyPlanInfo{}, err
	}
	return TidyPlanInfo{
		Head:     plan.Head,
		BaseName: plan.BaseName,
		Commits: mapSlice(plan.Commits, func(commit git.TidyCommit) TidyCommitInfo {
			return TidyCommitInfo(commit)
		}),
		Dirty:     plan.Dirty,
		HasMerges: plan.HasMerges,
	}, nil
}

func tidySteps(steps []TidyStepInfo) []git.TidyStep {
	return mapSlice(steps, func(step TidyStepInfo) git.TidyStep { return git.TidyStep(step) })
}

func (s *TidyService) Preview(repoPath string, steps []TidyStepInfo) (TidyPreviewInfo, error) {
	preview, err := git.PreviewTidy(context.Background(), repoPath, tidySteps(steps))
	if err != nil {
		return TidyPreviewInfo{}, err
	}
	return TidyPreviewInfo{
		Result: mapSlice(preview.Result, func(result git.TidyResultCommit) TidyResultInfo {
			return TidyResultInfo(result)
		}),
		Outcomes: mapSlice(preview.Replay, func(step git.ReplayStep) TidyOutcomeInfo {
			return TidyOutcomeInfo{step.SHA, string(step.Outcome), append([]string{}, step.Files...)}
		}),
		Predicted: preview.Predicted,
		ForcePush: preview.ForcePush,
		Upstream:  preview.Upstream,
	}, nil
}

func (s *TidyService) Run(repoPath, head string, steps []TidyStepInfo) error {
	return git.RunTidy(context.Background(), repoPath, head, tidySteps(steps))
}
