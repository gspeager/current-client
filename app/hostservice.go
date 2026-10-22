package app

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/gspeager/current-client/core/git"
	"github.com/gspeager/current-client/core/hostlink"
	"github.com/gspeager/current-client/internal/config"
)

// HostService works out which Git host each remote belongs to, from the
// remote URL and the user's settings alone. Nothing here contacts a host.
type HostService struct{}

// Provider and APIURL are what is in effect; the Detected and Default fields
// are what the remote URL alone gives, for Settings to show beside a choice.
// SSLVerifyOff reports Git's http.sslVerify=false for the API address, which
// Settings warns about; certificate checks stay on regardless.
type HostInfo struct {
	Remote           string `json:"remote"`
	Hostname         string `json:"hostname"`
	Project          string `json:"project"`
	Provider         string `json:"provider"`
	DetectedProvider string `json:"detectedProvider"`
	APIURL           string `json:"apiUrl"`
	DefaultAPIURL    string `json:"defaultApiUrl"`
	Enabled          bool   `json:"enabled"`
	HasToken         bool   `json:"hasToken"`
	SSLVerifyOff     bool   `json:"sslVerifyOff"`
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
	host := withHostConfig(detected, hosts)
	apiURL := hosts[host.Hostname()].APIURL
	if apiURL == "" {
		apiURL = host.API()
	}
	httpConfig, _ := git.HTTPConfigFor(context.Background(), "", apiURL)
	return HostInfo{
		Remote:           remote.Name,
		Hostname:         host.Hostname(),
		Project:          host.Project,
		Provider:         string(host.Style),
		DetectedProvider: string(detected.Style),
		APIURL:           apiURL,
		DefaultAPIURL:    host.API(),
		Enabled:          hosts[host.Hostname()].Enabled,
		HasToken:         hosts[host.Hostname()].HasToken,
		SSLVerifyOff:     !httpConfig.SSLVerify,
	}, true
}

func withHostConfig(host hostlink.Host, hosts map[string]config.HostConfig) hostlink.Host {
	if provider := hosts[host.Hostname()].Provider; provider != "" {
		host.Style = hostlink.Style(provider)
	}
	return host
}

func remoteURL(remote git.Remote) string {
	if remote.FetchURL != "" {
		return remote.FetchURL
	}
	return remote.PushURL
}
