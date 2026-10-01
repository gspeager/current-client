package gitexec

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type Info struct {
	Path    string
	Version string
}

// Detect resolves "git" on PATH when override is empty.
func Detect(ctx context.Context, override string) (Info, error) {
	path := override
	if path == "" {
		found, err := exec.LookPath("git")
		if err != nil {
			return Info{}, err
		}
		path = found
	}
	result, err := NewExecutor(path).Run(ctx, Command{Args: []string{"--version"}})
	if err != nil {
		return Info{}, err
	}
	if result.ExitCode != 0 {
		return Info{}, fmt.Errorf("git --version exited %d: %s", result.ExitCode, strings.TrimSpace(result.Stderr))
	}
	version, err := parseVersion(result.Stdout)
	if err != nil {
		return Info{}, err
	}
	return Info{Path: path, Version: version}, nil
}

func parseVersion(output string) (string, error) {
	const prefix = "git version "
	trimmed := strings.TrimSpace(output)
	if !strings.HasPrefix(trimmed, prefix) {
		return "", fmt.Errorf("unexpected git --version output: %q", trimmed)
	}
	return strings.TrimPrefix(trimmed, prefix), nil
}

// MinimumVersion is the oldest git that has every command the app runs;
// `git restore` (2.23) is the newest of them.
const MinimumVersion = "2.23"

// DetectSupported is Detect plus the minimum version check, with both
// failures worded for display.
func DetectSupported(ctx context.Context, override string) (Info, error) {
	info, err := Detect(ctx, override)
	if err != nil {
		return Info{}, &AppError{Message: "Git not found.", Detail: err.Error()}
	}
	if !versionAtLeast(info.Version, MinimumVersion) {
		return info, &AppError{Message: fmt.Sprintf("Git %s found; %s or newer is required.", info.Version, MinimumVersion)}
	}
	return info, nil
}

func versionAtLeast(version, minimum string) bool {
	have, want := versionNumbers(version), versionNumbers(minimum)
	for i, w := range want {
		h := 0
		if i < len(have) {
			h = have[i]
		}
		if h != w {
			return h > w
		}
	}
	return true
}

// versionNumbers reads the leading dotted numbers, so suffixes such as
// ".windows.1" or " (Apple Git-146)" are ignored.
func versionNumbers(version string) []int {
	var numbers []int
	for _, part := range strings.Split(version, ".") {
		n, err := strconv.Atoi(part)
		if err != nil {
			break
		}
		numbers = append(numbers, n)
	}
	return numbers
}
