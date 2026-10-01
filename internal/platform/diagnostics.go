package platform

import (
	"fmt"
	"path/filepath"
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
}

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
	b.WriteString("Recent errors:\n")
	if len(in.RecentErrors) == 0 {
		b.WriteString("  none\n")
	}
	for _, e := range in.RecentErrors {
		fmt.Fprintf(&b, "  %s  %s\n", e.At, e.Message)
	}
	return redact(b.String(), in.HomeDir, in.RepoPaths)
}

func redact(text, homeDir string, repoPaths []string) string {
	// Longest first, so a repository inside another is replaced whole.
	paths := append([]string(nil), repoPaths...)
	sort.Slice(paths, func(i, j int) bool { return len(paths[i]) > len(paths[j]) })
	for _, p := range paths {
		text = replacePath(text, p, "<repository>")
	}
	return replacePath(text, homeDir, "~")
}

// replacePath also catches the forward-slash spelling git uses on Windows.
func replacePath(text, path, with string) string {
	if path == "" {
		return text
	}
	text = strings.ReplaceAll(text, path, with)
	return strings.ReplaceAll(text, filepath.ToSlash(path), with)
}
