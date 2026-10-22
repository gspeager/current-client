//go:build !nohost

package app

import (
	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostapi/gitlab"
	"github.com/gspeager/current-client/core/hostlink"
)

// providerFor is the host's API through the one client, or nil for a
// provider type that has no implementation yet.
func providerFor(info HostInfo) hostapi.Provider {
	host := hostapi.Host{Name: info.Hostname, API: info.APIURL}
	switch hostlink.Style(info.Provider) {
	case hostlink.GitLab:
		return gitlab.New(hostClient, host)
	}
	return nil
}
