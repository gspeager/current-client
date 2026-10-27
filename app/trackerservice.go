package app

import (
	"errors"
	"net/url"
	"slices"

	"github.com/gspeager/current-client/core/hostlink"
	"github.com/gspeager/current-client/internal/config"
	"github.com/gspeager/current-client/internal/policy"
)

// TrackerService keeps the issue trackers beside the hosts, such as a
// team's Jira site, and the links they give a repository's references. The
// links are built from the address the user entered; nothing is asked of
// the tracker.
type TrackerService struct{}

// TrackerInfo is one tracker's row. Managed is the organization's default
// from policy.json, which every repository without its own tracker uses.
type TrackerInfo struct {
	Site    string   `json:"site"`
	Kind    string   `json:"kind"`
	Keys    []string `json:"keys"`
	Repos   []string `json:"repos"`
	Managed bool     `json:"managed"`
	// HostName is what the status bar names while its integration is on.
	HostName string `json:"hostName"`
	// Enabled is the integration switch in force, Locked says the policy
	// fixes it, and HasToken that a token was saved for it.
	Enabled  bool `json:"enabled"`
	Locked   bool `json:"locked"`
	HasToken bool `json:"hasToken"`
}

// TrackerList is Settings' group: the user's trackers, the organization's
// default if any, and the open repositories a tracker can be given to.
type TrackerList struct {
	Trackers  []TrackerInfo `json:"trackers"`
	Default   *TrackerInfo  `json:"default"`
	OpenRepos []string      `json:"openRepos"`
}

// TrackerLinks is what a repository's references link to. KeyPattern and
// BranchKeyPattern are regular expressions with the key in group 1, the same
// in Go and JavaScript.
type TrackerLinks struct {
	Site             string `json:"site"`
	Kind             string `json:"kind"`
	Browse           string `json:"browse"`
	KeyPattern       string `json:"keyPattern"`
	BranchKeyPattern string `json:"branchKeyPattern"`
	Managed          bool   `json:"managed"`
	// Reading is the tracker's integration on: its issues open in the app.
	HostName string `json:"hostName"`
	Reading  bool   `json:"reading"`
}

var errTrackerKind = errors.New("choose Jira Cloud or Jira Data Center")

func (s *TrackerService) List() TrackerList {
	cfg := configOrDefault()
	list := TrackerList{Trackers: []TrackerInfo{}, OpenRepos: append([]string{}, cfg.OpenTabs...)}
	for _, t := range cfg.Trackers {
		info := TrackerInfo{Site: t.Site, Kind: t.Kind, Keys: orEmpty(t.Keys), Repos: orEmpty(t.Repos)}
		list.Trackers = append(list.Trackers, withAccess(info, cfg))
	}
	if d, ok := defaultTracker(); ok {
		info := TrackerInfo{Site: d.Site, Kind: string(d.Kind), Keys: orEmpty(d.Keys), Repos: []string{}, Managed: true}
		list.Default = &info
		*list.Default = withAccess(info, cfg)
	}
	return list
}

func withAccess(info TrackerInfo, cfg config.Config) TrackerInfo {
	access := cfg.TrackerAccess[info.Site]
	info.HostName = trackerHostName(info.Site)
	info.HasToken = access.HasToken
	info.Enabled, info.Locked = access.Enabled, false
	if forced, managed := policy.Host(info.HostName); managed {
		info.Enabled, info.Locked = forced, true
	}
	return info
}

// trackerHostName is the site's host, with its port, as the client names it.
func trackerHostName(site string) string {
	u, err := url.Parse(site)
	if err != nil {
		return ""
	}
	return u.Host
}

