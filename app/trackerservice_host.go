//go:build !nohost

package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gspeager/current-client/core/hostapi/jira"
	"github.com/gspeager/current-client/core/hostlink"
	"github.com/gspeager/current-client/internal/config"
	"github.com/gspeager/current-client/internal/policy"
)

// SetEnabled is the one switch that lets the app contact a tracker. Turning
// it off drops everything held from it.
func (s *TrackerService) SetEnabled(site string, enabled bool) error {
	tracker, err := knownTracker(site)
	if err != nil {
		return err
	}
	name := jira.HostName(tracker)
	if _, managed := policy.Host(name); managed {
		return fmt.Errorf("integration for %s is managed by your organization", name)
	}
	if !enabled {
		hostClient.Forget(name)
	}
	return setTrackerAccess(site, func(a *config.TrackerAccess) { a.Enabled = enabled })
}

// SaveToken hands the token to Git's credential helper under the app's own
// key. Jira Cloud signs in with the account's email and an API token, Data
// Center with a personal access token alone.
func (s *TrackerService) SaveToken(site, email, token string) error {
	tracker, err := knownTracker(site)
	if err != nil {
		return err
	}
	secret := strings.TrimSpace(token)
	if tracker.Kind == hostlink.JiraCloud {
		email = strings.TrimSpace(email)
		if email == "" || strings.ContainsAny(email, ":\n\r") {
			return errors.New("Jira Cloud needs the account's email beside the API token")
		}
		secret = email + ":" + secret
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := trackerCredentials.Save(ctx, jira.HostName(tracker), secret); err != nil {
		return err
	}
	return setTrackerAccess(site, func(a *config.TrackerAccess) { a.HasToken = true })
}

func (s *TrackerService) RemoveToken(site string) error {
	tracker, err := knownTracker(site)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := trackerCredentials.Remove(ctx, jira.HostName(tracker)); err != nil {
		return err
	}
	return setTrackerAccess(site, func(a *config.TrackerAccess) { a.HasToken = false })
}

// TestConnection is the one request Settings makes, and only while the
// tracker is on.
func (s *TrackerService) TestConnection(site string) (string, error) {
	tracker, err := knownTracker(site)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	me, err := jira.New(hostClient, tracker).CurrentUser(ctx)
	if err != nil {
		return "", err
	}
	return "Signed in to " + jira.HostName(tracker) + " as " + me.Name + ".", nil
}

func knownTracker(site string) (hostlink.Tracker, error) {
	for _, t := range allTrackers() {
		if t.Site == site {
			return t, nil
		}
	}
	return hostlink.Tracker{}, errors.New(site + " isn't listed")
}

func setTrackerAccess(site string, change func(*config.TrackerAccess)) error {
	return updateConfig(func(cfg *config.Config) {
		if cfg.TrackerAccess == nil {
			cfg.TrackerAccess = map[string]config.TrackerAccess{}
		}
		access := cfg.TrackerAccess[site]
		change(&access)
		cfg.TrackerAccess[site] = access
	})
}

// forgetTracker drops what the client and Git's helper hold for a tracker
// being removed.
func forgetTracker(site string, hasToken bool) {
	name := trackerHostName(site)
	hostClient.Forget(name)
	if hasToken {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		trackerCredentials.Remove(ctx, name)
	}
}
