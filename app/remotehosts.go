package app

import (
	"context"
	"sync"

	"github.com/gspeager/current-client/core/hostlink"
)

// RemoteHostFunc names the host and project of a remote by its address, for one
// the embedding app knows; ok is false for the rest.
type RemoteHostFunc func(remoteURL string) (hostname, project string, ok bool)

var remoteHosts struct {
	sync.Mutex
	claims []RemoteHostFunc
}

// RegisterRemoteHost lets an embedding app say which host a remote is on when its
// address can't, such as a folder on this computer for a stand-in host
// registered with RegisterProvider. It's asked before the address is read. The
// host is off until turned on, as any other.
func RegisterRemoteHost(claim RemoteHostFunc) {
	remoteHosts.Lock()
	defer remoteHosts.Unlock()
	remoteHosts.claims = append(remoteHosts.claims, claim)
}

func resolveRemote(ctx context.Context, remoteURL string) (hostlink.Host, bool) {
	remoteHosts.Lock()
	claims := remoteHosts.claims
	remoteHosts.Unlock()
	for _, claim := range claims {
		if hostname, project, ok := claim(remoteURL); ok {
			return hostlink.Host{Root: "https://" + hostname + "/", Project: project, Style: hostlink.DefaultStyle(hostname)}, true
		}
	}
	return hostlink.Resolve(ctx, remoteURL)
}
