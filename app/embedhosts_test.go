//go:build !nohost

package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"testing"

	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/internal/policy"
)

type standIn struct {
	hostapi.Provider
	host hostapi.Host
}

func TestEmbeddingAppsShareTheClientAndItsLock(t *testing.T) {
	useTempSettings(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()
	host := hostapi.Host{Name: "current.example", API: server.URL + "/api"}
	RegisterProvider(host.Name, func(client *hostapi.Client, h hostapi.Host) hostapi.Provider {
		return standIn{host: h}
	})
	t.Cleanup(func() { delete(extraProviders, host.Name) })
	if p, ok := providerFor(HostInfo{Hostname: host.Name, APIURL: host.API}).(standIn); !ok || p.host != host {
		t.Fatalf("providerFor = %#v", p)
	}

	var changed []string
	stop := OnHostsChanged(func(name string) { changed = append(changed, name) })
	defer stop()
	ctx := context.Background()
	var out map[string]bool
	if err := HostClient().GetJSON(ctx, host, "check", nil, &out); !errors.Is(err, hostapi.ErrNotEnabled) {
		t.Fatalf("before turning it on: %v", err)
	}
	if err := SetHostEnabled(host.Name, true); err != nil {
		t.Fatal(err)
	}
	if enabled, managed := HostEnabled(host.Name); !enabled || managed || !slices.Contains(EnabledHosts(), host.Name) {
		t.Fatalf("HostEnabled = %v, %v; EnabledHosts = %v", enabled, managed, EnabledHosts())
	}
	if err := HostClient().GetJSON(ctx, host, "check", nil, &out); err != nil || !out["ok"] {
		t.Fatalf("through the shared client: %v, %v", out, err)
	}
	if !slices.Equal(changed, []string{host.Name}) {
		t.Fatalf("watchers told = %v", changed)
	}

	os.WriteFile(policy.Path(), []byte(`{"hosts": {"current.example": {"enabled": false}}}`), 0o644)
	if err := UsePolicyFile(policy.Path()); err != nil {
		t.Fatal(err)
	}
	if err := SetHostEnabled(host.Name, true); err == nil {
		t.Fatal("a host the policy locks off was turned on")
	}
	if enabled, managed := HostEnabled(host.Name); enabled || !managed || slices.Contains(EnabledHosts(), host.Name) {
		t.Fatalf("locked: HostEnabled = %v, %v", enabled, managed)
	}
	if err := HostClient().GetJSON(ctx, host, "check", nil, &out); !errors.Is(err, hostapi.ErrNotEnabled) {
		t.Fatalf("a locked host was asked: %v", err)
	}
	stop()
	SetHostEnabled("other.example", true)
	if len(changed) != 1 {
		t.Fatalf("a stopped watcher was told: %v", changed)
	}
}
