package github

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gspeager/current-client/core/hostapi"
)

var _ hostapi.Merger = (*Provider)(nil)

type repository struct {
	AllowMergeCommit  bool `json:"allow_merge_commit"`
	AllowSquashMerge  bool `json:"allow_squash_merge"`
	AllowRebaseMerge  bool `json:"allow_rebase_merge"`
	DeleteBranchAfter bool `json:"delete_branch_on_merge"`
}

func (p *Provider) MergeOptions(ctx context.Context, project string) (hostapi.MergeOptions, error) {
	var repo repository
	if err := p.client.GetJSON(ctx, p.host, repoPath(project), nil, &repo); err != nil {
		return hostapi.MergeOptions{}, hostapi.WithScope(err, scope)
	}
	options := hostapi.MergeOptions{DeleteBranch: repo.DeleteBranchAfter, DeleteBranchFixed: true}
	for _, m := range []struct {
		allowed bool
		method  string
	}{{repo.AllowMergeCommit, hostapi.MergeMethodMerge}, {repo.AllowSquashMerge, hostapi.MergeMethodSquash}, {repo.AllowRebaseMerge, hostapi.MergeMethodRebase}} {
		if m.allowed {
			options.Methods = append(options.Methods, m.method)
		}
	}
	return options, nil
}

// Merge names the head it was decided on, so a pull request that moved
// since the last refresh is refused (409) rather than merged blind.
func (p *Provider) Merge(ctx context.Context, project string, number int, how hostapi.MergeRequest) error {
	body := map[string]any{"merge_method": how.Method}
	if how.HeadSHA != "" {
		body["sha"] = how.HeadSHA
	}
	err := p.client.RequestJSON(ctx, p.host, http.MethodPut, repoPath(project, "pulls", strconv.Itoa(number), "merge"), body, nil)
	return hostapi.WithScope(hostapi.RefusedMerge(err, p.host.Name, mergeRefusals), scope)
}

var mergeRefusals = map[int]string{
	http.StatusMethodNotAllowed:    "it isn't mergeable yet, so a required check, review or rule is in the way, or it conflicts",
	http.StatusConflict:            "its head moved since the last refresh; refresh and look again",
	http.StatusUnprocessableEntity: "that merge method isn't allowed here",
}
