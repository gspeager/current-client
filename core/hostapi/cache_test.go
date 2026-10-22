package hostapi

import (
	"testing"
	"time"
)

func TestCacheClearsByHostAndRepository(t *testing.T) {
	cache := NewCache()
	cache.Put("gitlab.example.com", "/repo/a", "mr", "7", "a7")
	cache.Put("gitlab.example.com", "/repo/b", "mr", "8", "b8")
	cache.Put("github.com", "/repo/a", "pr", "9", "a9")

	value, at, ok := cache.Get("gitlab.example.com", "/repo/a", "mr", "7")
	if !ok || value != "a7" || time.Since(at) > time.Minute {
		t.Fatalf("Get = %v, %v, %v", value, at, ok)
	}
	if _, _, ok := cache.Get("gitlab.example.com", "/repo/a", "mr", "8"); ok {
		t.Fatal("a different id was found")
	}

	cache.ClearHost("gitlab.example.com")
	if _, _, ok := cache.Get("gitlab.example.com", "/repo/b", "mr", "8"); ok {
		t.Fatal("ClearHost left the host's entries")
	}
	if _, _, ok := cache.Get("github.com", "/repo/a", "pr", "9"); !ok {
		t.Fatal("ClearHost removed another host's entry")
	}

	cache.ClearRepo("/repo/a")
	if _, _, ok := cache.Get("github.com", "/repo/a", "pr", "9"); ok {
		t.Fatal("ClearRepo left the repository's entry")
	}
}
