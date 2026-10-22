// Package hostapitest is for tests of code that must make no request.
package hostapitest

import (
	"net/http"
	"testing"
)

// Forbid is a transport that fails the test on any request.
func Forbid(t testing.TB) http.RoundTripper {
	return forbid{t}
}

type forbid struct{ t testing.TB }

func (f forbid) RoundTrip(req *http.Request) (*http.Response, error) {
	f.t.Helper()
	f.t.Fatalf("a request was made to %s", req.URL.Redacted())
	return nil, nil
}

// IsolateGitConfig gives a test an empty Git configuration and no proxy or
// certificate overrides, so the machine's settings can't reach its servers.
func IsolateGitConfig(t testing.TB) (home string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, v := range []string{"GIT_SSL_CAINFO", "GIT_SSL_CAPATH", "GIT_SSL_NO_VERIFY", "HTTPS_PROXY", "HTTP_PROXY", "https_proxy", "http_proxy"} {
		t.Setenv(v, "")
	}
	return home
}

// AllowAll enables every host, for tests of what a provider sends.
func AllowAll(string) bool { return true }
