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
	now := time.Now()
	c.mu.RLock()
	entry, ok := c.entries[key]
	if ok && !now.After(entry.expiresAt) {
		c.mu.RUnlock()
		return entry.data, true
	}
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	// Expired — remove lazily
	c.mu.Lock()
	defer c.mu.Unlock()
	// Re-check under write lock; return fresh entry if present
	if e, exists := c.entries[key]; exists {
		if time.Now().After(e.expiresAt) {
			delete(c.entries, key)
			return nil, false
		}
		return e.data, true
	}
	return nil, false
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
