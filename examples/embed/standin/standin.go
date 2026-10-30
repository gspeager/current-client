// Package standin is how a host app uses Current Client's host integration
// for its own features: it turns its host on through Current Client, and
// asks it through Current Client's one client, so the user's switch, the
// organization's policy.json, Git's trust and proxy settings and the cache
// all apply as they do to Current Client itself.
package standin

import (
	"context"

	"github.com/gspeager/current-client/app"
	"github.com/gspeager/current-client/core/hostapi"
)

// Status is what the host app's own view would show for its host.
type Status struct {
	OK bool `json:"ok"`
}

// TurnOnAndCheck is the host app's own prompt saying yes: the host is
// turned on, unless policy.json locks it off, and asked once.
func TurnOnAndCheck(ctx context.Context, host hostapi.Host) (Status, error) {
	if err := app.SetHostEnabled(host.Name, true); err != nil {
		return Status{}, err
	}
	var status Status
	err := app.HostClient().GetJSON(ctx, host, "status", nil, &status)
	return status, err
}
