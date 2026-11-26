//go:build nohost

package app

import (
	"context"
	"time"

	"github.com/gspeager/current-client/core/gitexec"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Built with -tags nohost: no host API client, no HostService, no
// Integrations settings. For deployments that must not have the code at all.
const hostIntegrationBuilt = false

func hostServices() []application.Service { return nil }

func afterFetch(context.Context, string) {}

func mergedOnHost(string) map[string]hostMergedChange { return nil }

func openChangesByBranch(string) map[string]string { return nil }

func protectionOf(string, string, string) (*PushWarning, error) { return nil, nil }

func trackerReading(string) bool { return false }

func forgetTracker(string, bool) {}

func hostInboxRows(string) ([]InboxRow, bool, *time.Time, string) { return nil, false, nil, "" }

func markTodoDone(string, string) error { return &gitexec.AppError{Message: "Host integration is off."} }

func standupHost(*StandupRepo, time.Time) {}

func releaseHostChecks(string, string, string) []ReleaseCheck { return nil }