func orEmpty(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func (s *TrackerService) Add(site, kind string, keys []string) (TrackerInfo, error) {
	site, err := hostlink.TrackerSite(site)
	if err != nil {
		return TrackerInfo{}, err
	}
	if !validTrackerKind(kind) {
		return TrackerInfo{}, errTrackerKind
	}
	added := config.TrackerConfig{Site: site, Kind: kind, Keys: hostlink.ProjectKeys(keys)}
	err = updateConfigErr(func(cfg *config.Config) error {
		if trackerIndex(cfg, site) >= 0 {
			return errors.New(site + " is already listed")
		}
		cfg.Trackers = append(cfg.Trackers, added)
		return nil
	})
	return TrackerInfo{Site: site, Kind: kind, Keys: orEmpty(added.Keys), Repos: []string{}}, err
}

func (s *TrackerService) SetKeys(site string, keys []string) error {
	return updateTracker(site, func(cfg *config.Config, i int) {
		cfg.Trackers[i].Keys = hostlink.ProjectKeys(keys)
	})
}

// SetRepos gives the tracker these repositories, taking each from any other
// tracker that had it: a repository uses one tracker at most.
func (s *TrackerService) SetRepos(site string, repos []string) error {
	return updateTracker(site, func(cfg *config.Config, i int) {
		for j := range cfg.Trackers {
			if j != i {
				cfg.Trackers[j].Repos = slices.DeleteFunc(cfg.Trackers[j].Repos, func(r string) bool { return slices.Contains(repos, r) })
			}
		}
		cfg.Trackers[i].Repos = slices.Compact(slices.Sorted(slices.Values(repos)))
	})
}

// Remove forgets the tracker, its switch and its token.
func (s *TrackerService) Remove(site string) error {
	hasToken := configOrDefault().TrackerAccess[site].HasToken
	err := updateTracker(site, func(cfg *config.Config, i int) {
		cfg.Trackers = slices.Delete(cfg.Trackers, i, i+1)
		delete(cfg.TrackerAccess, site)
	})
	if err == nil {
		forgetTracker(site, hasToken)
	}
	return err
}

// Links is the repository's own tracker, else the organization's default,
// else nil.
func (s *TrackerService) Links(repoPath string) *TrackerLinks {
	tracker, managed, ok := trackerFor(repoPath)
	if !ok {
		return nil
	}
	return &TrackerLinks{
		Site:             tracker.Site,
		Kind:             string(tracker.Kind),
		Browse:           tracker.IssueURL(""),
		KeyPattern:       tracker.KeyPattern().String(),
		BranchKeyPattern: tracker.BranchKeyPattern().String(),
		Managed:          managed,
		HostName:         trackerHostName(tracker.Site),
		Reading:          trackerReading(trackerHostName(tracker.Site)),
	}
}

func trackerFor(repoPath string) (hostlink.Tracker, bool, bool) {
	for _, t := range configOrDefault().Trackers {
		if slices.Contains(t.Repos, repoPath) {
			return hostlink.Tracker{Site: t.Site, Kind: hostlink.TrackerKind(t.Kind), Keys: t.Keys}, false, true
		}
	}
	if d, ok := defaultTracker(); ok {
		return d, true, true
	}
	return hostlink.Tracker{}, false, false
}

func defaultTracker() (hostlink.Tracker, bool) {
	d := policy.DefaultTracker()
	if d == nil {
		return hostlink.Tracker{}, false
	}
	site, err := hostlink.TrackerSite(d.Site)
	if err != nil {
		return hostlink.Tracker{}, false
	}
	return hostlink.Tracker{Site: site, Kind: hostlink.TrackerKind(d.Kind), Keys: hostlink.ProjectKeys(d.Keys)}, true
}

// allTrackers is every tracker the app may know: the user's and the
// organization's default.
func allTrackers() []hostlink.Tracker {
	var trackers []hostlink.Tracker
	for _, t := range configOrDefault().Trackers {
		trackers = append(trackers, hostlink.Tracker{Site: t.Site, Kind: hostlink.TrackerKind(t.Kind), Keys: t.Keys})
	}
	if d, ok := defaultTracker(); ok {
		trackers = append(trackers, d)
	}
	return trackers
}

func validTrackerKind(kind string) bool {
	return kind == string(hostlink.JiraCloud) || kind == string(hostlink.JiraDataCenter)
}

func trackerIndex(cfg *config.Config, site string) int {
	return slices.IndexFunc(cfg.Trackers, func(t config.TrackerConfig) bool { return t.Site == site })
}

func updateTracker(site string, change func(*config.Config, int)) error {
	return updateConfigErr(func(cfg *config.Config) error {
		i := trackerIndex(cfg, site)
		if i < 0 {
			return errors.New(site + " isn't listed")
		}
		change(cfg, i)
		return nil
	})
}
