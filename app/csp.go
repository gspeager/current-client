package app

import (
	"net/http"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// contentSecurityPolicy is what the webview may load. Text from a Git host
// (merge request descriptions, review comments) is rendered inside this page
// next to bindings that run Git, so scripts, inline handlers and remote
// images are stopped by the browser, not by a renderer being careful.
// Styles allow inline because Radix injects a style element for scroll
// locking; that permits CSS, never script. A dev build also allows Vite's
// inline React Refresh preamble and its hot-reload socket.
func contentSecurityPolicy(dev bool) string {
	script := "'self'"
	connect := "'self'"
	if dev {
		script += " 'unsafe-inline'"
		connect += " ws: http://localhost:* http://127.0.0.1:*"
	}
	return strings.Join([]string{
		"default-src 'self'",
		"script-src " + script,
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data:",
		"font-src 'self'",
		"connect-src " + connect,
		"object-src 'none'",
		"frame-src 'none'",
		"base-uri 'none'",
		"form-action 'none'",
	}, "; ")
}

// CSPMiddleware sets the Content-Security-Policy on everything the asset
// server sends. A host app that embeds Current Client should add it to its
// own AssetOptions.
func CSPMiddleware() application.Middleware {
	policy := contentSecurityPolicy(devBuild)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Security-Policy", policy)
			next.ServeHTTP(w, r)
		})
	}
}
