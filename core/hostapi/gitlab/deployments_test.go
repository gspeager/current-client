package gitlab

import (
	"context"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
)

func TestDeploymentsKeepTheNewestPerEnvironment(t *testing.T) {
	p, f := newProvider(t, map[string]string{
		"GET " + api + "/deployments?order_by=id&per_page=100&sort=desc": `[
			{"sha": "c3", "ref": "main", "status": "running", "created_at": "2026-10-26T10:00:00Z", "user": {"name": "Ada", "username": "ada"}, "environment": {"id": 2, "name": "staging", "tier": "staging"}},
			{"sha": "c2", "ref": "main", "status": "success", "created_at": "2026-10-25T10:00:00Z", "user": {"username": "bob"}, "environment": {"id": 1, "name": "live", "tier": "production"}},
			{"sha": "c1", "ref": "main", "status": "success", "created_at": "2026-10-20T10:00:00Z", "environment": {"id": 2, "name": "staging", "tier": "staging"}},
			{"sha": "c0", "ref": "main", "status": "created", "created_at": "2026-10-19T10:00:00Z", "environment": {"id": 3, "name": "review/x"}}
		]`,
	})
	got, err := p.Deployments(context.Background(), project)
	if err != nil || len(got) != 3 {
		t.Fatalf("Deployments = %+v, %v", got, err)
	}
	staging, live, review := got[0], got[1], got[2]
	if staging.Environment != "staging" || staging.SHA != "c3" || staging.Status != hostapi.CIRunning || staging.Production || staging.DeployedBy.Username != "ada" ||
		staging.WebURL != "https://gitlab.example.com/gitlab-org/cli/-/environments/2" {
		t.Errorf("staging = %+v", staging)
	}
	if !live.Production || live.SHA != "c2" {
		t.Errorf("a production tier = %+v", live)
	}
	if review.Status != hostapi.CIPending {
		t.Errorf("a deployment not started yet = %+v", review)
	}
	if len(f.requests) != 1 {
		t.Errorf("%d requests, want one", len(f.requests))
	}
}
