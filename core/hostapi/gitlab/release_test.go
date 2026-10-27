package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
)

func TestPublishReleaseSendsWhatGitLabExpects(t *testing.T) {
	p, f := newProvider(t, map[string]string{
		"POST " + api + "/releases": `{"tag_name": "v1.2.0", "name": "1.2.0", "_links": {"self": "https://gitlab.com/gitlab-org/cli/-/releases/v1.2.0"}}`,
	})
	published, err := p.PublishRelease(context.Background(), project, hostapi.NewRelease{Tag: "v1.2.0", Name: "1.2.0", Notes: "- Issues."})
	if err != nil || published.WebURL != "https://gitlab.com/gitlab-org/cli/-/releases/v1.2.0" {
		t.Fatalf("PublishRelease = %+v, %v", published, err)
	}
	var body map[string]string
	json.Unmarshal([]byte(f.requests[0].Body), &body)
	if body["tag_name"] != "v1.2.0" || body["description"] != "- Issues." || body["ref"] != "" || len(body) != 3 {
		t.Errorf("body = %v", body)
	}
}

func TestPublishReleaseRefusals(t *testing.T) {
	for status, want := range map[string]string{"409": "already has one", "422": "push it first", "400": "push it first"} {
		p, _ := newProvider(t, map[string]string{"POST " + api + "/releases": status})
		_, err := p.PublishRelease(context.Background(), project, hostapi.NewRelease{Tag: "v1"})
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v", status, err)
		}
	}
	p, _ := newProvider(t, map[string]string{"POST " + api + "/releases": "403"})
	_, err := p.PublishRelease(context.Background(), project, hostapi.NewRelease{Tag: "v1"})
	var hostErr *hostapi.HostError
	if !errors.As(err, &hostErr) || hostErr.Scope != writeScope {
		t.Fatalf("403: %v", err)
	}
}
