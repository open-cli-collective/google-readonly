package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("creates cache with default TTL", func(t *testing.T) {
		c, err := New(0)
		require.NoError(t, err)
		assert.NotNil(t, c)
		assert.Equal(t, DefaultTTLHours, c.ttlHours)
		defer c.Clear()
	})

	t.Run("creates cache with custom TTL", func(t *testing.T) {
		c, err := New(12)
		require.NoError(t, err)
		assert.Equal(t, 12, c.ttlHours)
		defer c.Clear()
	})

	t.Run("creates cache directory", func(t *testing.T) {
		c, err := New(24)
		require.NoError(t, err)
		defer c.Clear()

		_, err = os.Stat(c.dir)
		assert.NoError(t, err)
	})
}

func TestCache_GetSetDrives(t *testing.T) {
	c, err := New(24)
	require.NoError(t, err)
	defer c.Clear()

	t.Run("returns nil for missing cache", func(t *testing.T) {
		drives, err := c.GetDrives()
		assert.NoError(t, err)
		assert.Nil(t, drives)
	})

	t.Run("stores and retrieves drives", func(t *testing.T) {
		input := []*CachedDrive{
			{ID: "drive1", Name: "Engineering"},
			{ID: "drive2", Name: "Marketing"},
		}

		err := c.SetDrives(input)
		require.NoError(t, err)

		drives, err := c.GetDrives()
		require.NoError(t, err)
		require.Len(t, drives, 2)
		assert.Equal(t, "drive1", drives[0].ID)
		assert.Equal(t, "Engineering", drives[0].Name)
		assert.Equal(t, "drive2", drives[1].ID)
		assert.Equal(t, "Marketing", drives[1].Name)
	})
}

func TestCache_Expiration(t *testing.T) {
	c, err := New(1) // 1 hour TTL
	require.NoError(t, err)
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
		require.NoError(t, err)

		path := filepath.Join(c.dir, DrivesFile)
		err = os.WriteFile(path, data, 0600)
		require.NoError(t, err)

		drives, err := c.GetDrives()
		assert.NoError(t, err)
		assert.Nil(t, drives, "expired cache should return nil")
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
		require.NoError(t, err)

		path := filepath.Join(c.dir, DrivesFile)
		err = os.WriteFile(path, data, 0600)
		require.NoError(t, err)

		drives, err := c.GetDrives()
		assert.NoError(t, err)
		require.Len(t, drives, 1)
		assert.Equal(t, "drive1", drives[0].ID)
	})
}

func TestCache_CorruptedCache(t *testing.T) {
	c, err := New(24)
	require.NoError(t, err)
	defer c.Clear()

	t.Run("returns nil for corrupted JSON", func(t *testing.T) {
		path := filepath.Join(c.dir, DrivesFile)
		err := os.WriteFile(path, []byte("not valid json"), 0600)
		require.NoError(t, err)

		drives, err := c.GetDrives()
		assert.NoError(t, err)
		assert.Nil(t, drives, "corrupted cache should return nil")
	})
}

func TestCache_Clear(t *testing.T) {
	c, err := New(24)
	require.NoError(t, err)

	// Add some data
	err = c.SetDrives([]*CachedDrive{{ID: "test", Name: "Test"}})
	require.NoError(t, err)

	// Verify file exists
	path := filepath.Join(c.dir, DrivesFile)
	_, err = os.Stat(path)
	require.NoError(t, err)

	// Clear cache
	err = c.Clear()
	require.NoError(t, err)

	// Verify directory is gone
	_, err = os.Stat(c.dir)
	assert.True(t, os.IsNotExist(err))
}

func TestCache_GetStatus(t *testing.T) {
	c, err := New(24)
	require.NoError(t, err)
	defer c.Clear()

	t.Run("returns status with no cache", func(t *testing.T) {
		status, err := c.GetStatus()
		require.NoError(t, err)
		assert.Equal(t, c.dir, status.Dir)
		assert.Equal(t, 24, status.TTLHours)
		assert.Nil(t, status.DrivesCache)
	})

	t.Run("returns status with drives cache", func(t *testing.T) {
		err := c.SetDrives([]*CachedDrive{
			{ID: "drive1", Name: "Test1"},
			{ID: "drive2", Name: "Test2"},
		})
		require.NoError(t, err)

		status, err := c.GetStatus()
		require.NoError(t, err)
		require.NotNil(t, status.DrivesCache)
		assert.Equal(t, 2, status.DrivesCache.Count)
		assert.False(t, status.DrivesCache.IsStale)
		assert.True(t, status.DrivesCache.ExpiresAt.After(time.Now()))
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
		require.NoError(t, err)
		require.NotNil(t, status.DrivesCache)
		assert.True(t, status.DrivesCache.IsStale)
	})
}

func TestCache_GetDir(t *testing.T) {
	c, err := New(24)
	require.NoError(t, err)
	defer c.Clear()

	assert.NotEmpty(t, c.GetDir())
	assert.Contains(t, c.GetDir(), "cache")
}
