// Package update finds the newest published release, only when asked: it
// is the one request the app makes that isn't Git or a host the user
// configured, and nothing here runs on its own.
package update

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/xml"
	"errors"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const ReleasesURL = "https://github.com/gspeager/current-client/releases"

// States of a check.
const (
	UpToDate  = "up-to-date"
	Newer     = "newer"
	Ahead     = "ahead"
	DevBuild  = "dev"
	Unchecked = ""
)

// Result compares the running version with the newest published release.
// ReleasedAt is the release's date when the feed gave it, else "".
type Result struct {
	State      string `json:"state"`
	Current    string `json:"current"`
	Latest     string `json:"latest"`
	ReleasedAt string `json:"releasedAt"`
	LatestURL  string `json:"latestUrl"`
}

// Failures, each in the words the user sees; no raw transport error leaks.
var (
	ErrUnreachable = errors.New("Couldn't reach github.com. Check your connection, or look at the releases page from another computer.")
	ErrTimeout     = errors.New("github.com didn't answer in time.")
	ErrBlocked     = errors.New("The connection to github.com was blocked or couldn't be verified.")
	ErrUnreadable  = errors.New("Couldn't read the latest version.")
)

// Checker reads the version from the redirect of releases/latest, which
// only a published release has (a draft isn't visible, the tag alone isn't
// a release), and the date from the releases feed; neither is the API, so
// no token and no rate limit.
type Checker struct {
	ReleasesURL string
	Client      *http.Client
}

func (c *Checker) Check(ctx context.Context, current string) (Result, error) {
	base := c.ReleasesURL
	if base == "" {
		base = ReleasesURL
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	latest, err := c.latest(ctx, client, base)
	if err != nil {
		return Result{}, err
	}
	result := Result{Current: current, Latest: latest, LatestURL: base + "/tag/v" + latest, ReleasedAt: c.releasedAt(ctx, client, base, latest)}
	cur, ok := parseVersion(current)
	if !ok {
		result.State = DevBuild
		return result, nil
	}
	new, _ := parseVersion(latest)
	switch compare(cur, new) {
	case 0:
		result.State = UpToDate
	case -1:
		result.State = Newer
	default:
		result.State = Ahead
	}
	return result, nil
}

var tagLocation = regexp.MustCompile(`/releases/tag/v?(\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?)$`)

func (c *Checker) latest(ctx context.Context, client *http.Client, base string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, base+"/latest", nil)
	if err != nil {
		return "", ErrUnreadable
	}
	noRedirect := *client
	noRedirect.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := noRedirect.Do(req)
	if err != nil {
		return "", classify(err)
	}
	resp.Body.Close()
	m := tagLocation.FindStringSubmatch(resp.Header.Get("Location"))
	if resp.StatusCode/100 != 3 || m == nil {
		return "", ErrUnreadable
	}
	return m[1], nil
}

type feed struct {
	Entries []struct {
		ID      string `xml:"id"`
		Updated string `xml:"updated"`
	} `xml:"entry"`
}

// releasedAt is best effort: no date is better than no answer.
func (c *Checker) releasedAt(ctx context.Context, client *http.Client, base, version string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+".atom", nil)
	if err != nil {
		return ""
	}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var f feed
	if resp.StatusCode != http.StatusOK || xml.NewDecoder(resp.Body).Decode(&f) != nil {
		return ""
	}
	for _, entry := range f.Entries {
		if strings.HasSuffix(entry.ID, "/v"+version) || strings.HasSuffix(entry.ID, "/"+version) {
			if t, err := time.Parse(time.RFC3339, entry.Updated); err == nil {
				return t.Format("2006-01-02")
			}
			return ""
		}
	}
	return ""
}

func classify(err error) error {
	var netErr net.Error
	var certErr *tls.CertificateVerificationError
	var unknownAuthority x509.UnknownAuthorityError
	var invalidCert x509.CertificateInvalidError
	var hostnameErr x509.HostnameError
	switch {
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &netErr) && netErr.Timeout():
		return ErrTimeout
	case errors.Is(err, context.Canceled):
		return context.Canceled
	case errors.As(err, &certErr), errors.As(err, &unknownAuthority), errors.As(err, &invalidCert), errors.As(err, &hostnameErr),
		strings.Contains(err.Error(), "proxy"), strings.Contains(err.Error(), "tls:"):
		return ErrBlocked
	}
	return ErrUnreachable
}

type version struct {
	major, minor, patch int
	pre                 string
}

var semver = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?$`)

func parseVersion(s string) (version, bool) {
	m := semver.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return version{}, false
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])
	return version{major, minor, patch, m[4]}, true
}

// compare follows semantic versioning: a prerelease sorts before its
// release, and prereleases by their identifiers.
func compare(a, b version) int {
	for _, d := range []int{a.major - b.major, a.minor - b.minor, a.patch - b.patch} {
		if d != 0 {
			return sign(d)
		}
	}
	switch {
	case a.pre == b.pre:
		return 0
	case a.pre == "":
		return 1
	case b.pre == "":
		return -1
	}
	return sign(strings.Compare(a.pre, b.pre))
}

func sign(d int) int {
	if d < 0 {
		return -1
	}
	return 1
}
