//go:build !nohost

package app

import (
	"os"
	"testing"

	"github.com/gspeager/current-client/core/hostapi/hostapitest"
	"github.com/gspeager/current-client/internal/config"
	"github.com/gspeager/current-client/internal/policy"
)

func installPolicy(t *testing.T, content string) {
	t.Helper()
	if err := os.WriteFile(policy.Path(), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := policy.Load(); err != nil {
		t.Fatal(err)
	}
}

func TestManagedPolicyLocksAHostOffAndForcesAnotherOn(t *testing.T) {
	useTempSettings(t)
	swapHostClient(t, hostapitest.Forbid(t))
	if err := updateConfig(func(cfg *config.Config) {
		cfg.Hosts = map[string]config.HostConfig{"gitlab.example.com": {Enabled: true}}
	}); err != nil {
		t.Fatal(err)
	}
	var s HostService
	if !hostEnabled("gitlab.example.com") {
		t.Fatal("the user's switch should count without a policy")
	}

	installPolicy(t, `{"hosts": {"gitlab.example.com": {"enabled": false}, "github.com": {"enabled": true}}}`)

	if hostEnabled("gitlab.example.com") || !hostEnabled("github.com") {
		t.Fatal("the client must follow the policy, not the user's settings")
	}
	hosts, err := s.List(nil)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]HostInfo{}
	for _, h := range hosts {
		seen[h.Hostname] = h
	}
	if locked := seen["gitlab.example.com"]; locked.Enabled || !locked.Managed {
		t.Fatalf("locked host = %+v", locked)
	}
	if err := s.SetEnabled("gitlab.example.com", true); err == nil {
		t.Fatal("a managed switch must refuse")
	}
	if _, err := s.TestConnection("gitlab.example.com"); err == nil {
		t.Fatal("a locked host must not be contacted")
	}
}

func TestManagedPolicyFixesBackgroundFetch(t *testing.T) {
	useTempSettings(t)
	var s SettingsService
	if err := s.SetAutoFetchIntervalMinutes(15); err != nil {
		t.Fatal(err)
	}
	installPolicy(t, `{"autoFetch": {"enabled": false}}`)

	settings, err := s.GetSettings()
	if err != nil || settings.AutoFetchIntervalMinutes != 0 || !settings.AutoFetchManaged {
		t.Fatalf("settings = %+v, %v", settings, err)
	}
	if err := s.SetAutoFetchIntervalMinutes(5); err == nil {
		t.Fatal("a managed interval must refuse")
	}
}
