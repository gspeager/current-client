package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gspeager/current-client/core/hostapi"
)

var _ hostapi.ReleasePublisher = (*Provider)(nil)

type release struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	HTMLURL string `json:"html_url"`
}

// PublishRelease checks the tag is on the host first: given a tag it doesn't
// have, GitHub would make one at the default branch's tip.
func (p *Provider) PublishRelease(ctx context.Context, project string, r hostapi.NewRelease) (hostapi.PublishedRelease, error) {
	var ref struct{}
	err := p.client.GetJSON(ctx, p.host, repoPath(project, "git", "ref", "tags", strings.ReplaceAll(url.PathEscape(r.Tag), "%2F", "/")), nil, &ref)
	var hostErr *hostapi.HostError
	if errors.As(err, &hostErr) && hostErr.Status == http.StatusNotFound {
		return hostapi.PublishedRelease{}, fmt.Errorf("%s has no tag %s yet; push it first", p.host.Name, r.Tag)
	}
	if err != nil {
		return hostapi.PublishedRelease{}, hostapi.WithScope(err, scope)
	}
	var created release
	err = p.client.PostJSON(ctx, p.host, repoPath(project, "releases"), map[string]string{"tag_name": r.Tag, "name": r.Name, "body": r.Notes}, &created)
	if errors.As(err, &hostErr) && hostErr.Status == http.StatusUnprocessableEntity {
		return hostapi.PublishedRelease{}, fmt.Errorf("%s refused the release: %s already has one", p.host.Name, r.Tag)
	}
	if err != nil {
		return hostapi.PublishedRelease{}, hostapi.WithScope(err, scope)
	}
	return hostapi.PublishedRelease{Tag: created.TagName, Name: created.Name, WebURL: created.HTMLURL}, nil
}
