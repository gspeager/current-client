//go:build !nohost

package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/hostapi"
	"github.com/gspeager/current-client/core/hostlink"
	"github.com/gspeager/current-client/internal/config"
	"github.com/gspeager/current-client/internal/policy"
)

// HostService works out which Git host each remote belongs to, from the
// remote URL and the user's settings alone, and holds the per-host switches.
// Only TestConnection contacts a host, and only one that is enabled.
type HostService struct{}

// Provider and APIURL are what is in effect; the Detected and Default fields
// are what the remote URL alone gives, for Settings to show beside a choice.
// SSLVerifyOff reports Git's http.sslVerify=false for the API address, which
// Settings warns about; certificate checks stay on regardless. Remote and
// Project are empty for a host added by hand.
type HostInfo struct {
	Remote           string `json:"remote"`
	Hostname         string `json:"hostname"`
	Project          string `json:"project"`
	Provider         string `json:"provider"`
	DetectedProvider string `json:"detectedProvider"`
	APIURL           string `json:"apiUrl"`
	DefaultAPIURL    string `json:"defaultApiUrl"`
	Enabled          bool   `json:"enabled"`
	Managed          bool   `json:"managed"`
	HasToken         bool   `json:"hasToken"`
	Dismissed        bool   `json:"dismissed"`
	SSLVerifyOff     bool   `json:"sslVerifyOff"`
}

type HostStatusInfo struct {
	LastRequest *time.Time `json:"lastRequest"`
	InFlight    bool       `json:"inFlight"`
}

func (s *HostService) Detect(repoPath string) ([]HostInfo, error) {
	remotes, err := git.ListRemotes(context.Background(), repoPath)
	if err != nil {
		return nil, err
	}
	hosts := configOrDefault().Hosts
	infos := []HostInfo{}
	for _, remote := range remotes {
		if info, ok := hostInfoFor(remote, hosts); ok {
			infos = append(infos, info)
		}
	}
	return infos, nil
}

