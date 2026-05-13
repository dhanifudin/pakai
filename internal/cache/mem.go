package cache

import (
	"sync"
	"time"

	"github.com/dhanifudin/pakai/internal/model"
)

const DefaultMemTTL = 40 * time.Second

type memEntry struct {
	value     model.Usage
	timestamp time.Time
}

type MemCache struct {
	mu    sync.RWMutex
	ttl   time.Duration
	items map[string]memEntry
}

func NewMemCache(ttl time.Duration) *MemCache {
	if ttl <= 0 {
		ttl = DefaultMemTTL
	}
	return &MemCache{
		ttl:   ttl,
		items: make(map[string]memEntry),
	}
}

func (c *MemCache) Get(key string) (model.Usage, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.items[key]
	if !ok {
		return model.Usage{}, true
	}

	stale := time.Since(entry.timestamp) > c.ttl
	return entry.value, stale
}

func (c *MemCache) Set(key string, value model.Usage) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = memEntry{
		value:     value,
		timestamp: time.Now(),
	}
}

func (c *MemCache) All() []model.Usage {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]model.Usage, 0, len(c.items))
	for _, entry := range c.items {
		result = append(result, entry.value)
	}
	return result
}
