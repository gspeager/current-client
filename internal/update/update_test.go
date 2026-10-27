package update

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func releases(t *testing.T, latest string, handler http.HandlerFunc) *Checker {
	t.Helper()
	mux := http.NewServeMux()
	if handler != nil {
		mux.HandleFunc("/releases/latest", handler)
	} else {
		mux.HandleFunc("/releases/latest", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/gspeager/current-client/releases/tag/v"+latest, http.StatusFound)
		})
	}
	mux.HandleFunc("/releases.atom", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<?xml version="1.0"?><feed xmlns="http://www.w3.org/2005/Atom">
			<entry><id>tag:github.com,2008:Repository/1/v9.9.9-rc.1</id><updated>2026-11-01T10:00:00Z</updated></entry>
			<entry><id>tag:github.com,2008:Repository/1/v` + latest + `</id><updated>2026-10-24T18:30:00Z</updated></entry></feed>`))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return &Checker{ReleasesURL: server.URL + "/releases", Client: &http.Client{Timeout: 2 * time.Second}}
}

func TestCheckComparesWithTheLatestPublishedRelease(t *testing.T) {
	c := releases(t, "0.4.0", nil)
	for current, want := range map[string]string{"0.4.0": UpToDate, "0.3.2": Newer, "0.5.0-rc.1": Ahead, "0.4.0-beta.2": Newer, "dev": DevBuild} {
		result, err := c.Check(context.Background(), current)
		if err != nil || result.State != want || result.Latest != "0.4.0" || result.ReleasedAt != "2026-10-24" {
			t.Errorf("%s: result = %+v, %v, want %s", current, result, err, want)
		}
		if result.LatestURL != c.ReleasesURL+"/tag/v0.4.0" {
			t.Errorf("latest url = %s", result.LatestURL)
		}
	}
}

func TestCheckFailsInPlainWords(t *testing.T) {
	// A server that was closed: nothing answers at that address any more.
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	unreachable := &Checker{ReleasesURL: srv.URL + "/releases", Client: &http.Client{Timeout: time.Second}}
	if _, err := unreachable.Check(context.Background(), "0.4.0"); !errors.Is(err, ErrUnreachable) {
		t.Errorf("closed server: %v, want ErrUnreachable", err)
	}

	slow := releases(t, "0.4.0", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		http.Redirect(w, r, "/releases/tag/v0.4.0", http.StatusFound)
	})
	slow.Client = &http.Client{Timeout: 50 * time.Millisecond}
	if _, err := slow.Check(context.Background(), "0.4.0"); !errors.Is(err, ErrTimeout) {
		t.Errorf("slow server: %v, want ErrTimeout", err)
	}

	plain := releases(t, "0.4.0", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("<html>")) })
	if _, err := plain.Check(context.Background(), "0.4.0"); !errors.Is(err, ErrUnreadable) {
		t.Errorf("no redirect: %v, want ErrUnreadable", err)
	}
	odd := releases(t, "0.4.0", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/releases/tag/latest-build", http.StatusFound)
	})
	if _, err := odd.Check(context.Background(), "0.4.0"); !errors.Is(err, ErrUnreadable) {
		t.Errorf("unparsable location: %v, want ErrUnreadable", err)
	}

	tlsServer := httptest.NewTLSServer(http.NotFoundHandler())
	t.Cleanup(tlsServer.Close)
	blocked := &Checker{ReleasesURL: tlsServer.URL + "/releases", Client: &http.Client{Timeout: time.Second}}
	if _, err := blocked.Check(context.Background(), "0.4.0"); !errors.Is(err, ErrBlocked) {
		t.Errorf("unverifiable certificate: %v, want ErrBlocked", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := unreachable.Check(ctx, "0.4.0"); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v, want context.Canceled", err)
	}
}

func TestCompareFollowsSemver(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0}, {"1.0.0", "1.0.1", -1}, {"1.1.0", "1.0.9", 1}, {"2.0.0", "1.9.9", 1},
		{"1.0.0-rc.1", "1.0.0", -1}, {"1.0.0", "1.0.0-rc.1", 1}, {"1.0.0-alpha", "1.0.0-beta", -1},
	} {
		a, _ := parseVersion(tc.a)
		b, _ := parseVersion(tc.b)
		if got := compare(a, b); got != tc.want {
			t.Errorf("compare(%s, %s) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}
