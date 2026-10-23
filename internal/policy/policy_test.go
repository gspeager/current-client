package policy

import (
	"os"
	"path/filepath"
	"testing"
)

func useFile(t *testing.T, content string) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "policy.json")
	if content != "" {
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	SetPath(p)
	t.Cleanup(func() { SetPath("") })
}

func TestNoPolicyFileChangesNothing(t *testing.T) {
	useFile(t, "")
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	if enabled, managed := Host("gitlab.example.com"); enabled || managed {
		t.Fatalf("Host = %v, %v", enabled, managed)
	}
	if minutes, managed := AutoFetchMinutes(15); minutes != 15 || managed {
		t.Fatalf("AutoFetchMinutes = %v, %v", minutes, managed)
	}
}

func TestLocksOffForcesOnAndLeavesTheRest(t *testing.T) {
	useFile(t, `{"hosts": {"gitlab.example.com": {"enabled": false}, "github.com": {"enabled": true}, "other": {}}, "autoFetch": {"enabled": true}}`)
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	cases := map[string][2]bool{"gitlab.example.com": {false, true}, "github.com": {true, true}, "other": {false, false}, "unknown": {false, false}}
	for host, want := range cases {
		if enabled, managed := Host(host); enabled != want[0] || managed != want[1] {
			t.Errorf("Host(%s) = %v, %v; want %v", host, enabled, managed, want)
		}
	}
	if minutes, managed := AutoFetchMinutes(0); minutes != 5 || !managed {
		t.Fatalf("forced on without an interval: %v, %v", minutes, managed)
	}

	useFile(t, `{"autoFetch": {"enabled": false, "intervalMinutes": 30}}`)
	Load()
	if minutes, managed := AutoFetchMinutes(15); minutes != 0 || !managed {
		t.Fatalf("locked off: %v, %v", minutes, managed)
	}
}

func TestAMalformedFileIsReportedAndIgnored(t *testing.T) {
	useFile(t, `{"hosts": [`)
	if err := Load(); err == nil {
		t.Fatal("expected an error")
	}
	if _, managed := Host("gitlab.example.com"); managed {
		t.Fatal("a file that can't be read must not lock anything")
	}
}

func TestPathIsMachineWide(t *testing.T) {
	SetPath("")
	p := Path()
	if filepath.Base(p) != "policy.json" || !filepath.IsAbs(p) {
		t.Fatalf("Path = %q", p)
	}
}
