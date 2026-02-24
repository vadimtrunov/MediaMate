package tmdb

import (
	"sync"
	"time"
)

// cacheEntry holds a cached value and its expiration time.
type cacheEntry struct {
	data      any
	expiresAt time.Time
}

// cache is a thread-safe in-memory cache with TTL-based expiration.
type cache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
	writes  int
}

// newCache creates a new cache with the given TTL for entries.
func newCache(ttl time.Duration) *cache {
	return &cache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
	}
}

// Get returns the cached value for key, or false if not found or expired.
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

// Set stores a value in the cache with the configured TTL.
func (c *cache) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	c.writes++
	// Clean up expired entries periodically (every 100 writes)
	if c.writes%100 == 0 {
		for k, e := range c.entries {
			if now.After(e.expiresAt) {
				delete(c.entries, k)
			}
		}
	}

	c.entries[key] = cacheEntry{
		data:      value,
		expiresAt: now.Add(c.ttl),
	}
}
