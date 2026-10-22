package hostapi

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gspeager/current-client/core/hostapi/hostapitest"
)

func enabledOnly(names ...string) func(string) bool {
	return func(name string) bool {
		for _, n := range names {
			if n == name {
				return true
			}
		}
		return false
	}
}

// isolateGitConfig gives the test its own, empty Git configuration, so the
// machine's proxy or CA settings can't reach the test servers.
func isolateGitConfig(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, v := range []string{"GIT_SSL_CAINFO", "GIT_SSL_CAPATH", "GIT_SSL_NO_VERIFY", "HTTPS_PROXY", "HTTP_PROXY", "https_proxy", "http_proxy"} {
		t.Setenv(v, "")
	}
	return home
}

func writeGitConfig(t *testing.T, home, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRefusesHostsNotEnabledBeforeAnyConnection(t *testing.T) {
	client := New(Options{Transport: hostapitest.Forbid(t)})
	host := Host{Name: "gitlab.example.com", API: "https://gitlab.example.com/api/v4"}
	_, err := client.Get(context.Background(), host, "projects", nil)
	if !errors.Is(err, ErrNotEnabled) {
		t.Fatalf("err = %v, want ErrNotEnabled", err)
	}
	if _, err := client.GetPages(context.Background(), host, "projects", nil, func(*Response) error { return nil }); !errors.Is(err, ErrNotEnabled) {
		t.Fatalf("GetPages err = %v, want ErrNotEnabled", err)
	}
}

func TestRefusesURLsOutsideTheAPIAddress(t *testing.T) {
	client := New(Options{Enabled: enabledOnly("gitlab.example.com"), Transport: hostapitest.Forbid(t)})
	host := Host{Name: "gitlab.example.com", API: "https://gitlab.example.com/api/v4"}
	for _, raw := range []string{
		"https://evil.example.com/api/v4/projects",
		"http://gitlab.example.com/api/v4/projects",
		"https://gitlab.example.com/other/projects",
	} {
		req, _ := http.NewRequest(http.MethodGet, raw, nil)
		if _, err := client.Do(context.Background(), host, req); err == nil {
			t.Errorf("%s was allowed", raw)
		}
	}
}

func TestSendsTheTokenAndDecodesJSON(t *testing.T) {
	isolateGitConfig(t)
	var auth, accept string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, accept = r.Header.Get("Authorization"), r.Header.Get("Accept")
		w.Write([]byte(`{"iid": 7, "title": "Fix"}`))
	}))
	t.Cleanup(server.Close)
	host := Host{Name: "gitlab.example.com", API: server.URL + "/api/v4"}
	client := New(Options{Enabled: enabledOnly(host.Name), Tokens: tokenSourceFunc(func(string) string { return "t0k3n" })})

	var got struct {
		IID   int    `json:"iid"`
		Title string `json:"title"`
	}
	if err := client.GetJSON(context.Background(), host, "/projects/1/merge_requests/7", nil, &got); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if got.IID != 7 || got.Title != "Fix" || auth != "Bearer t0k3n" || accept != "application/json" {
		t.Fatalf("got %+v, Authorization %q, Accept %q", got, auth, accept)
	}
}

type tokenSourceFunc func(hostName string) string

func (f tokenSourceFunc) Token(_ context.Context, hostName string) (string, error) {
	return f(hostName), nil
}

