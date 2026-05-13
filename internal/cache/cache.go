package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dhanifudin/pakai/internal/schema"
)

const (
	// TTL is the cache time-to-live.
	TTL = 60 * time.Second
)

// CacheEntry represents a single cache entry.
type CacheEntry struct {
	Data     []*schema.Usage `json:"data"`
	CachedAt time.Time       `json:"cached_at"`
}

// Cache provides simple JSON file caching.
type Cache struct {
	path string
}

// New creates a new Cache at the given path.
func New(path string) *Cache {
	return &Cache{path: path}
}

// DefaultCache returns a Cache using the default path.
func DefaultCache() (*Cache, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	cacheDir := filepath.Join(home, ".cache", "pakai")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, err
	}
	return New(filepath.Join(cacheDir, "cache.json")), nil
}

// Read reads the cache and returns the entry if it exists and is not expired.
func (c *Cache) Read() (*CacheEntry, error) {
	data, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read cache: %w", err)
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to parse cache: %w", err)
	}

	return &entry, nil
}

// ReadValid reads the cache only if it is within TTL.
func (c *Cache) ReadValid() (*CacheEntry, error) {
	entry, err := c.Read()
	if err != nil || entry == nil {
		return nil, err
	}

	if time.Since(entry.CachedAt) > TTL {
		return nil, nil
	}

	return entry, nil
}

// Write writes the cache entry to disk.
func (c *Cache) Write(data []*schema.Usage) error {
	entry := CacheEntry{
		Data:     data,
		CachedAt: time.Now(),
	}

	b, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create cache dir: %w", err)
	}

	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return fmt.Errorf("failed to write cache: %w", err)
	}

	if err := os.Rename(tmp, c.path); err != nil {
		return fmt.Errorf("failed to rename cache: %w", err)
	}

	return nil
}

// IsStale returns true if the cache is older than TTL.
func (c *Cache) IsStale() bool {
	entry, err := c.Read()
	if err != nil || entry == nil {
		return true
	}
	return time.Since(entry.CachedAt) > TTL
}
