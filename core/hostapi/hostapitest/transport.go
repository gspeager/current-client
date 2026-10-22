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
