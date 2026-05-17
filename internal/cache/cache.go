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

// New creates a new Cache instance rooted at the OS cache dir (B2b). It also
// runs a transparent, best-effort one-time relocation of a pre-B2b cache that
// lived inside the config dir; relocation never fails New (the cache is
// disposable — it simply repopulates).
func New(ttlHours int) (*Cache, error) {
	cacheDir, err := config.GetCacheDir()
	if err != nil {
		return nil, err
	}

	migrateLegacyCacheDir(cacheDir)

	if ttlHours <= 0 {
		ttlHours = DefaultTTLHours
	}

	return &Cache{
		dir:      cacheDir,
		ttlHours: ttlHours,
	}, nil
}

// migrateLegacyCacheDir relocates a pre-B2b cache (a "cache" subdir of the
// config dir) into newDir, then removes the legacy dir. Strictly silent and
// best-effort: any failure is abandoned without touching New's result. If the
// warm-cache carry fails the legacy dir is left intact (don't delete data we
// could not carry); a stuck legacy dir is force-cleaned by `config clear
// --all`. Idempotent: once the legacy dir is gone this is a single stat.
func migrateLegacyCacheDir(newDir string) {
	legacy, err := config.LegacyCacheDir()
	if err != nil {
		return
	}
	if _, err := os.Stat(legacy); err != nil {
		return // absent (steady state) or unreadable — nothing safe to do
	}

	legacyDrives := filepath.Join(legacy, DrivesFile)
	newDrives := filepath.Join(newDir, DrivesFile)
	switch _, serr := os.Stat(newDrives); {
	case serr == nil:
		// New cache already present: nothing to carry, safe to drop legacy.
	case os.IsNotExist(serr):
		// New cache absent: try to carry the warm legacy file.
		data, rerr := os.ReadFile(legacyDrives) //nolint:gosec // G304: path from config dir, not user input
		switch {
		case rerr == nil:
			if werr := os.WriteFile(newDrives, data, config.TokenPerm); werr != nil { //nolint:gosec // G703: config-derived cache path, user's own disposable data
				return // carry failed: keep legacy intact, retry next run
			}
		case os.IsNotExist(rerr):
			// No warm file to carry — fine, fall through to cleanup.
		default:
			return // legacy drives file exists but unreadable: do NOT delete it
		}
	default:
		return // ambiguous stat on the new path: do not risk deleting un-carried legacy
	}
	_ = os.RemoveAll(legacy) // best-effort; cosmetic if it lingers
}

// GetDrives returns cached shared drives, or nil if cache is stale or missing
func (c *Cache) GetDrives() ([]*CachedDrive, error) {
	path := filepath.Join(c.dir, DrivesFile)

	data, err := os.ReadFile(path) //nolint:gosec // Path constructed from known config directory
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
	data, err := os.ReadFile(drivesPath) //nolint:gosec // Path constructed from known config directory
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
