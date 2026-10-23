package hostapi

import (
	"sync"
	"time"
)

// Cache holds what was last fetched from each host for each repository, in
// memory only, each entry with the time it was fetched so a view can label
// its age. Nothing expires: a refresh replaces an entry, turning the host
// off or closing the repository removes it.
type Cache struct {
	mu      sync.Mutex
	entries map[cacheKey]cacheEntry
}

type cacheKey struct{ host, repo, kind, id string }

type cacheEntry struct {
	value any
	at    time.Time
}

func NewCache() *Cache {
	return &Cache{entries: map[cacheKey]cacheEntry{}}
}

func (c *Cache) Put(host, repo, kind, id string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[cacheKey{host, repo, kind, id}] = cacheEntry{value: value, at: time.Now()}
}

func (c *Cache) Get(host, repo, kind, id string) (value any, fetchedAt time.Time, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[cacheKey{host, repo, kind, id}]
	return entry.value, entry.at, ok
}

// Values returns every entry of one kind for a host and repository.
func (c *Cache) Values(host, repo, kind string) []any {
	c.mu.Lock()
	defer c.mu.Unlock()
	var values []any
	for key, entry := range c.entries {
		if key.host == host && key.repo == repo && key.kind == kind {
			values = append(values, entry.value)
		}
	}
	return values
}

func (c *Cache) ClearHost(host string) {
	c.clear(func(key cacheKey) bool { return key.host == host })
}

func (c *Cache) ClearRepo(repo string) {
	c.clear(func(key cacheKey) bool { return key.repo == repo })
}

// ClearKind drops one kind of entry for a host and repository, for data a
// refresh must ask for again rather than merge.
func (c *Cache) ClearKind(host, repo, kind string) {
	c.clear(func(key cacheKey) bool { return key.host == host && key.repo == repo && key.kind == kind })
}

func (c *Cache) clear(match func(cacheKey) bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.entries {
		if match(key) {
			delete(c.entries, key)
		}
	}
}