// List is every host Settings shows: those of the given repositories'
// remotes, then any other host in settings, by name.
func (s *HostService) List(repoPaths []string) ([]HostInfo, error) {
	hosts := configOrDefault().Hosts
	infos := []HostInfo{}
	seen := map[string]bool{}
	for _, repoPath := range repoPaths {
		detected, err := s.Detect(repoPath)
		if err != nil {
			return nil, err
		}
		for _, info := range detected {
			if !seen[info.Hostname] {
				seen[info.Hostname] = true
				infos = append(infos, info)
			}
		}
	}
	names := make([]string, 0, len(hosts))
	for name := range hosts {
		if !seen[name] {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		infos = append(infos, infoFor(hostByName(name), hosts))
	}
	return infos, nil
}

func (s *HostService) Status(hostname string) HostStatusInfo {
	status := hostClient.Status(hostname)
	info := HostStatusInfo{InFlight: status.InFlight > 0}
	if !status.LastRequest.IsZero() {
		info.LastRequest = &status.LastRequest
	}
	return info
}

// SetEnabled is the one switch that lets the app contact a host. Turning a
// host off drops everything the client holds for it, as the integrations
// guide promises.
func (s *HostService) SetEnabled(hostname string, enabled bool) error {
	if _, managed := policy.Host(hostname); managed {
		return fmt.Errorf("integration for %s is managed by your organization", hostname)
	}
	if !enabled {
		hostClient.Forget(hostname)
		hostCredentials.Forget(hostname)
		deleteFetchedHeads(hostname)
	}
	return setHostFlag(hostname, func(host *config.HostConfig) {
		host.Enabled = enabled
		if enabled {
			host.Dismissed = false
		}
	})
}

func (s *HostService) Dismiss(hostname string) error {
	return setHostFlag(hostname, func(host *config.HostConfig) { host.Dismissed = true })
}

// Add records a host that no open repository's remote points at yet.
func (s *HostService) Add(hostname, provider string) error {
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	if hostname == "" || strings.ContainsAny(hostname, "/ \t") {
		return fmt.Errorf("enter a hostname, such as gitlab.example.com")
	}
	if provider == "" {
		provider = string(hostlink.DefaultStyle(hostname))
	}
	return s.SetHost(hostname, provider, "")
}

// Remove forgets the host's settings and, when one was saved, its token.
func (s *HostService) Remove(hostname string) error {
	if configOrDefault().Hosts[hostname].HasToken {
		if err := hostCredentials.Remove(context.Background(), hostname); err != nil {
			return err
		}
	}
	hostClient.Forget(hostname)
	hostCredentials.Forget(hostname)
	return updateConfig(func(cfg *config.Config) { delete(cfg.Hosts, hostname) })
}

// SetHost records the provider type and API address for a host. Empty values
// return to what the remote URL gives.
func (s *HostService) SetHost(hostname, provider, apiURL string) error {
	switch hostlink.Style(provider) {
	case "", hostlink.GitHub, hostlink.GitLab:
	default:
		return fmt.Errorf("unknown provider %q", provider)
	}
	apiURL = strings.TrimRight(strings.TrimSpace(apiURL), "/")
	if apiURL != "" {
		u, err := url.Parse(apiURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("API address must be an http(s) URL: %q", apiURL)
		}
	}
	hostClient.Forget(hostname)
	return setHostFlag(hostname, func(host *config.HostConfig) { host.Provider, host.APIURL = provider, apiURL })
}

// SaveToken hands a pasted API token to Git's credential helper under its
// own key (see hostapi.GitCredentials) and records only that one exists.
func (s *HostService) SaveToken(hostname, token string) error {
	if err := hostCredentials.Save(context.Background(), hostname, token); err != nil {
		return err
	}
	return setHostFlag(hostname, func(host *config.HostConfig) { host.HasToken = true })
}

func (s *HostService) RemoveToken(hostname string) error {
	if err := hostCredentials.Remove(context.Background(), hostname); err != nil {
		return err
	}
	return setHostFlag(hostname, func(host *config.HostConfig) { host.HasToken = false })
}

// TestConnection asks the host who the token belongs to, through the same
// client and rules as every other request: the provider where one exists,
// else the host's user endpoint.
func (s *HostService) TestConnection(hostname string) (string, error) {
	hosts := configOrDefault().Hosts
	info := infoFor(hostByName(hostname), hosts)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var name string
	var err error
	if provider := providerFor(info); provider != nil {
		var user hostapi.User
		user, err = provider.CurrentUser(ctx)
		name = user.Username
	} else {
		var user struct {
			Username string `json:"username"`
			Login    string `json:"login"`
		}
		err = hostClient.GetJSON(ctx, hostapi.Host{Name: hostname, API: info.APIURL}, "user", nil, &user)
		name = user.Username
		if name == "" {
			name = user.Login
		}
	}
	if errors.Is(err, hostapi.ErrNotEnabled) {
		return "", fmt.Errorf("integration for %s is off", hostname)
	}
	if err != nil {
		return "", err
	}
	if name == "" {
		return "Connected to " + info.APIURL + ".", nil
	}
	return "Connected to " + info.APIURL + " as " + name + ".", nil
}

func setHostFlag(hostname string, set func(*config.HostConfig)) error {
	return updateConfig(func(cfg *config.Config) {
		host := cfg.Hosts[hostname]
		set(&host)
		if host == (config.HostConfig{}) {
			delete(cfg.Hosts, hostname)
			return
		}
		if cfg.Hosts == nil {
			cfg.Hosts = map[string]config.HostConfig{}
		}
		cfg.Hosts[hostname] = host
	})
}

func hostInfoFor(remote git.Remote, hosts map[string]config.HostConfig) (HostInfo, bool) {
	detected, ok := hostlink.Resolve(context.Background(), remoteURL(remote))
	if !ok {
		return HostInfo{}, false
	}
	info := infoFor(detected, hosts)
	info.Remote, info.Project = remote.Name, detected.Project
	return info, true
}

// hostByName is a host with no remote behind it: its style is guessed from
// the name alone.
func hostByName(hostname string) hostlink.Host {
	return hostlink.Host{Root: "https://" + hostname + "/", Style: hostlink.DefaultStyle(hostname)}
}

func infoFor(detected hostlink.Host, hosts map[string]config.HostConfig) HostInfo {
	name := detected.Hostname()
	cfg := hosts[name]
	host := withHostConfig(detected, hosts)
	apiURL := cfg.APIURL
	if apiURL == "" {
		apiURL = host.API()
	}
	httpConfig, _ := git.HTTPConfigFor(context.Background(), "", apiURL)
	enabled, managed := effectiveHostEnabled(name, cfg.Enabled)
	return HostInfo{
		Hostname:         name,
		Provider:         string(host.Style),
		DetectedProvider: string(detected.Style),
		APIURL:           apiURL,
		DefaultAPIURL:    host.API(),
		Enabled:          enabled,
		Managed:          managed,
		HasToken:         cfg.HasToken,
		Dismissed:        cfg.Dismissed,
		SSLVerifyOff:     !httpConfig.SSLVerify,
	}
}
