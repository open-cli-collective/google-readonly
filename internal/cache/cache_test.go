package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/open-cli-collective/google-readonly/internal/config"
	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

// hermetic isolates every cache path resolver from the developer's real
// dirs (also fixes a pre-existing pollution bug). Covers all OSes
// os.UserCacheDir / config dir resolution can consult: HOME (macOS
// ~/Library/Caches), XDG_CACHE_HOME (Linux/CI), LOCALAPPDATA (Windows),
// XDG_CONFIG_HOME (legacy-cache-dir resolution).
func hermetic(t *testing.T) {
	t.Helper()
	d := t.TempDir()
	t.Setenv("HOME", d)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(d, "xcache"))
	t.Setenv("LOCALAPPDATA", filepath.Join(d, "localappdata"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(d, "xconfig"))
}

func TestNew(t *testing.T) {
	hermetic(t)
	t.Run("creates cache with default TTL", func(t *testing.T) {
		c, err := New(0)
		testutil.NoError(t, err)
		testutil.NotNil(t, c)
		testutil.Equal(t, c.ttlHours, DefaultTTLHours)
		defer c.Clear()
	})

	t.Run("creates cache with custom TTL", func(t *testing.T) {
		c, err := New(12)
		testutil.NoError(t, err)
		testutil.Equal(t, c.ttlHours, 12)
		defer c.Clear()
	})

	t.Run("creates cache directory", func(t *testing.T) {
		c, err := New(24)
		testutil.NoError(t, err)
		defer c.Clear()

		_, err = os.Stat(c.dir)
		testutil.NoError(t, err)
	})
}

func TestCache_GetSetDrives(t *testing.T) {
	hermetic(t)
	c, err := New(24)
	testutil.NoError(t, err)
	defer c.Clear()

	t.Run("returns nil for missing cache", func(t *testing.T) {
		drives, err := c.GetDrives()
		testutil.NoError(t, err)
		testutil.Nil(t, drives)
	})

	t.Run("stores and retrieves drives", func(t *testing.T) {
		input := []*CachedDrive{
			{ID: "drive1", Name: "Engineering"},
			{ID: "drive2", Name: "Marketing"},
		}

		err := c.SetDrives(input)
		testutil.NoError(t, err)

		drives, err := c.GetDrives()
		testutil.NoError(t, err)
		testutil.Len(t, drives, 2)
		testutil.Equal(t, drives[0].ID, "drive1")
		testutil.Equal(t, drives[0].Name, "Engineering")
		testutil.Equal(t, drives[1].ID, "drive2")
		testutil.Equal(t, drives[1].Name, "Marketing")
	})
}

func TestCache_Expiration(t *testing.T) {
	hermetic(t)
	c, err := New(1) // 1 hour TTL
	testutil.NoError(t, err)
	defer c.Clear()

	t.Run("returns nil for expired cache", func(t *testing.T) {
		// Write cache with expired timestamp
		expiredCache := DriveCache{
			CachedAt: time.Now().Add(-2 * time.Hour), // 2 hours ago
			TTLHours: 1,
			Drives: []*CachedDrive{
				{ID: "drive1", Name: "Test"},
			},
		}

		data, err := json.Marshal(expiredCache)
		testutil.NoError(t, err)

		path := filepath.Join(c.dir, DrivesFile)
		err = os.WriteFile(path, data, 0600)
		testutil.NoError(t, err)

		drives, err := c.GetDrives()
		testutil.NoError(t, err)
		testutil.Nil(t, drives)
	})

	t.Run("returns drives for valid cache", func(t *testing.T) {
		// Write fresh cache
		freshCache := DriveCache{
			CachedAt: time.Now(),
			TTLHours: 1,
			Drives: []*CachedDrive{
				{ID: "drive1", Name: "Test"},
			},
		}

		data, err := json.Marshal(freshCache)
		testutil.NoError(t, err)

		path := filepath.Join(c.dir, DrivesFile)
		err = os.WriteFile(path, data, 0600)
		testutil.NoError(t, err)

		drives, err := c.GetDrives()
		testutil.NoError(t, err)
		testutil.Len(t, drives, 1)
		testutil.Equal(t, drives[0].ID, "drive1")
	})
}

func TestCache_CorruptedCache(t *testing.T) {
	hermetic(t)
	c, err := New(24)
	testutil.NoError(t, err)
	defer c.Clear()

	t.Run("returns nil for corrupted JSON", func(t *testing.T) {
		path := filepath.Join(c.dir, DrivesFile)
		err := os.WriteFile(path, []byte("not valid json"), 0600)
		testutil.NoError(t, err)

		drives, err := c.GetDrives()
		testutil.NoError(t, err)
		testutil.Nil(t, drives)
	})
}

func TestCache_Clear(t *testing.T) {
	hermetic(t)
	c, err := New(24)
	testutil.NoError(t, err)

	// Add some data
	err = c.SetDrives([]*CachedDrive{{ID: "test", Name: "Test"}})
	testutil.NoError(t, err)

	// Verify file exists
	path := filepath.Join(c.dir, DrivesFile)
	_, err = os.Stat(path)
	testutil.NoError(t, err)

	// Clear cache
	err = c.Clear()
	testutil.NoError(t, err)

	// Verify directory is gone
	_, err = os.Stat(c.dir)
	testutil.True(t, os.IsNotExist(err))
}

