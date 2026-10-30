package github

import (
	"context"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
)

func TestDeploymentsAskEachEnvironmentsStatus(t *testing.T) {
	p, f := newProvider(t, map[string]string{
		"GET " + v3 + "/deployments?per_page=100": `[
			{"id": 3, "sha": "c3", "ref": "main", "environment": "staging", "created_at": "2026-10-26T10:00:00Z", "creator": {"login": "ada"}},
			{"id": 2, "sha": "c2", "ref": "v1.0", "environment": "live", "production_environment": true, "created_at": "2026-10-25T10:00:00Z"},
			{"id": 1, "sha": "c1", "ref": "main", "environment": "staging", "created_at": "2026-10-20T10:00:00Z"},
			{"id": 4, "sha": "c4", "ref": "main", "environment": "production", "created_at": "2026-10-19T10:00:00Z"}
		]`,
		"GET " + v3 + "/deployments/3/statuses?per_page=1": `[{"state": "failure"}]`,
		"GET " + v3 + "/deployments/2/statuses?per_page=1": `[{"state": "success"}]`,
		"GET " + v3 + "/deployments/4/statuses?per_page=1": `[]`,
	})
	got, err := p.Deployments(context.Background(), project)
	if err != nil || len(got) != 3 {
		t.Fatalf("Deployments = %+v, %v", got, err)
	}
	if got[0].Environment != "staging" || got[0].Status != hostapi.CIFailed || got[0].Production || got[0].DeployedBy.Username != "ada" {
		t.Errorf("staging = %+v", got[0])
	}
	if !got[1].Production || got[1].Status != hostapi.CISuccess || got[1].WebURL != "https://github.example.com/cli/cli/deployments/live" {
		t.Errorf("live = %+v", got[1])
	}
	if got[2].Status != hostapi.CIPending || !got[2].Production {
		t.Errorf("an environment whose deployment has no status yet = %+v", got[2])
	}
	if f.count("GET", v3+"/deployments/1/statuses") != 0 {
		t.Error("an older deployment's status was asked")
	}
}
