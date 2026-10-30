package gitlab

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gspeager/current-client/core/hostapi"
)

var _ hostapi.ReleasePublisher = (*Provider)(nil)

type release struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Links   struct {
		Self string `json:"self"`
	} `json:"_links"`
}

// PublishRelease sends no ref, so GitLab refuses a tag it doesn't have
// rather than making one.
func (p *Provider) PublishRelease(ctx context.Context, project string, r hostapi.NewRelease) (hostapi.PublishedRelease, error) {
	var created release
	err := p.client.PostJSON(ctx, p.host, projectPath(project, "releases"), map[string]string{"tag_name": r.Tag, "name": r.Name, "description": r.Notes}, &created)
	var hostErr *hostapi.HostError
	if errors.As(err, &hostErr) {
		switch hostErr.Status {
		case http.StatusConflict:
			return hostapi.PublishedRelease{}, fmt.Errorf("%s refused the release: %s already has one", p.host.Name, r.Tag)
		case http.StatusBadRequest, http.StatusUnprocessableEntity:
			return hostapi.PublishedRelease{}, fmt.Errorf("%s has no tag %s yet; push it first", p.host.Name, r.Tag)
		}
	}
	if err != nil {
		return hostapi.PublishedRelease{}, hostapi.WithScope(err, writeScope)
	}
	return hostapi.PublishedRelease{Tag: created.TagName, Name: created.Name, WebURL: created.Links.Self}, nil
}
