//go:build !nohost

package app

import (
	"sync"

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/internal/policy"
)

// For an embedding app's own host features: the same client, cache,
// enabled set and organization lock as Current Client's, so the embedding
// app never runs a second client that ignores policy.json. None of it is in
// the nohost build.

// HostClient is Current Client's one client for host APIs, with its cache.
// Requests through it meet the same enabled set, Git's trust and proxy
// settings, back-off and page cap; read what Current Client already fetched
// from its Cache.
func HostClient() *hostapi.Client {
	return hostClient
}

// HostEnabled is whether the client may contact a host or tracker now: the
// user's switch unless policy.json fixes it, which managed says.
func HostEnabled(hostname string) (enabled, managed bool) {
	_, managed = policy.Host(hostname)
	return hostEnabled(hostname), managed
}

// EnabledHosts is the effective enabled set: every host and tracker the
// client may contact now.
func EnabledHosts() []string {
	candidates := map[string]bool{}
	for name := range configOrDefault().Hosts {
		candidates[name] = true
	}
	for name := range policy.Hosts() {
		candidates[name] = true
	}
	for _, t := range allTrackers() {
		candidates[trackerHostName(t.Site)] = true
	}
	var enabled []string
	for name := range candidates {
		if hostEnabled(name) {
			enabled = append(enabled, name)
		}
	}
	return enabled
}

// SetHostEnabled turns a host on or off from the embedding app's own
// prompt, as Settings does: refused while policy.json fixes it, and turning
// it off drops everything held from it.
func SetHostEnabled(hostname string, enabled bool) error {
	return (&HostService{}).SetEnabled(hostname, enabled)
}

var (
	hostWatchers   = map[int]func(string){}
	hostWatchersID int
	hostWatchersMu sync.Mutex
)

// OnHostsChanged calls watch with the hostname whenever a host's or a
// tracker's switch changes, from Settings, the first-use prompt or
// SetHostEnabled. The returned function stops it.
func OnHostsChanged(watch func(hostname string)) (stop func()) {
	hostWatchersMu.Lock()
	defer hostWatchersMu.Unlock()
	hostWatchersID++
	id := hostWatchersID
	hostWatchers[id] = watch
	return func() {
		hostWatchersMu.Lock()
		defer hostWatchersMu.Unlock()
		delete(hostWatchers, id)
	}
}

func hostsChanged(hostname string) {
	hostWatchersMu.Lock()
	watchers := make([]func(string), 0, len(hostWatchers))
	for _, watch := range hostWatchers {
		watchers = append(watchers, watch)
	}
	hostWatchersMu.Unlock()
	for _, watch := range watchers {
		watch(hostname)
	}
}

// ProviderFactory builds a provider for a host on Current Client's client;
// host is the hostname and the API address Settings or detection gives it.
type ProviderFactory func(client *hostapi.Client, host hostapi.Host) hostapi.Provider

var (
	extraProviders   = map[string]ProviderFactory{}
	extraProvidersMu sync.Mutex
)

// RegisterProvider gives a host Current Client doesn't know, such as an
// embedding app's own stand-in host, a provider beside GitLab and GitHub.
// It is used for that hostname wherever Current Client asks a host, and
// only while the host is on, like any other.
func RegisterProvider(hostname string, factory ProviderFactory) {
	extraProvidersMu.Lock()
	defer extraProvidersMu.Unlock()
	extraProviders[hostname] = factory
}

func extraProvider(hostname string) (ProviderFactory, bool) {
	extraProvidersMu.Lock()
	defer extraProvidersMu.Unlock()
	factory, ok := extraProviders[hostname]
	return factory, ok
}
