//go:build !nohost

package app

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// The seams the rest of app calls into host integration through. nohost.go
// stubs each for a build made with -tags nohost; a new seam needs a stub there.
const hostIntegrationBuilt = true

func hostServices() []application.Service {
	return []application.Service{application.NewService(&HostService{})}
}

func afterFetch(ctx context.Context, repoPath string) {
	refreshHostChanges(ctx, repoPath)
}

func trackerReading(hostName string) bool { return hostEnabled(hostName) }
