//go:build nohost

package app

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Built with -tags nohost: no host API client, no HostService, no
// Integrations settings. For deployments that must not have the code at all.
const hostIntegrationBuilt = false

func hostServices() []application.Service { return nil }

func afterFetch(context.Context, string) {}