func TestTrustsGitsCertificateAuthority(t *testing.T) {
	home := isolateGitConfig(t)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok": true}`))
	}))
	t.Cleanup(server.Close)
	caFile := filepath.Join(home, "ca.pem")
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	if err := os.WriteFile(caFile, caPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	host := Host{Name: "gitlab.example.com", API: server.URL + "/api/v4"}

	untrusting := New(Options{Enabled: enabledOnly(host.Name)})
	_, err := untrusting.Get(context.Background(), host, "projects", nil)
	var unknownCA x509.UnknownAuthorityError
	if !errors.As(err, &unknownCA) {
		t.Fatalf("without Git's CA the request should fail certificate checks, got %v", err)
	}
	if strings.Contains(err.Error(), "127.0.0.1:") && !strings.Contains(err.Error(), host.Name) {
		t.Errorf("error should name the host: %v", err)
	}

	writeGitConfig(t, home, "[http \""+server.URL+"\"]\n\tsslCAInfo = "+filepath.ToSlash(caFile)+"\n")
	trusting := New(Options{Enabled: enabledOnly(host.Name)})
	if _, err := trusting.Get(context.Background(), host, "projects", nil); err != nil {
		t.Fatalf("with http.sslCAInfo: %v", err)
	}

	caDir := filepath.Join(home, "cas")
	os.Mkdir(caDir, 0o700)
	os.WriteFile(filepath.Join(caDir, "corp.pem"), caPEM, 0o600)
	writeGitConfig(t, home, "[http]\n\tsslCAPath = "+filepath.ToSlash(caDir)+"\n")
	byPath := New(Options{Enabled: enabledOnly(host.Name)})
	if _, err := byPath.Get(context.Background(), host, "projects", nil); err != nil {
		t.Fatalf("with http.sslCAPath: %v", err)
	}
}

func TestSSLVerifyOffNeverTurnsVerificationOff(t *testing.T) {
	home := isolateGitConfig(t)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(server.Close)
	writeGitConfig(t, home, "[http \""+server.URL+"\"]\n\tsslVerify = false\n")
	host := Host{Name: "gitlab.example.com", API: server.URL + "/api/v4"}
	client := New(Options{Enabled: enabledOnly(host.Name)})
	_, err := client.Get(context.Background(), host, "projects", nil)
	var unknownCA x509.UnknownAuthorityError
	if !errors.As(err, &unknownCA) {
		t.Fatalf("sslVerify=false must not skip certificate checks, got %v", err)
	}
}

func TestUsesGitsProxy(t *testing.T) {
	home := isolateGitConfig(t)
	var proxied atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Host != "api.example.test" {
			http.Error(w, "not a proxy request for the API", http.StatusBadGateway)
			return
		}
		proxied.Add(1)
		w.Write([]byte(`[]`))
	}))
	t.Cleanup(proxy.Close)
	writeGitConfig(t, home, "[http \"http://api.example.test\"]\n\tproxy = "+proxy.URL+"\n")
	host := Host{Name: "api.example.test", API: "http://api.example.test/api/v4"}
	client := New(Options{Enabled: enabledOnly(host.Name)})
	if _, err := client.Get(context.Background(), host, "projects", nil); err != nil {
		t.Fatalf("through Git's proxy: %v", err)
	}
	if proxied.Load() != 1 {
		t.Fatalf("proxy saw %d requests", proxied.Load())
	}
}

func TestBacksOffWhenRateLimited(t *testing.T) {
	var hits atomic.Int32
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		hits.Add(1)
		return &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{"Retry-After": {"120"}}, Body: http.NoBody}, nil
	})
	host := Host{Name: "gitlab.example.com", API: "https://gitlab.example.com/api/v4"}
	client := New(Options{Enabled: enabledOnly(host.Name), Transport: transport})

	_, err := client.Get(context.Background(), host, "projects", nil)
	var limited *RateLimitError
	if !errors.As(err, &limited) || limited.Host != host.Name {
		t.Fatalf("err = %v, want RateLimitError", err)
	}
	if wait := time.Until(limited.Until); wait < 100*time.Second || wait > 120*time.Second {
		t.Fatalf("Until is %v away, want about 120s", wait)
	}
	if _, err := client.Get(context.Background(), host, "projects", nil); !errors.As(err, &limited) {
		t.Fatalf("second request err = %v, want RateLimitError", err)
	}
	if hits.Load() != 1 {
		t.Fatalf("the host was contacted %d times while asked to wait", hits.Load())
	}
	client.Forget(host.Name)
	client.Get(context.Background(), host, "projects", nil)
	if hits.Load() != 2 {
		t.Fatal("Forget should drop the pause")
	}
}

