//go:build !nohost

package app

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/internal/gittest"
)

// A stand-in host with one made-up pull request, counting what it's asked.
type madeUpHost struct {
	hostapi.Provider
	asked *int
}

func (h madeUpHost) OpenChanges(context.Context, string) ([]hostapi.Change, bool, error) {
	*h.asked++
	return []hostapi.Change{{Number: 7, Title: "Try the stand-in", State: hostapi.ChangeOpen, SourceBranch: "feat", TargetBranch: "main"}}, false, nil
}

func (h madeUpHost) ClosedChanges(context.Context, string) ([]hostapi.Change, bool, error) {
	*h.asked++
	return nil, false, nil
}

func (h madeUpHost) Reviews(context.Context, string, hostapi.Change) ([]hostapi.Review, error) {
	return nil, nil
}

func (h madeUpHost) LatestPipeline(context.Context, string, string) (hostapi.Pipeline, bool, error) {
	return hostapi.Pipeline{}, false, nil
}

func (h madeUpHost) CurrentUser(context.Context) (hostapi.User, error) {
	return hostapi.User{Username: "me"}, nil
}

func (h madeUpHost) BranchRules(context.Context, string, []string) ([]hostapi.BranchRule, error) {
	return nil, nil
}

func TestARemoteInAFolderOnAStandInHost(t *testing.T) {
	useTempSettings(t)
	remote := filepath.Join(t.TempDir(), "example.git")
	gittest.Run(t, filepath.Dir(remote), "init", "-q", "--bare", remote)
	repo := notesRepo(t)
	gittest.Run(t, repo, "remote", "add", "origin", remote)
	gittest.Run(t, repo, "push", "-q", "origin", "main", "side:feat")

	asked := 0
	RegisterProvider("current.example", func(*hostapi.Client, hostapi.Host) hostapi.Provider {
		return madeUpHost{asked: &asked}
	})
	RegisterRemoteHost(func(url string) (string, string, bool) {
		return "current.example", "examples/app", url == remote
	})
	t.Cleanup(func() {
		delete(extraProviders, "current.example")
		remoteHosts.claims = nil
	})
	service := &HostService{}

	if err := service.Refresh(context.Background(), repo); err != nil {
		t.Fatal(err)
	}
	off, err := service.Changes(repo)
	if err != nil {
		t.Fatal(err)
	}
	if off.Enabled || len(off.Changes) != 0 || asked != 0 {
		t.Fatalf("with the host off: %+v, asked %d", off, asked)
	}
	detected, err := service.Detect(repo)
	if err != nil || len(detected) != 1 || detected[0].Hostname != "current.example" || detected[0].Project != "examples/app" || detected[0].Enabled {
		t.Fatalf("the folder's host: %+v, %v", detected, err)
	}

	if err := SetHostEnabled("current.example", true); err != nil {
		t.Fatal(err)
	}
	if err := service.Refresh(context.Background(), repo); err != nil {
		t.Fatal(err)
	}
	on, err := service.Changes(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !on.Enabled || len(on.Changes) != 1 || on.Changes[0].Number != 7 || asked == 0 {
		t.Fatalf("with the host on: %+v, asked %d", on, asked)
	}
	branches, err := service.BranchChanges(repo)
	if err != nil {
		t.Fatal(err)
	}
	if branches["origin/feat"].Number != 7 {
		t.Fatalf("the branch's change: %+v", branches)
	}
}
