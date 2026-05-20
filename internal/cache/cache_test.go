package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/open-cli-collective/cli-common/statedirtest"

	"github.com/open-cli-collective/google-readonly/internal/config"
	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

// hermetic isolates the §3.1 7-var env set so os.UserCacheDir /
// os.UserConfigDir resolve under a per-test temp root on every OS.
func hermetic(t *testing.T) {
	t.Helper()
	statedirtest.Hermetic(t)
}

func TestNew(t *testing.T) {
	hermetic(t)
	t.Run("creates cache", func(t *testing.T) {
		c, err := New()
		testutil.NoError(t, err)
		testutil.NotNil(t, c)
		defer c.Clear()
	})

	t.Run("creates cache directory", func(t *testing.T) {
		c, err := New()
		testutil.NoError(t, err)
		defer c.Clear()

		_, err = os.Stat(c.loc.Root)
		testutil.NoError(t, err)
	})
}

func TestCache_GetSetDrives(t *testing.T) {
	hermetic(t)
	c, err := New()
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
	c, err := New()
	testutil.NoError(t, err)
	defer c.Clear()

	t.Run("classifies stale envelope as miss", func(t *testing.T) {
		testutil.NoError(t, c.SetDrives([]*CachedDrive{{ID: "x", Name: "y"}}))

		// Pretend "now" is two days ahead of the envelope's FetchedAt (TTL is 24h).
		origNow := nowFn
		nowFn = func() time.Time { return time.Now().Add(48 * time.Hour) }
		defer func() { nowFn = origNow }()

		drives, err := c.GetDrives()
		testutil.NoError(t, err)
		testutil.Nil(t, drives)
	})

	t.Run("returns drives for fresh envelope", func(t *testing.T) {
		testutil.NoError(t, c.SetDrives([]*CachedDrive{{ID: "drive1", Name: "Test"}}))
		drives, err := c.GetDrives()
		testutil.NoError(t, err)
		testutil.Len(t, drives, 1)
		testutil.Equal(t, drives[0].ID, "drive1")
	})
}

func TestCache_CorruptedCache(t *testing.T) {
	hermetic(t)
	c, err := New()
	testutil.NoError(t, err)
	defer c.Clear()

	t.Run("malformed JSON treated as miss (disposable state)", func(t *testing.T) {
		// Write a malformed envelope at the same path cli-common would use:
		// <Root>/<InstanceKey>/<resource>.json.
		path := filepath.Join(c.loc.Root, c.loc.InstanceKey, drivesResource+".json")
		testutil.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		testutil.NoError(t, os.WriteFile(path, []byte("not valid json"), 0o600))

		drives, err := c.GetDrives()
		testutil.NoError(t, err)
		testutil.Nil(t, drives)
	})

	t.Run("old pre-MON-5371 DriveCache shape treated as miss", func(t *testing.T) {
		// cli-common envelope expects Version=1, Resource="drives",
		// Instance="default"; the pre-MON-5371 shape supplied none of those,
		// so ReadResource classifies it as ErrCacheMiss.
		path := filepath.Join(c.loc.Root, c.loc.InstanceKey, drivesResource+".json")
		testutil.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		oldShape := `{"cached_at":"2026-01-01T00:00:00Z","ttl_hours":24,"drives":[{"id":"d1","name":"x"}]}`
		testutil.NoError(t, os.WriteFile(path, []byte(oldShape), 0o600))

		drives, err := c.GetDrives()
		testutil.NoError(t, err)
		testutil.Nil(t, drives)
	})
}

func TestCache_Clear(t *testing.T) {
	hermetic(t)
	c, err := New()
	testutil.NoError(t, err)

	testutil.NoError(t, c.SetDrives([]*CachedDrive{{ID: "test", Name: "Test"}}))

	testutil.NoError(t, c.Clear())
	// Clear scoped to <Root>/<InstanceKey>, not the tool-level Root.
	_, err = os.Stat(filepath.Join(c.loc.Root, c.loc.InstanceKey))
	testutil.True(t, os.IsNotExist(err))
}

