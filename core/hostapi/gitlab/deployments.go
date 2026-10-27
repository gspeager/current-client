package gitlab

import (
	"context"
	"net/url"
	"strconv"
	"time"

	"github.com/gspeager/current-client/core/hostapi"
)

var _ hostapi.Deployer = (*Provider)(nil)

type deployment struct {
	SHA         string    `json:"sha"`
	Ref         string    `json:"ref"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	User        user      `json:"user"`
	Environment struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
		Tier string `json:"tier"`
	} `json:"environment"`
}

// Deployments reads one page of the project's newest deployments and keeps
// the first per environment, in one request.
func (p *Provider) Deployments(ctx context.Context, project string) ([]hostapi.Deployment, error) {
	var page []deployment
	query := url.Values{"order_by": {"id"}, "sort": {"desc"}, "per_page": {"100"}}
	if err := p.client.GetJSON(ctx, p.host, projectPath(project, "deployments"), query, &page); err != nil {
		return nil, hostapi.WithScope(err, readScope)
	}
	var out []hostapi.Deployment
	seen := map[string]bool{}
	for _, d := range page {
		name := d.Environment.Name
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, hostapi.Deployment{
			Environment: name,
			SHA:         d.SHA,
			Ref:         d.Ref,
			Status:      ciStatus(d.Status),
			DeployedAt:  d.CreatedAt,
			DeployedBy:  toUser(d.User),
			WebURL:      "https://" + p.host.Name + "/" + project + "/-/environments/" + strconv.FormatInt(d.Environment.ID, 10),
			Production:  d.Environment.Tier == "production" || (d.Environment.Tier == "" && hostapi.IsProductionName(name)),
		})
	}
	return out, nil
}
