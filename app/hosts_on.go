//go:build !nohost

package app

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const hostIntegrationBuilt = true

func hostServices() []application.Service {
	return []application.Service{application.NewService(&HostService{})}
}

// afterFetch refreshes the change numbers and, for hosts that are on, the
// open changes, since a fetch is one of the refresh triggers.
func afterFetch(ctx context.Context, repoPath string) {
	refreshHostChanges(ctx, repoPath)
}

func trackerReading(hostName string) bool { return hostEnabled(hostName) }
