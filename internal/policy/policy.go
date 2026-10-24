// Package policy reads the machine-wide managed policy an organization
// installs, which sits above every user's own settings. It is read once, at
// start-up, from one fixed path per platform, and never contacts anything.
package policy

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// Policy locks a networked feature off, forces it on, or, where a switch is
// absent, leaves it to the user. Hosts are keyed by hostname, as Settings
// shows them.
type Policy struct {
	Hosts     map[string]Switch `json:"hosts"`
	AutoFetch *AutoFetch        `json:"autoFetch"`
}

type Switch struct {
	Enabled *bool `json:"enabled"`
}

// AutoFetch forced on without an interval fetches every five minutes.
type AutoFetch struct {
	Switch
	IntervalMinutes int `json:"intervalMinutes"`
}

const defaultIntervalMinutes = 5

var (
	mu      sync.Mutex
	path    string
	current Policy
)

// Path is where the policy file lives: a machine-wide, administrator-owned
// place on each platform.
func Path() string {
	mu.Lock()
	defer mu.Unlock()
	if path != "" {
		return path
	}
	return defaultPath()
}

func defaultPath() string {
	switch runtime.GOOS {
	case "windows":
		programData := os.Getenv("ProgramData")
		if programData == "" {
			programData = `C:\ProgramData`
		}
		return filepath.Join(programData, "Current Client", "policy.json")
	case "darwin":
		return "/Library/Application Support/Current Client/policy.json"
	}
	return "/etc/current-client/policy.json"
}

// SetPath points Load at another file, for tests and embedding apps. It
// clears what was loaded.
func SetPath(p string) {
	mu.Lock()
	defer mu.Unlock()
	path, current = p, Policy{}
}

// Load reads the file at Path. A missing file is no policy; one that can't
// be read or parsed is reported and treated the same.
func Load() error {
	data, err := os.ReadFile(Path())
	mu.Lock()
	defer mu.Unlock()
	current = Policy{}
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var loaded Policy
	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}
	current = loaded
	return nil
}

// Host is what the policy fixes for a host's integration: managed says a
// switch is set, and enabled is its value.
func Host(hostname string) (enabled, managed bool) {
	mu.Lock()
	defer mu.Unlock()
	sw, ok := current.Hosts[hostname]
	if !ok || sw.Enabled == nil {
		return false, false
	}
	return *sw.Enabled, true
}

// AutoFetchMinutes is the background fetch interval in force: the policy's
// when it sets one, else the user's. Zero is off.
func AutoFetchMinutes(user int) (minutes int, managed bool) {
	mu.Lock()
	defer mu.Unlock()
	af := current.AutoFetch
	if af == nil || af.Enabled == nil {
		return user, false
	}
	if !*af.Enabled {
		return 0, true
	}
	if af.IntervalMinutes > 0 {
		return af.IntervalMinutes, true
	}
	return defaultIntervalMinutes, true
}