func TestCache_GetDir(t *testing.T) {
	hermetic(t)
	c, err := New()
	testutil.NoError(t, err)
	defer c.Clear()

	want, err := config.CacheDirPath()
	testutil.NoError(t, err)
	testutil.NotEmpty(t, c.GetDir())
	testutil.Equal(t, c.GetDir(), want)
}

func TestMigrateLegacyCacheDir(t *testing.T) {
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

		c, err := New()
		testutil.NoError(t, err)
		defer c.Clear()

		got, err := os.ReadFile(filepath.Join(newDir, c.loc.InstanceKey, DrivesFile))
		testutil.NoError(t, err)
		testutil.Contains(t, string(got), `"d1"`)
		_, statErr := os.Stat(legacy)
		testutil.True(t, os.IsNotExist(statErr))

		// Idempotent: a second New is a no-op and keeps the new cache.
		c2, err := New()
		testutil.NoError(t, err)
		defer c2.Clear()
		_, err = os.Stat(filepath.Join(newDir, c.loc.InstanceKey, DrivesFile))
		testutil.NoError(t, err)
	})

	t.Run("does not overwrite an existing new cache; still removes legacy", func(t *testing.T) {
		legacy, _ := setup(t)
		c, err := New()
		testutil.NoError(t, err)
		defer c.Clear()
		testutil.NoError(t, c.SetDrives([]*CachedDrive{{ID: "new", Name: "Keep"}}))
		seedLegacy(t, legacy, `{"drives":[{"id":"old","name":"Stale"}]}`)

		c2, err := New()
		testutil.NoError(t, err)
		defer c2.Clear()

		drives, err := c2.GetDrives()
		testutil.NoError(t, err)
		testutil.Len(t, drives, 1)
		testutil.Equal(t, drives[0].ID, "new")
		_, statErr := os.Stat(legacy)
		testutil.True(t, os.IsNotExist(statErr))
	})

	t.Run("unreadable legacy drives file: legacy preserved, no partial carry", func(t *testing.T) {
		legacy, newDir := setup(t)
		// drives.json as a directory => os.ReadFile errors with a
		// non-IsNotExist error => carry fails => legacy must NOT be removed
		// and nothing must be written into the new cache.
		testutil.NoError(t, os.MkdirAll(filepath.Join(legacy, DrivesFile), 0o700))

		c, err := New()
		testutil.NoError(t, err)
		defer c.Clear()

		_, statErr := os.Stat(legacy)
		testutil.NoError(t, statErr)
		_, newErr := os.Stat(filepath.Join(newDir, c.loc.InstanceKey, DrivesFile))
		testutil.True(t, os.IsNotExist(newErr))
	})

	t.Run("no legacy is a clean no-op", func(t *testing.T) {
		setup(t)
		c, err := New()
		testutil.NoError(t, err)
		defer c.Clear()
	})

	t.Run("old-hand-rolled legacy cache subdir is carried (pre-MON-5371 install)", func(t *testing.T) {
		hermetic(t)
		oldLegacy, err := config.OldHandRolledLegacyCacheDir()
		testutil.NoError(t, err)
		newDir, err := config.CacheDirPath()
		testutil.NoError(t, err)
		// On Linux this is the same path as LegacyCacheDir — dedup handles it.
		// On macOS/Windows this is the path a pure pre-MON-5371 install lived
		// at and is what the dual-probe needs to find.
		testutil.NoError(t, os.MkdirAll(oldLegacy, 0o700))
		testutil.NoError(t, os.WriteFile(filepath.Join(oldLegacy, DrivesFile),
			[]byte(`{"cached_at":"2026-01-01T00:00:00Z","ttl_hours":24,"drives":[{"id":"hand","name":"R"}]}`), 0o600))

		c, err := New()
		testutil.NoError(t, err)
		defer c.Clear()

		got, err := os.ReadFile(filepath.Join(newDir, c.loc.InstanceKey, DrivesFile))
		testutil.NoError(t, err)
		testutil.Contains(t, string(got), `"hand"`)
		_, statErr := os.Stat(oldLegacy)
		testutil.True(t, os.IsNotExist(statErr))
	})
}
