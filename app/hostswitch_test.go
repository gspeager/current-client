//go:build !nohost

package app

import (
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
)

func swapHostClient(t *testing.T, transport hostapiTransport) {
	t.Helper()
	previous := hostClient
	hostClient = newHostClient(transport)
	t.Cleanup(func() { hostClient = previous })
}

type hostapiTransport = http.RoundTripper

type jsonTransport struct {
	sent *atomic.Int32
	body string
}

func (j jsonTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	j.sent.Add(1)
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(j.body))}, nil
}

func TestSetEnabledIsTheOnlyThingThatLetsARequestThrough(t *testing.T) {
	useTempSettings(t)
	var sent atomic.Int32
	swapHostClient(t, jsonTransport{sent: &sent, body: `{"username": "ada"}`})
	var s HostService
	host := "gitlab.example.com"

	if _, err := s.TestConnection(host); err == nil || !strings.Contains(err.Error(), "is off") {
		t.Fatalf("TestConnection while off: %v", err)
	}
	if err := s.Dismiss(host); err != nil {
		t.Fatal(err)
	}
	if err := s.SetEnabled(host, true); err != nil {
		t.Fatal(err)
	}
	if cfg := configOrDefault().Hosts[host]; !cfg.Enabled || cfg.Dismissed {
		t.Fatalf("after enabling: %+v", cfg)
	}
	if msg, err := s.TestConnection(host); err != nil || msg != "Connected to https://gitlab.example.com/api/v4 as ada." {
		t.Fatalf("TestConnection while on: %q, %v", msg, err)
	}
	if status := s.Status(host); status.LastRequest == nil || status.InFlight {
		t.Fatalf("Status after a request: %+v", status)
	}
	if err := s.SetEnabled(host, false); err != nil {
		t.Fatal(err)
	}
	if _, err := s.TestConnection(host); err == nil {
		t.Fatal("TestConnection after disabling should be refused")
	}
	if status := s.Status(host); status.LastRequest != nil {
		t.Fatalf("turning off should drop the status: %+v", status)
	}
	if sent.Load() != 1 {
		t.Fatalf("%d requests sent, want 1", sent.Load())
	}
}

func TestListShowsRemotesFirstThenHostsAddedByHand(t *testing.T) {
	useTempSettings(t)
	var s HostService
	if err := s.Add(" GitLab.internal.example ", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.Add("bad host/name", ""); err == nil {
		t.Fatal("a hostname with a slash was accepted")
	}
	repo := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"remote", "add", "origin", "git@github.com:acme/app.git"}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	infos, err := s.List([]string{repo})
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 2 || infos[0].Hostname != "github.com" || infos[0].Remote != "origin" || infos[0].Project != "acme/app" {
		t.Fatalf("List = %+v", infos)
	}
	added := infos[1]
	if added.Hostname != "gitlab.internal.example" || added.Provider != "gitlab" || added.Remote != "" || added.APIURL != "https://gitlab.internal.example/api/v4" {
		t.Fatalf("added host = %+v", added)
	}

	if err := s.Remove("gitlab.internal.example"); err != nil {
		t.Fatal(err)
	}
	if infos, _ := s.List(nil); len(infos) != 0 {
		t.Fatalf("after Remove: %+v", infos)
	}
}
