package github

import (
	"context"
	"net/url"
	"strconv"
	"time"

	"github.com/gspeager/current-client/core/hostapi"
)

var _ hostapi.Deployer = (*Provider)(nil)

// deploymentStatusCap bounds the status requests one refresh makes: one per
// environment, newest environments first.
const deploymentStatusCap = 10

type deployment struct {
	ID          int64     `json:"id"`
	SHA         string    `json:"sha"`
	Ref         string    `json:"ref"`
	Environment string    `json:"environment"`
	Production  *bool     `json:"production_environment"`
	CreatedAt   time.Time `json:"created_at"`
	Creator     user      `json:"creator"`
}

type deploymentStatus struct {
	State string `json:"state"`
}

// Deployments reads one page of deployments, keeps the first per
// environment, and asks each of those for its latest status: GitHub keeps
// the state there rather than on the deployment.
func (p *Provider) Deployments(ctx context.Context, project string) ([]hostapi.Deployment, error) {
	var page []deployment
	if err := p.client.GetJSON(ctx, p.host, repoPath(project, "deployments"), url.Values{"per_page": {"100"}}, &page); err != nil {
		return nil, hostapi.WithScope(err, scope)
	}
	var out []hostapi.Deployment
	seen := map[string]bool{}
	for _, d := range page {
		if d.Environment == "" || seen[d.Environment] {
			continue
		}
		seen[d.Environment] = true
		production := hostapi.IsProductionName(d.Environment)
		if d.Production != nil {
			production = *d.Production
		}
		item := hostapi.Deployment{
			Environment: d.Environment,
			SHA:         d.SHA,
			Ref:         d.Ref,
			Status:      hostapi.CIPending,
			DeployedAt:  d.CreatedAt,
			DeployedBy:  toUser(d.Creator),
			WebURL:      p.webRoot() + project + "/deployments/" + url.PathEscape(d.Environment),
			Production:  production,
		}
		if len(out) < deploymentStatusCap {
			var statuses []deploymentStatus
			path := repoPath(project, "deployments", strconv.FormatInt(d.ID, 10), "statuses")
			if err := p.client.GetJSON(ctx, p.host, path, url.Values{"per_page": {"1"}}, &statuses); err != nil {
				return nil, hostapi.WithScope(err, scope)
			}
			if len(statuses) > 0 {
				item.Status = deploymentState(statuses[0].State)
			}
		} else {
			item.Status = hostapi.CIUnknown
		}
		out = append(out, item)
	}
	return out, nil
}

func deploymentState(state string) hostapi.CIStatus {
	switch state {
	case "success", "inactive":
		return hostapi.CISuccess
	case "failure", "error":
		return hostapi.CIFailed
	case "in_progress":
		return hostapi.CIRunning
	}
	return hostapi.CIPending
}
