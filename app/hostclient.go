package app

import (
	"sync"

	"github.com/gspeager/current-client/core/hostapi"
)

// hostClient is the app's one client for host APIs. It is handed what
// Settings says is enabled, so core/ itself refuses every other host.
var hostClient = sync.OnceValue(func() *hostapi.Client { return newHostClient(nil) })

func newHostClient(transport hostapi.Transport) *hostapi.Client {
	return hostapi.New(hostapi.Options{Enabled: hostEnabled, Transport: transport})
}

func hostEnabled(hostName string) bool {
	return configOrDefault().Hosts[hostName].Enabled
}
