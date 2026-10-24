//go:build !nohost

package app

import (
	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/internal/policy"
)

// hostClient is the app's one client for host APIs. It is handed what
// Settings says is enabled, so core/ itself refuses every other host, and
// gets tokens from Git's credential helpers through hostCredentials.
var hostClient = newHostClient(nil)

var hostCredentials = &hostapi.GitCredentials{Saved: func(hostName string) bool {
	return configOrDefault().Hosts[hostName].HasToken
}}

func newHostClient(transport hostapi.Transport) *hostapi.Client {
	return hostapi.New(hostapi.Options{Enabled: hostEnabled, Tokens: hostCredentials, Transport: transport})
}

func hostEnabled(hostName string) bool {
	enabled, _ := effectiveHostEnabled(hostName, configOrDefault().Hosts[hostName].Enabled)
	return enabled
}

// effectiveHostEnabled is the user's switch unless the managed policy fixes
// it; managed says the policy did.
func effectiveHostEnabled(hostName string, user bool) (enabled, managed bool) {
	if forced, managed := policy.Host(hostName); managed {
		return forced, true
	}
	return user, false
}