func TestGitHubStyleRateLimitUsesResetHeader(t *testing.T) {
	reset := time.Now().Add(45 * time.Second).Unix()
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusForbidden, Header: http.Header{
			"X-Ratelimit-Remaining": {"0"}, "X-Ratelimit-Reset": {itoa(reset)},
		}, Body: http.NoBody}, nil
	})
	host := Host{Name: "github.com", API: "https://api.github.com"}
	client := New(Options{Enabled: enabledOnly(host.Name), Transport: transport})
	_, err := client.Get(context.Background(), host, "repos/o/r/pulls", nil)
	var limited *RateLimitError
	if !errors.As(err, &limited) {
		t.Fatalf("err = %v, want RateLimitError", err)
	}
	if wait := time.Until(limited.Until); wait < 40*time.Second || wait > 45*time.Second {
		t.Fatalf("Until is %v away, want about 45s", wait)
	}
}

func TestAnswersOf400OrMoreAreHostErrorsWithoutTheBody(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusNotFound, Header: http.Header{}, Body: http.NoBody}, nil
	})
	host := Host{Name: "gitlab.example.com", API: "https://gitlab.example.com/api/v4"}
	client := New(Options{Enabled: enabledOnly(host.Name), Transport: transport})
	_, err := client.Get(context.Background(), host, "projects/1", nil)
	var hostErr *HostError
	if !errors.As(err, &hostErr) || hostErr.Status != 404 || err.Error() != "gitlab.example.com answered 404 Not Found" {
		t.Fatalf("err = %v", err)
	}
}

func TestPaginationFollowsNextLinksUpToTheCap(t *testing.T) {
	var served atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served.Add(1)
		page := r.URL.Query().Get("page")
		next := "2"
		if page != "" {
			next = itoa(int64(atoi(page) + 1))
		}
		w.Header().Set("Link", `<`+"http://"+r.Host+r.URL.Path+"?page="+next+`>; rel="next", <`+"http://"+r.Host+r.URL.Path+`?page=9>; rel="last"`)
		w.Write([]byte(`[` + page + `]`))
	}))
	t.Cleanup(server.Close)
	isolateGitConfig(t)
	host := Host{Name: "gitlab.example.com", API: server.URL + "/api/v4"}
	client := New(Options{Enabled: enabledOnly(host.Name), PageCap: 3})

	var pages int
	truncated, err := client.GetPages(context.Background(), host, "projects/1/merge_requests", url.Values{"state": {"opened"}}, func(*Response) error {
		pages++
		return nil
	})
	if err != nil || !truncated || pages != 3 || served.Load() != 3 {
		t.Fatalf("truncated=%v err=%v pages=%d served=%d", truncated, err, pages, served.Load())
	}
}

func TestPaginationStopsAtALinkLeavingTheAPI(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Link": {`<https://elsewhere.example.com/x?page=2>; rel="next"`}}, Body: http.NoBody}, nil
	})
	host := Host{Name: "gitlab.example.com", API: "https://gitlab.example.com/api/v4"}
	client := New(Options{Enabled: enabledOnly(host.Name), Transport: transport})
	var pages int
	truncated, err := client.GetPages(context.Background(), host, "projects", nil, func(*Response) error { pages++; return nil })
	if err != nil || truncated || pages != 1 {
		t.Fatalf("truncated=%v err=%v pages=%d", truncated, err, pages)
	}
}

func TestTimesOut(t *testing.T) {
	isolateGitConfig(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	t.Cleanup(server.Close)
	host := Host{Name: "gitlab.example.com", API: server.URL + "/api/v4"}
	client := New(Options{Enabled: enabledOnly(host.Name), Timeout: 50 * time.Millisecond})
	_, err := client.Get(context.Background(), host, "projects", nil)
	if err == nil || !strings.Contains(err.Error(), "no answer in time") {
		t.Fatalf("err = %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
