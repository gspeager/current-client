package platform

import (
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

type RecentError struct {
	At      string
	Message string
}

type DiagnosticsInput struct {
	AppVersion      string
	GitVersion      string
	GitPath         string
	GitProblem      string
	SettingsVersion int
	RecentErrors    []RecentError
	HomeDir         string
	// RepoPaths are replaced wherever they appear, since error messages can
	// quote a path.
	RepoPaths []string
	// HostIntegration is "not built in", "off" or "on for <hosts>".
	HostIntegration string
	// Hostnames are the Git hosts the app knows; with OmitHostnames they are
	// replaced wherever they appear.
	Hostnames     []string
	OmitHostnames bool
}

// Tokens and host answers never belong in a report, however they got into
// an error message: known token shapes, bearer and private-token headers,
// anything after "token=", JSON objects and HTML pages.
var (
	tokenPattern    = regexp.MustCompile(`(?i)glpat-[A-Za-z0-9_-]{8,}|gh[pousr]_[A-Za-z0-9]{16,}|github_pat_[A-Za-z0-9_]{16,}|bearer\s+\S+|private-token:\s*\S+|token[=:]\s*\S+`)
	responsePattern = regexp.MustCompile(`(?is)\{[^{}]*"[^{}]*\}|<(?:!doctype|html)\b[^\n]*`)
)

// Diagnostics formats a report to paste into a bug report. It contains no
// repository paths, and the home folder is shown as "~".
func Diagnostics(in DiagnosticsInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "App version: %s\n", in.AppVersion)
	fmt.Fprintf(&b, "OS: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	switch {
	case in.GitProblem != "":
		fmt.Fprintf(&b, "Git: %s\n", in.GitProblem)
	default:
		fmt.Fprintf(&b, "Git: %s at %s\n", in.GitVersion, in.GitPath)
	}
	fmt.Fprintf(&b, "Settings format: %d\n", in.SettingsVersion)
	if in.HostIntegration != "" {
		fmt.Fprintf(&b, "Host integration: %s\n", in.HostIntegration)
	}
	b.WriteString("Recent errors:\n")
	if len(in.RecentErrors) == 0 {
		b.WriteString("  none\n")
	}
	for _, e := range in.RecentErrors {
		fmt.Fprintf(&b, "  %s  %s\n", e.At, e.Message)
	}
	text := redact(b.String(), in.HomeDir, in.RepoPaths)
	text = tokenPattern.ReplaceAllString(text, "<token>")
	text = responsePattern.ReplaceAllString(text, "<response omitted>")
	if in.OmitHostnames {
		text = replaceAll(text, in.Hostnames, "<host>")
	}
	return text
}

func redact(text, homeDir string, repoPaths []string) string {
	for _, p := range longestFirst(repoPaths) {
		text = replacePath(text, p, "<repository>")
	}
	return replacePath(text, homeDir, "~")
}

func replaceAll(text string, values []string, with string) string {
	for _, v := range longestFirst(values) {
		if v != "" {
			text = strings.ReplaceAll(text, v, with)
		}
	}
	return text
}

// longestFirst keeps a value inside another (a repository in a repository,
// api.<host> in <host>) from being replaced piecemeal.
func longestFirst(values []string) []string {
	sorted := append([]string(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return len(sorted[i]) > len(sorted[j]) })
	return sorted
}

// replacePath also catches the forward-slash spelling git uses on Windows.
func replacePath(text, path, with string) string {
	if path == "" {
		return text
	}
	text = strings.ReplaceAll(text, path, with)
	return strings.ReplaceAll(text, filepath.ToSlash(path), with)
}
