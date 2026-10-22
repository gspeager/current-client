//go:build !nohost

package app

import "github.com/wailsapp/wails/v3/pkg/application"

const hostIntegrationBuilt = true

func hostServices() []application.Service {
	return []application.Service{application.NewService(&HostService{})}
}
