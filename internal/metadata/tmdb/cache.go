package tmdb

import (
	"sync"
	"time"
)

type cacheEntry struct {
	data      any
	expiresAt time.Time
}

type cache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
	writes  int
}

func newCache(ttl time.Duration) *cache {
	return &cache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
	}
}

func (c *cache) Get(key string) (any, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok || time.Now().After(entry.expiresAt) {
		if ok {
			// Expired — remove lazily
			c.mu.Lock()
			// Re-check under write lock to avoid deleting a fresh entry
			if e, exists := c.entries[key]; exists && time.Now().After(e.expiresAt) {
				delete(c.entries, key)
			}
			c.mu.Unlock()
		}
		return nil, false
	}
	return entry.data, true
}

func (c *cache) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.writes++
	// Clean up expired entries periodically (every 100 writes)
	if c.writes%100 == 0 {
		now := time.Now()
		for k, e := range c.entries {
			if now.After(e.expiresAt) {
				delete(c.entries, k)
			}
		}
	}

	c.entries[key] = cacheEntry{
		data:      value,
		expiresAt: time.Now().Add(c.ttl),
	}
}
