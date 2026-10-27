//go:build !nohost

package app

import (
	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostapi/github"
	"github.com/gspeager/current-client/core/hostapi/gitlab"
	"github.com/gspeager/current-client/core/hostlink"
)

// providerFor is the host's API through the one client, or nil for a
// provider type without one.
// issueTrackerFor is the host's own issues; a tracker beside the host
// would be chosen here once there is one.
func issueTrackerFor(info HostInfo) hostapi.IssueTracker {
	if tracker, ok := providerFor(info).(hostapi.IssueTracker); ok {
		return tracker
	}
	return nil
}

func providerFor(info HostInfo) hostapi.Provider {
	host := hostapi.Host{Name: info.Hostname, API: info.APIURL}
	if factory, ok := extraProvider(info.Hostname); ok {
		return factory(hostClient, host)
	}
	switch hostlink.Style(info.Provider) {
	case hostlink.GitLab:
		return gitlab.New(hostClient, host)
	case hostlink.GitHub:
		return github.New(hostClient, host)
	}
	return nil
}
