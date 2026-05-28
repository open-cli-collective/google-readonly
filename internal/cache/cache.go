// Package cache wraps cli-common/cache for gro's Drive metadata cache.
//
// Per cli-common/docs/working-with-state.md §4, gro's cache is disposable
// state at os.UserCacheDir()/google-readonly (via statedir.Cache). Writes are
// atomic via cli-common/cache's temp+rename envelope; TTL is hard-coded per
// resource (no user-configurable cache_ttl_hours — §4.4); reads classify a
// version/identity mismatch as a miss so schema bumps self-heal. The pre-B2b
// "<configdir>/cache/" relocation is retained for installs that pre-date the
// B2b cache move.
package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	clicache "github.com/open-cli-collective/cli-common/cache"

	"github.com/open-cli-collective/google-readonly/internal/config"
)

const (
	// DrivesFile is the cache file for shared drives. Retained as the legacy
	// pre-B2b file name so migrateLegacyCacheDir can find and relocate it.
	DrivesFile = "drives.json"
	// drivesResource is the cli-common cache resource name. The on-disk file
	// becomes <cachedir>/<instanceKey>/drives.json.
	drivesResource = "drives"
	// instanceKey is gro's single-instance discriminator (Locator requires
	// one). Per cli-common docs: single-instance CLIs use "default".
	instanceKey = "default"
	// drivesTTL is the §4.4 hard-coded per-resource TTL for shared drives —
	// same 24-hour default the user-configurable knob previously defaulted to.
	drivesTTL = "24h"
)

// CachedDrive represents a cached shared drive entry. Public so callers
// (drives.go) can populate it directly.
type CachedDrive struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Cache is gro's wrapper around the cli-common envelope cache.
type Cache struct {
	loc clicache.Locator
}

// New creates a new Cache instance rooted at the OS cache dir (B2b via
// cli-common/statedir). It also runs a transparent, best-effort one-time
// relocation of a pre-B2b cache that lived inside the config dir; relocation
// never fails New (the cache is disposable — it simply repopulates).
func New() (*Cache, error) {
	cacheDir, err := config.GetCacheDir()
	if err != nil {
		return nil, err
	}
	migrateLegacyCacheDir(cacheDir)
	return &Cache{
		loc: clicache.Locator{Root: cacheDir, InstanceKey: instanceKey},
	}, nil
}

// migrateLegacyCacheDir relocates a pre-B2b cache (a "cache" subdir of the
// config dir) into newDir, then removes the legacy dir. Strictly silent and
// best-effort: any failure is abandoned without touching New's result. If the
// warm-cache carry fails the legacy dir is left intact; a stuck legacy dir is
// force-cleaned by `config clear --all`. The carry is a byte copy and does
// NOT parse the envelope; a stale shape arriving in newDir is harmless
// because cli-common/cache's identity/version check + this package's
// parse-error-mapped-to-miss demote it on the next read.
func migrateLegacyCacheDir(newDir string) {
	// Two possible legacy locations on macOS/Windows: (a) the pre-B2b
	// "cache" subdir under the NEW (statedir-resolved) config dir — what a
	// user reached if they already ran post-port code once — and (b) the
	// same subdir under the OLD hand-rolled config dir, which is where a
	// pure pre-MON-5371 install lived. Both are byte-carried best-effort;
	// any failure abandons the migration without touching New's result. On
	// Linux the two paths collapse, so the second probe is a stat-only no-op.
	probes := []func() (string, error){
		config.LegacyCacheDir,
		config.OldHandRolledLegacyCacheDir,
	}
	tried := map[string]bool{}
	for _, p := range probes {
		legacy, err := p()
		if err != nil || tried[legacy] {
			continue
		}
		tried[legacy] = true
		tryCarryLegacyCache(legacy, newDir)
	}
}

