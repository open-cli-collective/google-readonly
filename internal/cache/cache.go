// Package cache provides TTL-based caching for API metadata.
package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/open-cli-collective/google-readonly/internal/config"
)

const (
	// DefaultTTLHours is the default cache TTL if not configured
	DefaultTTLHours = 24
	// CacheDir is the subdirectory within config for cache files
	CacheDir = "cache"
	// DrivesFile is the cache file for shared drives
	DrivesFile = "drives.json"
)

// CachedDrive represents a cached shared drive entry
type CachedDrive struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// DriveCache represents the cached shared drives data
type DriveCache struct {
	CachedAt time.Time      `json:"cached_at"`
	TTLHours int            `json:"ttl_hours"`
	Drives   []*CachedDrive `json:"drives"`
}

// Cache provides TTL-based caching for API metadata
type Cache struct {
	dir      string
	ttlHours int
}

// New creates a new Cache instance
func New(ttlHours int) (*Cache, error) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return nil, err
	}

	cacheDir := filepath.Join(configDir, CacheDir)
	if err := os.MkdirAll(cacheDir, config.DirPerm); err != nil {
		return nil, err
	}

	if ttlHours <= 0 {
		ttlHours = DefaultTTLHours
	}

	return &Cache{
		dir:      cacheDir,
		ttlHours: ttlHours,
	}, nil
}

// GetDrives returns cached shared drives, or nil if cache is stale or missing
func (c *Cache) GetDrives() ([]*CachedDrive, error) {
	path := filepath.Join(c.dir, DrivesFile)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Cache miss, not an error
		}
		return nil, err
	}

	var cache DriveCache
	if err := json.Unmarshal(data, &cache); err != nil {
		// Corrupted cache, treat as miss
		return nil, nil
	}

	// Check if cache is stale
	ttl := time.Duration(cache.TTLHours) * time.Hour
	if time.Since(cache.CachedAt) > ttl {
		return nil, nil // Cache expired
	}

	return cache.Drives, nil
}

// SetDrives updates the cached shared drives
func (c *Cache) SetDrives(drives []*CachedDrive) error {
	cache := DriveCache{
		CachedAt: time.Now(),
		TTLHours: c.ttlHours,
		Drives:   drives,
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(c.dir, DrivesFile)
	return os.WriteFile(path, data, config.TokenPerm)
}

// Clear removes all cached data
func (c *Cache) Clear() error {
	return os.RemoveAll(c.dir)
}

// Status returns information about the cache state
type Status struct {
	Dir         string    `json:"dir"`
	TTLHours    int       `json:"ttl_hours"`
	DrivesCache *FileInfo `json:"drives_cache,omitempty"`
}

// FileInfo contains information about a cache file
type FileInfo struct {
	Path      string    `json:"path"`
	CachedAt  time.Time `json:"cached_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IsStale   bool      `json:"is_stale"`
	Count     int       `json:"count"`
}

// GetStatus returns the current cache status
func (c *Cache) GetStatus() (*Status, error) {
	status := &Status{
		Dir:      c.dir,
		TTLHours: c.ttlHours,
	}

	// Check drives cache
	drivesPath := filepath.Join(c.dir, DrivesFile)
	data, err := os.ReadFile(drivesPath)
	if err == nil {
		var cache DriveCache
		if json.Unmarshal(data, &cache) == nil {
			ttl := time.Duration(cache.TTLHours) * time.Hour
			expiresAt := cache.CachedAt.Add(ttl)
			status.DrivesCache = &FileInfo{
				Path:      drivesPath,
				CachedAt:  cache.CachedAt,
				ExpiresAt: expiresAt,
				IsStale:   time.Now().After(expiresAt),
				Count:     len(cache.Drives),
			}
		}
	}

	return status, nil
}

// GetDir returns the cache directory path
func (c *Cache) GetDir() string {
	return c.dir
}
