//go:build !nohost

package app

import (
	"context"
	"encoding/base64"

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostapi/jira"
	"github.com/gspeager/current-client/core/hostlink"
	"github.com/gspeager/current-client/internal/policy"
)

// hostClient is the app's one client for host APIs. It is handed what
// Settings says is enabled, so core/ itself refuses every other host, and
// gets tokens from Git's credential helpers through hostCredentials.
var hostClient = newHostClient(nil)

var hostCredentials = &hostapi.GitCredentials{Saved: func(hostName string) bool {
	return configOrDefault().Hosts[hostName].HasToken
}}

// trackerCredentials holds issue trackers' tokens under the app's own key
// only: no remote signs in to a tracker, so there is no Git credential to
// fall back on.
var trackerCredentials = &hostapi.GitCredentials{OwnKeyOnly: true, Saved: func(hostName string) bool {
	site, ok := trackerSiteByHost(hostName)
	return ok && configOrDefault().TrackerAccess[site.Site].HasToken
}}

func newHostClient(transport hostapi.Transport) *hostapi.Client {
	return hostapi.New(hostapi.Options{Enabled: hostEnabled, Tokens: apiTokens{}, Transport: transport})
}

// hostEnabled is a code host's switch, or an issue tracker's for a tracker's
// host, with the managed policy over either.
func hostEnabled(hostName string) bool {
	cfg := configOrDefault()
	user := cfg.Hosts[hostName].Enabled
	if site, ok := trackerSiteByHost(hostName); ok {
		user = cfg.TrackerAccess[site.Site].Enabled
	}
	enabled, _ := effectiveHostEnabled(hostName, user)
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

// apiTokens writes the Authorization header: a bearer token for a code host
// or a Jira Data Center site, and basic auth from "email:token" for Jira
// Cloud.
type apiTokens struct{}

func (apiTokens) Token(ctx context.Context, hostName string) (string, error) {
	return hostCredentials.Token(ctx, hostName)
}

func (apiTokens) Authorization(ctx context.Context, hostName string) (string, error) {
	site, isTracker := trackerSiteByHost(hostName)
	credentials := hostCredentials
	if isTracker {
		credentials = trackerCredentials
	}
	token, err := credentials.Token(ctx, hostName)
	if err != nil || token == "" {
		return "", err
	}
	if isTracker && site.Kind == hostlink.JiraCloud {
		return "Basic " + base64.StdEncoding.EncodeToString([]byte(token)), nil
	}
	return "Bearer " + token, nil
}

func (apiTokens) Forget(hostName string) {
	hostCredentials.Forget(hostName)
	trackerCredentials.Forget(hostName)
}

// trackerSiteByHost is the configured tracker, the user's or the
// organization's, whose site is on hostName.
func trackerSiteByHost(hostName string) (hostlink.Tracker, bool) {
	for _, t := range allTrackers() {
		if jira.HostName(t) == hostName {
			return t, true
		}
	}
	return hostlink.Tracker{}, false
}
