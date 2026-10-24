//go:build !nohost

package app

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostapi/hostapitest"
	"github.com/gspeager/current-client/internal/config"
)

func TestHostClientSendsOnlyWhileSettingsEnableTheHost(t *testing.T) {
	useTempSettings(t)
	host := hostapi.Host{Name: "gitlab.example.com", API: "https://gitlab.example.com/api/v4"}

	off := newHostClient(hostapitest.Forbid(t))
	if _, err := off.Get(context.Background(), host, "projects", nil); !errors.Is(err, hostapi.ErrNotEnabled) {
		t.Fatalf("with nothing enabled: %v", err)
	}

	var sent atomic.Int32
	on := newHostClient(countingTransport{&sent})
	setHostEnabled(t, host.Name, true)
	if _, err := on.Get(context.Background(), host, "projects", nil); err != nil {
		t.Fatalf("enabled: %v", err)
	}
	setHostEnabled(t, host.Name, false)
	if _, err := on.Get(context.Background(), host, "projects", nil); !errors.Is(err, hostapi.ErrNotEnabled) {
		t.Fatalf("after disabling: %v", err)
	}
	if sent.Load() != 1 {
		t.Fatalf("%d requests sent, want 1", sent.Load())
	}
}

func setHostEnabled(t *testing.T, hostName string, enabled bool) {
	t.Helper()
	err := updateConfig(func(cfg *config.Config) {
		if cfg.Hosts == nil {
			cfg.Hosts = map[string]config.HostConfig{}
		}
		host := cfg.Hosts[hostName]
		host.Enabled = enabled
		cfg.Hosts[hostName] = host
	})
	if err != nil {
		t.Fatal(err)
	}
}

type countingTransport struct{ sent *atomic.Int32 }

func (c countingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.sent.Add(1)
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: http.NoBody}, nil
}
