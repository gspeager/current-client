package standin

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/gspeager/current-client/app"
	"github.com/gspeager/current-client/core/hostapi"
)

// The example project's remote is a folder on this computer named this, which
// Current Client takes as being on a stand-in host with made-up pull requests,
// so the host features show with no real host and no network.
const (
	ExampleHost    = "current.example"
	ExampleProject = "examples/app"
	ExampleRemote  = "example.git"
)

var errMadeUp = errors.New("the stand-in host only lists its made-up pull requests")

// RegisterExampleHost names the stand-in host for a remote folder called
// ExampleRemote and registers its provider. The host is off until turned on.
func RegisterExampleHost() {
	app.RegisterRemoteHost(func(remoteURL string) (string, string, bool) {
		// A path or file:// address; git@host:example.git is a real host's.
		local := strings.HasPrefix(remoteURL, "file://") || filepath.IsAbs(remoteURL) || strings.HasPrefix(remoteURL, ".")
		return ExampleHost, ExampleProject, local && filepath.Base(strings.TrimSuffix(remoteURL, "/")) == ExampleRemote
	})
	app.RegisterProvider(ExampleHost, func(*hostapi.Client, hostapi.Host) hostapi.Provider { return madeUp{} })
}

var exampleChange = hostapi.Change{
	Number:       1,
	Title:        "Try the stand-in host",
	Description:  "A made-up pull request, from the example's stand-in host.",
	State:        hostapi.ChangeOpen,
	SourceBranch: "feature",
	TargetBranch: "main",
	Author:       hostapi.User{Username: "example", Name: "Example"},
}

// madeUp answers from what it makes up, and refuses what would change anything.
type madeUp struct{}

func (madeUp) OpenChanges(context.Context, string) ([]hostapi.Change, bool, error) {
	return []hostapi.Change{exampleChange}, false, nil
}
func (madeUp) ClosedChanges(context.Context, string) ([]hostapi.Change, bool, error) {
	return nil, false, nil
}
func (madeUp) Change(_ context.Context, _ string, number int) (hostapi.Change, error) {
	if number != exampleChange.Number {
		return hostapi.Change{}, errMadeUp
	}
	return exampleChange, nil
}
func (madeUp) ChangeFiles(context.Context, string, int) ([]hostapi.ChangeFile, error) {
	return nil, nil
}
func (madeUp) ChangeCommits(context.Context, string, int) ([]hostapi.ChangeCommit, error) {
	return nil, nil
}
func (madeUp) Threads(context.Context, string, int, string) ([]hostapi.Thread, error) {
	return nil, nil
}
func (madeUp) Reviews(context.Context, string, hostapi.Change) ([]hostapi.Review, error) {
	return nil, nil
}
func (madeUp) LatestPipeline(context.Context, string, string) (hostapi.Pipeline, bool, error) {
	return hostapi.Pipeline{}, false, nil
}
func (madeUp) Jobs(context.Context, string, int64) ([]hostapi.Job, error) { return nil, nil }
func (madeUp) JobLog(context.Context, string, int64, int) (string, bool, error) {
	return "", false, errMadeUp
}
func (madeUp) Issue(context.Context, string, int) (hostapi.Issue, error) {
	return hostapi.Issue{}, errMadeUp
}
func (madeUp) AssignedIssues(context.Context, string, string) ([]hostapi.Issue, error) {
	return nil, nil
}
func (madeUp) CreateChange(context.Context, string, hostapi.NewChange) (hostapi.Change, error) {
	return hostapi.Change{}, errMadeUp
}
func (madeUp) BranchRules(context.Context, string, []string) ([]hostapi.BranchRule, error) {
	return nil, nil
}
func (madeUp) CurrentUser(context.Context) (hostapi.User, error) {
	return hostapi.User{Username: "example", Name: "Example"}, nil
}
