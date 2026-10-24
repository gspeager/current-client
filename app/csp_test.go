package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReleasePolicyAllowsNoInlineOrRemoteScript(t *testing.T) {
	policy := contentSecurityPolicy(false)
	for _, want := range []string{
		"default-src 'self'",
		"script-src 'self'",
		"img-src 'self' data:",
		"connect-src 'self'",
		"object-src 'none'",
		"frame-src 'none'",
	} {
		if !strings.Contains(policy, want) {
			t.Errorf("policy lacks %q: %s", want, policy)
		}
	}
	for _, directive := range strings.Split(policy, "; ") {
		if strings.HasPrefix(directive, "script-src") && strings.Contains(directive, "unsafe") {
			t.Errorf("release scripts must not allow inline or eval: %s", directive)
		}
		if strings.HasPrefix(directive, "img-src") && (strings.Contains(directive, "http") || strings.Contains(directive, "*")) {
			t.Errorf("release images must not load from the network: %s", directive)
		}
	}
	if dev := contentSecurityPolicy(true); !strings.Contains(dev, "script-src 'self' 'unsafe-inline'") || !strings.Contains(dev, "ws:") {
		t.Errorf("dev policy should allow Vite's preamble and socket: %s", dev)
	}
}

func TestCSPMiddlewareSetsTheHeaderOnEveryResponse(t *testing.T) {
	handler := CSPMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<!doctype html>"))
	}))
	for _, path := range []string{"/", "/index.html", "/assets/app.js"} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if got := recorder.Header().Get("Content-Security-Policy"); !strings.HasPrefix(got, "default-src 'self'") {
			t.Errorf("%s: Content-Security-Policy = %q", path, got)
		}
	}
}
