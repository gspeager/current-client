package gitlab

import (
	"context"
	"net/url"

	"github.com/gspeager/current-client/core/hostapi"
)

// Every request names the token scope it needs, which a refusal reports.

func (p *Provider) get(ctx context.Context, scope, path string, query url.Values, out any) error {
	return hostapi.WithScope(p.client.GetJSON(ctx, p.host, path, query, out), scope)
}

// getBody is a response read as it is, such as a job's log.
func (p *Provider) getBody(ctx context.Context, scope, path string, query url.Values) (*hostapi.Response, error) {
	resp, err := p.client.Get(ctx, p.host, path, query)
	return resp, hostapi.WithScope(err, scope)
}

func (p *Provider) post(ctx context.Context, scope, path string, body, out any) error {
	return hostapi.WithScope(p.client.PostJSON(ctx, p.host, path, body, out), scope)
}

func (p *Provider) send(ctx context.Context, scope, method, path string, body, out any) error {
	return hostapi.WithScope(p.client.RequestJSON(ctx, p.host, method, path, body, out), scope)
}

func getAll[T any](ctx context.Context, p *Provider, scope, path string, query url.Values) ([]T, bool, error) {
	items, truncated, err := hostapi.GetAll[T](ctx, p.client, p.host, path, query)
	return items, truncated, hostapi.WithScope(err, scope)
}

func getPage[T any](ctx context.Context, p *Provider, scope, path string, query url.Values) ([]T, bool, error) {
	items, more, err := hostapi.GetPage[T](ctx, p.client, p.host, path, query)
	return items, more, hostapi.WithScope(err, scope)
}