func TestCache_GetStatus(t *testing.T) {
	hermetic(t)
	c, err := New(24)
	testutil.NoError(t, err)
	defer c.Clear()

	t.Run("returns status with no cache", func(t *testing.T) {
		status, err := c.GetStatus()
		testutil.NoError(t, err)
		testutil.Equal(t, status.Dir, c.dir)
		testutil.Equal(t, status.TTLHours, 24)
		testutil.Nil(t, status.DrivesCache)
	})

	t.Run("returns status with drives cache", func(t *testing.T) {
		err := c.SetDrives([]*CachedDrive{
			{ID: "drive1", Name: "Test1"},
			{ID: "drive2", Name: "Test2"},
		})
		testutil.NoError(t, err)

		status, err := c.GetStatus()
		testutil.NoError(t, err)
		testutil.NotNil(t, status.DrivesCache)
		testutil.Equal(t, status.DrivesCache.Count, 2)
		testutil.False(t, status.DrivesCache.IsStale)
		testutil.True(t, status.DrivesCache.ExpiresAt.After(time.Now()))
	})

	t.Run("marks stale cache as stale", func(t *testing.T) {
		// Write expired cache
		expiredCache := DriveCache{
			CachedAt: time.Now().Add(-48 * time.Hour),
			TTLHours: 24,
			Drives:   []*CachedDrive{{ID: "test", Name: "Test"}},
		}
		data, _ := json.Marshal(expiredCache)
		path := filepath.Join(c.dir, DrivesFile)
		os.WriteFile(path, data, 0600)

		status, err := c.GetStatus()
		testutil.NoError(t, err)
		testutil.NotNil(t, status.DrivesCache)
		testutil.True(t, status.DrivesCache.IsStale)
	})
}

func TestCache_GetDir(t *testing.T) {
	hermetic(t)
	c, err := New(24)
	testutil.NoError(t, err)
	defer c.Clear()

	// Assert the exact resolved path, not a substring: with no Windows
	// \Cache suffix and macOS ~/Library/Caches there is no portable
	// "cache" substring (Codex Minor-2).
	want, err := config.CacheDirPath()
	testutil.NoError(t, err)
	testutil.NotEmpty(t, c.GetDir())
	testutil.Equal(t, c.GetDir(), want)
}

func TestMigrateLegacyCacheDir(t *testing.T) {
	// Each subtest fully isolates its own fixtures: a fresh hermetic env +
	// freshly-resolved legacy/newDir, so there is no ordered-cleanup coupling
	// between subtests.
	setup := func(t *testing.T) (legacy, newDir string) {
		t.Helper()
		hermetic(t)
		legacy, err := config.LegacyCacheDir()
		testutil.NoError(t, err)
		newDir, err = config.CacheDirPath()
		testutil.NoError(t, err)
		return legacy, newDir
	}
	seedLegacy := func(t *testing.T, legacy, payload string) {
		t.Helper()
		testutil.NoError(t, os.MkdirAll(legacy, 0o700))
		testutil.NoError(t, os.WriteFile(filepath.Join(legacy, DrivesFile), []byte(payload), 0o600))
	}

	t.Run("carries warm cache then removes legacy", func(t *testing.T) {
		legacy, newDir := setup(t)
		seedLegacy(t, legacy, `{"cached_at":"2026-01-01T00:00:00Z","ttl_hours":24,"drives":[{"id":"d1","name":"Eng"}]}`)

		c, err := New(24)
		testutil.NoError(t, err)
		defer c.Clear()

		got, err := os.ReadFile(filepath.Join(newDir, DrivesFile))
		testutil.NoError(t, err)
		testutil.Contains(t, string(got), `"d1"`)
		_, statErr := os.Stat(legacy)
		testutil.True(t, os.IsNotExist(statErr)) // legacy removed

		// Idempotent: a second New is a no-op and keeps the new cache.
		c2, err := New(24)
		testutil.NoError(t, err)
		defer c2.Clear()
		_, err = os.Stat(filepath.Join(newDir, DrivesFile))
		testutil.NoError(t, err)
	})

	t.Run("does not overwrite an existing new cache; still removes legacy", func(t *testing.T) {
		legacy, _ := setup(t)
		// New cache already warm with distinct content.
		c, err := New(24)
		testutil.NoError(t, err)
		defer c.Clear()
		testutil.NoError(t, c.SetDrives([]*CachedDrive{{ID: "new", Name: "Keep"}}))
		seedLegacy(t, legacy, `{"drives":[{"id":"old","name":"Stale"}]}`)

		c2, err := New(24)
		testutil.NoError(t, err)
		defer c2.Clear()

		drives, err := c2.GetDrives()
		testutil.NoError(t, err)
		testutil.Len(t, drives, 1)
		testutil.Equal(t, drives[0].ID, "new") // not overwritten by legacy
		_, statErr := os.Stat(legacy)
		testutil.True(t, os.IsNotExist(statErr)) // legacy still removed
	})

	t.Run("unreadable legacy drives file: legacy preserved, no partial carry", func(t *testing.T) {
		legacy, newDir := setup(t)
		// drives.json as a directory => os.ReadFile errors with a
		// non-IsNotExist error => carry fails => legacy must NOT be removed
		// and nothing must be written into the new cache.
		testutil.NoError(t, os.MkdirAll(filepath.Join(legacy, DrivesFile), 0o700))

		c, err := New(24)
		testutil.NoError(t, err) // migration never fails New
		defer c.Clear()

		_, statErr := os.Stat(legacy)
		testutil.NoError(t, statErr) // legacy left intact (not deleted)
		_, newErr := os.Stat(filepath.Join(newDir, DrivesFile))
		testutil.True(t, os.IsNotExist(newErr)) // no partial/corrupt carry
	})

	t.Run("no legacy is a clean no-op", func(t *testing.T) {
		setup(t)
		c, err := New(24)
		testutil.NoError(t, err) // migration never fails New
		defer c.Clear()
	})
}
