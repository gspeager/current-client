package standin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/gspeager/current-client/app"
	"github.com/gspeager/current-client/core/hostapi"
)

func settingsIn(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, v := range []string{"HOME", "XDG_CONFIG_HOME", "APPDATA"} {
		t.Setenv(v, dir)
	}
	app.SetSettingsFolder("current-client-embed-example-test")
	return dir
}

func TestTheHostAppsRequestGoesThroughTheSharedClient(t *testing.T) {
	dir := settingsIn(t)
	if err := app.UsePolicyFile(filepath.Join(dir, "none.json")); err != nil {
		t.Fatal(err)
	}
	asked := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked++
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()
	host := hostapi.Host{Name: "current.example", API: server.URL + "/api"}

	status, err := TurnOnAndCheck(context.Background(), host)
	if err != nil || !status.OK || asked != 1 {
		t.Fatalf("TurnOnAndCheck = %+v, %v, asked %d", status, err, asked)
	}
	if enabled, _ := app.HostEnabled(host.Name); !enabled {
		t.Fatal("the host should be on in Current Client's settings too")
	}
}

func TestPolicyLockOffRefusesIt(t *testing.T) {
	dir := settingsIn(t)
	path := filepath.Join(dir, "policy.json")
	os.WriteFile(path, []byte(`{"hosts": {"current.example": {"enabled": false}}}`), 0o644)
	if err := app.UsePolicyFile(path); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("a locked host was asked: %s", r.URL)
	}))
	defer server.Close()
	if _, err := TurnOnAndCheck(context.Background(), hostapi.Host{Name: "current.example", API: server.URL + "/api"}); err == nil {
		t.Fatal("a host the policy locks off was turned on")
	}
}

func TestAnExampleProjectOnTheStandInHost(t *testing.T) {
	dir := settingsIn(t)
	if err := app.UsePolicyFile(filepath.Join(dir, "none.json")); err != nil {
		t.Fatal(err)
	}
	remote := filepath.Join(dir, ExampleRemote)
	repo := filepath.Join(dir, "app")
	for _, args := range [][]string{
		{"init", "-q", "--bare", remote},
		{"init", "-q", "-b", "main", repo},
		{"-C", repo, "-c", "user.name=E", "-c", "user.email=e@example.com", "commit", "-q", "--allow-empty", "-m", "first"},
		{"-C", repo, "remote", "add", "origin", remote},
		{"-C", repo, "push", "-q", "origin", "main", "main:feature"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	RegisterExampleHost()
	other := filepath.Join(dir, "elsewhere")
	for _, args := range [][]string{
		{"init", "-q", other},
		{"-C", other, "remote", "add", "origin", "git@github.com:team/" + ExampleRemote},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	if found, _ := (&app.HostService{}).Detect(other); len(found) != 1 || found[0].Hostname != "github.com" {
		t.Fatalf("a real host's remote of the same name: %+v", found)
	}
	if err := app.SetHostEnabled(ExampleHost, true); err != nil {
		t.Fatal(err)
	}
	hosts := &app.HostService{}
	if err := hosts.Refresh(context.Background(), repo); err != nil {
		t.Fatal(err)
	}
	list, err := hosts.Changes(repo)
	if err != nil || !list.Enabled || list.Host != ExampleHost || len(list.Changes) != 1 || list.Changes[0].Title != exampleChange.Title {
		t.Fatalf("the example's pull requests: %+v, %v", list, err)
	}
}
