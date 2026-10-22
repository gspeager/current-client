// Package hostapi is the one path from the app to a Git host's API, the way
// gitexec is the one path to Git. It sends only to hosts its caller enabled,
// trusts and proxies as Git does for the same URL, backs off when a host
// asks, and keeps what it fetched in a Cache the app reads between
// refreshes.
package hostapi

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gspeager/current-client/core/git"
)

// Host is one the caller enabled: Name as the user sees it in settings
// (hostname, with its port when the remote has one) and API its base URL.
type Host struct {
	Name string
	API  string
}

type TokenSource interface {
	Token(ctx context.Context, hostName string) (string, error)
}

// Transport is what tests hand in instead of a real connection, so the app
// never needs net/http itself.
type Transport = http.RoundTripper

type Options struct {
	// Enabled is asked before every request; the caller decides from its
	// own settings. Nil enables nothing.
	Enabled func(hostName string) bool
	// Tokens is nil for unauthenticated requests.
	Tokens TokenSource
	// Transport replaces the one built from Git's configuration; tests use it.
	Transport Transport
	// Timeout bounds each request; zero means 30 seconds.
	Timeout time.Duration
	// PageCap bounds GetPages; zero means 10 pages.
	PageCap int
}

type Client struct {
	opts  Options
	Cache *Cache

	mu      sync.Mutex
	clients map[string]*http.Client
	waitFor map[string]time.Time
	status  map[string]*HostStatus
}

// HostStatus is what an indicator shows: when the host was last contacted
// and whether a request is in flight now.
type HostStatus struct {
	LastRequest time.Time
	InFlight    int
}

func New(opts Options) *Client {
	if opts.Enabled == nil {
		opts.Enabled = func(string) bool { return false }
	}
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.PageCap == 0 {
		opts.PageCap = 10
	}
	return &Client{opts: opts, Cache: NewCache(), clients: map[string]*http.Client{}, waitFor: map[string]time.Time{}, status: map[string]*HostStatus{}}
}

func (c *Client) Status(hostName string) HostStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	if status, ok := c.status[hostName]; ok {
		return *status
	}
	return HostStatus{}
}

func (c *Client) track(hostName string, delta int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	status, ok := c.status[hostName]
	if !ok {
		status = &HostStatus{}
		c.status[hostName] = status
	}
	status.InFlight += delta
	if delta > 0 {
		status.LastRequest = time.Now()
	}
}

type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

var ErrNotEnabled = errors.New("host integration is off")

// HostError is an answer the host gave that the caller can't use. It carries
// the status and never the response body. A caller that knows which token
// scope the request needed sets Scope, so a 403 can name it.
type HostError struct {
	Host   string
	Status int
	Scope  string
}

func (e *HostError) Error() string {
	switch e.Status {
	case http.StatusUnauthorized:
		return fmt.Sprintf("The token for %s was rejected. Update it in Settings → Integrations.", e.Host)
	case http.StatusForbidden:
		if e.Scope != "" {
			return fmt.Sprintf("The token for %s lacks the %s scope this needs. Update it in Settings → Integrations.", e.Host, e.Scope)
		}
		return fmt.Sprintf("%s refused this: the token lacks permission. Check its scopes in Settings → Integrations.", e.Host)
	}
	return fmt.Sprintf("%s answered %d %s", e.Host, e.Status, http.StatusText(e.Status))
}

// RateLimitError says the host asked for a pause; requests to it fail with
// the same error, without being sent, until Until.
type RateLimitError struct {
	Host  string
	Until time.Time
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("%s asked to wait until %s", e.Host, e.Until.Format(time.Kitchen))
}

const (
	maxBody    = 16 << 20
	maxBackoff = time.Hour
)

func (c *Client) Get(ctx context.Context, host Host, path string, query url.Values) (*Response, error) {
	req, err := http.NewRequest(http.MethodGet, apiURL(host, path, query), nil)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, host, req)
}

