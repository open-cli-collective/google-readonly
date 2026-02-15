package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestNew(t *testing.T) {
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
	c, err := New(24)
	testutil.NoError(t, err)
	defer c.Clear()

	testutil.NotEmpty(t, c.GetDir())
	testutil.Contains(t, c.GetDir(), "cache")
}
