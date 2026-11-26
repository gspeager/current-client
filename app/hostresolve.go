//go:build !nohost

package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/hostapi"
)

var errHostOff = errors.New("integration is off for this repository's host")

// How long a host request may take, and one that streams or writes a lot, such
// as a job's full log or a review with its comments.
const (
	hostTimeout     = 30 * time.Second
	slowHostTimeout = time.Minute
)

func hostContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), hostTimeout)
}

func primaryHost(repoPath string) (git.Remote, HostInfo, bool, error) {
	remotes, err := git.ListRemotes(context.Background(), repoPath)
	if err != nil || len(remotes) == 0 {
		return git.Remote{}, HostInfo{}, false, err
	}
	remote := primaryRemote(remotes)
	info, ok := hostInfoFor(remote, configOrDefault().Hosts)
	return remote, info, ok, nil
}

// enabledHost is primaryHost with ok false unless the host's integration is on.
func enabledHost(repoPath string) (git.Remote, HostInfo, bool, error) {
	remote, info, ok, err := primaryHost(repoPath)
	return remote, info, ok && info.Enabled, err
}

// requireHost is enabledHost for a call that fails, with errHostOff, when the host is off.
func requireHost(repoPath string) (git.Remote, HostInfo, error) {
	remote, info, ok, err := enabledHost(repoPath)
	if err == nil && !ok {
		err = errHostOff
	}
	return remote, info, err
}

func webRoot(info HostInfo) string {
	return "https://" + info.Hostname + "/"
}

func cachedChange(repoPath string, number int) (git.Remote, HostInfo, hostapi.Change, error) {
	remote, info, err := requireHost(repoPath)
	if err != nil {
		return remote, info, hostapi.Change{}, err
	}
	for _, kind := range []cacheKind[cachedChanges]{cacheChanges, cacheClosed} {
		if cached, _, ok := kind.get(info.Hostname, repoPath, info.Project); ok {
			for _, change := range cached.changes {
				if change.Number == number {
					return remote, info, change, nil
				}
			}
		}
	}
	return remote, info, hostapi.Change{}, fmt.Errorf("%s%d isn't in the last refresh; refresh and try again", changePrefix(info), number)
}

func inParallel(count int, request func(i int)) {
	var wg sync.WaitGroup
	slots := make(chan struct{}, ciParallel)
	for i := range count {
		wg.Add(1)
		slots <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-slots }()
			request(i)
		}()
	}
	wg.Wait()
}
