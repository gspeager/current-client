package github

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
)

func TestPublishReleaseNeedsTheTagOnTheHost(t *testing.T) {
	p, f := newProvider(t, map[string]string{
		"GET " + v3 + "/git/ref/tags/v1.2.0": `{"ref": "refs/tags/v1.2.0"}`,
		"POST " + v3 + "/releases":           `{"tag_name": "v1.2.0", "name": "1.2.0", "html_url": "https://github.com/cli/cli/releases/tag/v1.2.0"}`,
		"GET " + v3 + "/git/ref/tags/v9":     "404",
	})
	ctx := context.Background()
	published, err := p.PublishRelease(ctx, project, hostapi.NewRelease{Tag: "v1.2.0", Name: "1.2.0", Notes: "- Issues."})
	if err != nil || published.WebURL == "" || published.Tag != "v1.2.0" {
		t.Fatalf("PublishRelease = %+v, %v", published, err)
	}
	var body map[string]string
	json.Unmarshal([]byte(f.requests[1].Body), &body)
	if body["tag_name"] != "v1.2.0" || body["name"] != "1.2.0" || body["body"] != "- Issues." || len(body) != 3 {
		t.Errorf("body = %v", body)
	}
	if _, err := p.PublishRelease(ctx, project, hostapi.NewRelease{Tag: "v9"}); err == nil || !strings.Contains(err.Error(), "push it first") {
		t.Fatalf("a tag the host lacks: %v", err)
	}
	if f.count("POST", v3+"/releases") != 1 {
		t.Fatal("a release was sent for a tag the host lacks")
	}
}

func TestPublishReleaseSaysWhenOneExists(t *testing.T) {
	p, _ := newProvider(t, map[string]string{
		"GET " + v3 + "/git/ref/tags/v1": `{}`,
		"POST " + v3 + "/releases":       "422",
	})
	_, err := p.PublishRelease(context.Background(), project, hostapi.NewRelease{Tag: "v1"})
	if err == nil || !strings.Contains(err.Error(), "already has one") {
		t.Fatalf("err = %v", err)
	}
}