func tryCarryLegacyCache(legacy, newDir string) {
	if _, err := os.Stat(legacy); err != nil {
		return
	}

	legacyDrives := filepath.Join(legacy, DrivesFile)
	newDrives := filepath.Join(newDir, instanceKey, DrivesFile)
	switch _, serr := os.Stat(newDrives); {
	case serr == nil:
		// New cache already present: nothing to carry, safe to drop legacy.
	case os.IsNotExist(serr):
		data, rerr := os.ReadFile(legacyDrives) //nolint:gosec // path from legacy config dir
		switch {
		case rerr == nil:
			if mkErr := os.MkdirAll(filepath.Dir(newDrives), 0o700); mkErr != nil {
				return
			}
			if werr := os.WriteFile(newDrives, data, config.TokenPerm); werr != nil { //nolint:gosec // disposable cache
				return
			}
		case os.IsNotExist(rerr):
			// No warm file to carry — fine, fall through to cleanup.
		default:
			return
		}
	default:
		return
	}
	_ = os.RemoveAll(legacy)
}

// GetDrives returns cached shared drives, or nil if cache is stale, missing,
// or corrupt. Corrupt-as-miss preserves the pre-MON-5371 behavior: caches
// are disposable, so a JSON parse error self-heals on the next API call. I/O
// errors (read failure, permission denied) propagate.
func (c *Cache) GetDrives() ([]*CachedDrive, error) {
	env, err := clicache.ReadResource[[]*CachedDrive](c.loc, drivesResource)
	switch {
	case errors.Is(err, clicache.ErrCacheMiss):
		return nil, nil
	case err != nil:
		var syn *json.SyntaxError
		var ute *json.UnmarshalTypeError
		if errors.As(err, &syn) || errors.As(err, &ute) {
			return nil, nil // corrupt → miss (self-heals on next write)
		}
		return nil, fmt.Errorf("reading drives cache: %w", err)
	}

	if clicache.Classify(env.FetchedAt, env.TTL, nowFn()) == clicache.StatusStale {
		return nil, nil // stale → miss
	}
	return env.Data, nil
}

// SetDrives atomically writes the drives cache with the §4.4 hard-coded TTL.
func (c *Cache) SetDrives(drives []*CachedDrive) error {
	if err := clicache.WriteResource(c.loc, drivesResource, drivesTTL, drives); err != nil {
		return fmt.Errorf("writing drives cache: %w", err)
	}
	return nil
}

// DrivesStatus reports the freshness of the cached drives entry without
// fetching from the API. Returns (fetchedAt, ttl, status). A missing or
// corrupt envelope returns (time.Time{}, drivesTTL, StatusUninitialized);
// I/O errors propagate. The TTL string is the hard-coded §4.4 value so the
// `refresh --status` table can render it without callers re-deriving it.
func (c *Cache) DrivesStatus() (time.Time, string, clicache.Status, error) {
	env, err := clicache.ReadResource[[]*CachedDrive](c.loc, drivesResource)
	switch {
	case errors.Is(err, clicache.ErrCacheMiss):
		return time.Time{}, drivesTTL, clicache.StatusUninitialized, nil
	case err != nil:
		var syn *json.SyntaxError
		var ute *json.UnmarshalTypeError
		if errors.As(err, &syn) || errors.As(err, &ute) {
			return time.Time{}, drivesTTL, clicache.StatusUninitialized, nil
		}
		return time.Time{}, drivesTTL, clicache.StatusUninitialized, fmt.Errorf("reading drives cache: %w", err)
	}
	return env.FetchedAt, drivesTTL, clicache.Classify(env.FetchedAt, env.TTL, nowFn()), nil
}

// Clear removes all cached data for this instance. Scoped to
// <Root>/<InstanceKey> rather than the tool-level Root so a future move to a
// multi-instance Locator can't have one instance's Clear() silently evict
// every other instance's cache.
func (c *Cache) Clear() error {
	return os.RemoveAll(filepath.Join(c.loc.Root, c.loc.InstanceKey))
}

// GetDir returns the cache directory path.
func (c *Cache) GetDir() string {
	return c.loc.Root
}

// nowFn is a testing seam for cache classification.
var nowFn = func() time.Time { return time.Now() }
