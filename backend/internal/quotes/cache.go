package quotes

import (
	"sync"
	"time"
)

type cacheItem struct {
	quote   Quote
	expires time.Time
}

// cache is a tiny TTL cache keyed by "provider:symbol" to avoid hammering
// upstream APIs (important for NSE, which blocks aggressive callers).
type cache struct {
	mu    sync.Mutex
	items map[string]cacheItem
}

func newCache() *cache {
	return &cache{items: map[string]cacheItem{}}
}

func (c *cache) get(key string) (Quote, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	it, ok := c.items[key]
	if !ok || time.Now().After(it.expires) {
		return Quote{}, false
	}
	return it.quote, true
}

func (c *cache) set(key string, q Quote, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = cacheItem{quote: q, expires: time.Now().Add(ttl)}
}
