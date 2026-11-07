package gitlab

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gspeager/current-client/core/hostapi"
)

var _ hostapi.Merger = (*Provider)(nil)

// MergeOptions reads the project's squash setting: never, always, or a
// default either way. The project's own merge method (merge commit,
// semi-linear or fast-forward) applies to a plain merge, so "merge" stands
// for whichever it is.
func (p *Provider) MergeOptions(ctx context.Context, project string) (hostapi.MergeOptions, error) {
	var settings struct {
		MergeMethod       string `json:"merge_method"`
		SquashOption      string `json:"squash_option"`
		DeleteBranchAfter bool   `json:"remove_source_branch_after_merge"`
	}
	if err := p.client.GetJSON(ctx, p.host, projectPath(project), nil, &settings); err != nil {
		return hostapi.MergeOptions{}, hostapi.WithScope(err, readScope)
	}
	options := hostapi.MergeOptions{DeleteBranch: settings.DeleteBranchAfter}
	switch settings.SquashOption {
	case "always":
		options.Methods = []string{hostapi.MergeMethodSquash}
	case "default_on":
		options.Methods = []string{hostapi.MergeMethodSquash, hostapi.MergeMethodMerge}
	case "default_off":
		options.Methods = []string{hostapi.MergeMethodMerge, hostapi.MergeMethodSquash}
	default:
		options.Methods = []string{hostapi.MergeMethodMerge}
	}
	return options, nil
}

func (p *Provider) Merge(ctx context.Context, project string, number int, how hostapi.MergeRequest) error {
	body := map[string]any{
		"squash":                      how.Method == hostapi.MergeMethodSquash,
		"should_remove_source_branch": how.DeleteBranch,
	}
	if how.HeadSHA != "" {
		body["sha"] = how.HeadSHA
	}
	err := p.client.RequestJSON(ctx, p.host, http.MethodPut, projectPath(project, "merge_requests", strconv.Itoa(number), "merge"), body, nil)
	return hostapi.WithScope(hostapi.RefusedMerge(err, p.host.Name, mergeRefusals), writeScope)
}

var mergeRefusals = map[int]string{
	http.StatusUnauthorized:        "this token's user may not merge it",
	http.StatusMethodNotAllowed:    "it can't be merged yet, so a pipeline, approval, thread or rule is in the way, or it's a draft",
	http.StatusNotAcceptable:       "it conflicts with the target branch",
	http.StatusConflict:            "its head moved since the last refresh; refresh and look again",
	http.StatusUnprocessableEntity: "the merge failed on the host",
}