func (c *Client) GetJSON(ctx context.Context, host Host, path string, query url.Values, out any) error {
	resp, err := c.Get(ctx, host, path, query)
	if err != nil {
		return err
	}
	return json.Unmarshal(resp.Body, out)
}

// GetPages follows the Link rel="next" header both GitHub and GitLab send,
// calling each per page, and stops at the page cap; truncated says it did.
// A next link that leaves the API address is not followed.
func (c *Client) GetPages(ctx context.Context, host Host, path string, query url.Values, each func(*Response) error) (truncated bool, err error) {
	next := apiURL(host, path, query)
	for page := 0; next != ""; page++ {
		if page == c.opts.PageCap {
			return true, nil
		}
		req, err := http.NewRequest(http.MethodGet, next, nil)
		if err != nil {
			return false, err
		}
		resp, err := c.Do(ctx, host, req)
		if err != nil {
			return false, err
		}
		if err := each(resp); err != nil {
			return false, err
		}
		next = nextLink(resp.Header)
		if next != "" && !underAPI(host, next) {
			return false, nil
		}
	}
	return false, nil
}

// Do refuses, before any connection, a host the caller hasn't enabled and a
// URL outside the host's API address. An answer of 400 or more is a
// HostError, so a caller can still tell a 404 apart.
func (c *Client) Do(ctx context.Context, host Host, req *http.Request) (*Response, error) {
	if !c.opts.Enabled(host.Name) {
		return nil, fmt.Errorf("%s: %w", host.Name, ErrNotEnabled)
	}
	if !underAPI(host, req.URL.String()) {
		return nil, fmt.Errorf("%s is outside %s's API address", req.URL.Redacted(), host.Name)
	}
	if until, ok := c.waiting(host.Name); ok {
		return nil, &RateLimitError{Host: host.Name, Until: until}
	}
	if c.opts.Tokens != nil {
		token, err := c.opts.Tokens.Token(ctx, host.Name)
		if err != nil {
			return nil, err
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	req.Header.Set("Accept", "application/json")
	ctx, cancel := context.WithTimeout(ctx, c.opts.Timeout)
	defer cancel()

	httpClient, err := c.httpClient(ctx, host)
	if err != nil {
		return nil, err
	}
	c.track(host.Name, 1)
	defer c.track(host.Name, -1)
	raw, err := httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", host.Name, plainError(err))
	}
	defer raw.Body.Close()
	body, err := io.ReadAll(io.LimitReader(raw.Body, maxBody))
	if err != nil {
		return nil, fmt.Errorf("%s: reading the answer: %w", host.Name, plainError(err))
	}
	resp := &Response{Status: raw.StatusCode, Header: raw.Header, Body: body}

	if resp.Status == http.StatusTooManyRequests ||
		(resp.Status == http.StatusForbidden && raw.Header.Get("X-RateLimit-Remaining") == "0") {
		until := c.backOff(host.Name, raw.Header)
		return nil, &RateLimitError{Host: host.Name, Until: until}
	}
	if resp.Status == http.StatusUnauthorized {
		if forgetter, ok := c.opts.Tokens.(interface{ Forget(string) }); ok {
			forgetter.Forget(host.Name)
		}
	}
	if resp.Status >= 400 {
		return nil, &HostError{Host: host.Name, Status: resp.Status}
	}
	return resp, nil
}

// Forget drops everything held for a host turned off: its transport, any
// pause it asked for, and what was cached from it.
func (c *Client) Forget(hostName string) {
	c.mu.Lock()
	delete(c.clients, hostName)
	delete(c.waitFor, hostName)
	delete(c.status, hostName)
	c.mu.Unlock()
	c.Cache.ClearHost(hostName)
}

func (c *Client) waiting(hostName string) (time.Time, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	until, ok := c.waitFor[hostName]
	if !ok || !time.Now().Before(until) {
		delete(c.waitFor, hostName)
		return time.Time{}, false
	}
	return until, true
}

// backOff reads Retry-After (seconds or an HTTP date), then GitHub's
// X-RateLimit-Reset or GitLab's RateLimit-Reset (Unix seconds), and waits a
// minute when the host gave neither; never longer than an hour.
func (c *Client) backOff(hostName string, header http.Header) time.Time {
	now := time.Now()
	wait := time.Minute
	if v := header.Get("Retry-After"); v != "" {
		if seconds, err := strconv.Atoi(v); err == nil {
			wait = time.Duration(seconds) * time.Second
		} else if at, err := http.ParseTime(v); err == nil {
			wait = at.Sub(now)
		}
	} else {
		for _, key := range []string{"X-RateLimit-Reset", "RateLimit-Reset"} {
			if epoch, err := strconv.ParseInt(header.Get(key), 10, 64); err == nil {
				wait = time.Unix(epoch, 0).Sub(now)
				break
			}
		}
	}
	wait = max(time.Second, min(wait, maxBackoff))
	until := now.Add(wait)
	c.mu.Lock()
	c.waitFor[hostName] = until
	c.mu.Unlock()
	return until
}

func (c *Client) httpClient(ctx context.Context, host Host) (*http.Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if client, ok := c.clients[host.Name]; ok {
		return client, nil
	}
	transport := c.opts.Transport
	if transport == nil {
		var err error
		if transport, err = transportFor(ctx, host.API); err != nil {
			return nil, err
		}
	}
	client := &http.Client{Transport: transport}
	c.clients[host.Name] = client
	return client, nil
}

// transportFor trusts and proxies as Git does for the API address: its CA
// file or directory instead of the system store when configured, its proxy
// before the environment's. http.sslVerify=false is reported through
// git.HTTPConfigFor for Settings to warn about; verification stays on.
func transportFor(ctx context.Context, apiURL string) (*http.Transport, error) {
	cfg, err := git.HTTPConfigFor(ctx, "", apiURL)
	if err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.Proxy != "" {
		proxyURL, err := url.Parse(cfg.Proxy)
		if err != nil {
			return nil, fmt.Errorf("http.proxy in Git's configuration isn't a URL: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	if cfg.CAInfo != "" || cfg.CAPath != "" {
		pool, err := certPool(cfg.CAInfo, cfg.CAPath)
		if err != nil {
			return nil, err
		}
		transport.TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	}
	return transport, nil
}

func certPool(caInfo, caPath string) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	var files []string
	if caInfo != "" {
		files = append(files, caInfo)
	}
	if caPath != "" {
		entries, err := os.ReadDir(caPath)
		if err != nil {
			return nil, fmt.Errorf("http.sslCAPath: %w", err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				files = append(files, filepath.Join(caPath, entry.Name()))
			}
		}
	}
	for _, file := range files {
		pem, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("reading Git's certificate authority: %w", err)
		}
		pool.AppendCertsFromPEM(pem)
	}
	return pool, nil
}

func apiURL(host Host, path string, query url.Values) string {
	u := strings.TrimRight(host.API, "/") + "/" + strings.TrimLeft(path, "/")
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	return u
}

func underAPI(host Host, rawURL string) bool {
	api, err := url.Parse(host.API)
	if err != nil {
		return false
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return u.Scheme == api.Scheme && u.Host == api.Host &&
		strings.HasPrefix(u.Path, strings.TrimRight(api.Path, "/")+"/")
}

func nextLink(header http.Header) string {
	for _, link := range header.Values("Link") {
		for _, part := range strings.Split(link, ",") {
			target, params, _ := strings.Cut(strings.TrimSpace(part), ";")
			if strings.Contains(params, `rel="next"`) {
				return strings.Trim(strings.TrimSpace(target), "<>")
			}
		}
	}
	return ""
}

// plainError keeps a transport failure readable and free of anything but
// the cause: a timeout, a refused connection, a certificate problem.
func plainError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("no answer in time")
	}
	return err
}
