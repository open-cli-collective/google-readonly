package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfigDir(t *testing.T) {
	t.Run("uses XDG_CONFIG_HOME if set", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		dir, err := GetConfigDir()
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(tmpDir, DirName), dir)

		// Verify directory was created
		info, err := os.Stat(dir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("uses ~/.config if XDG_CONFIG_HOME not set", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")

		dir, err := GetConfigDir()
		require.NoError(t, err)

		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".config", DirName)
		assert.Equal(t, expected, dir)
	})

	t.Run("creates directory with correct permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		dir, err := GetConfigDir()
		require.NoError(t, err)

		info, err := os.Stat(dir)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0700), info.Mode().Perm())
	})
}

func TestGetCredentialsPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	path, err := GetCredentialsPath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmpDir, DirName, CredentialsFile), path)
}

func TestGetTokenPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	path, err := GetTokenPath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmpDir, DirName, TokenFile), path)
}

func TestShortenPath(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "replaces home directory with tilde",
			input:    filepath.Join(home, ".config", "google-readonly", "credentials.json"),
			expected: "~/.config/google-readonly/credentials.json",
		},
		{
			name:     "replaces home directory only",
			input:    home,
			expected: "~",
		},
		{
			name:     "preserves path not under home",
			input:    "/tmp/test/file.txt",
			expected: "/tmp/test/file.txt",
		},
		{
			name:     "preserves relative path",
			input:    "relative/path/file.txt",
			expected: "relative/path/file.txt",
		},
		{
			name:     "handles path that starts with home prefix but is different",
			input:    home + "extra/path",
			expected: "~extra/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShortenPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "google-readonly", DirName)
	assert.Equal(t, "credentials.json", CredentialsFile)
	assert.Equal(t, "token.json", TokenFile)
	assert.Equal(t, "config.json", ConfigFile)
	assert.Equal(t, 24, DefaultCacheTTLHours)
}

func TestGetConfigPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	path, err := GetConfigPath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmpDir, DirName, ConfigFile), path)
}

func TestLoadConfig(t *testing.T) {
	t.Run("returns default config when file does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		cfg, err := LoadConfig()
		require.NoError(t, err)
		assert.Equal(t, DefaultCacheTTLHours, cfg.CacheTTLHours)
	})

	t.Run("loads config from file", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		// Create config directory and file
		configDir := filepath.Join(tmpDir, DirName)
		require.NoError(t, os.MkdirAll(configDir, DirPerm))

		configData := `{"cache_ttl_hours": 48}`
		require.NoError(t, os.WriteFile(filepath.Join(configDir, ConfigFile), []byte(configData), TokenPerm))

		cfg, err := LoadConfig()
		require.NoError(t, err)
		assert.Equal(t, 48, cfg.CacheTTLHours)
	})

	t.Run("applies default for zero or negative TTL", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		configDir := filepath.Join(tmpDir, DirName)
		require.NoError(t, os.MkdirAll(configDir, DirPerm))

		configData := `{"cache_ttl_hours": 0}`
		require.NoError(t, os.WriteFile(filepath.Join(configDir, ConfigFile), []byte(configData), TokenPerm))

		cfg, err := LoadConfig()
		require.NoError(t, err)
		assert.Equal(t, DefaultCacheTTLHours, cfg.CacheTTLHours)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		configDir := filepath.Join(tmpDir, DirName)
		require.NoError(t, os.MkdirAll(configDir, DirPerm))

		require.NoError(t, os.WriteFile(filepath.Join(configDir, ConfigFile), []byte("not json"), TokenPerm))

		_, err := LoadConfig()
		assert.Error(t, err)
	})
}

func TestSaveConfig(t *testing.T) {
	t.Run("saves config to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		cfg := &Config{CacheTTLHours: 12}
		err := SaveConfig(cfg)
		require.NoError(t, err)

		// Verify file was created
		path, _ := GetConfigPath()
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"cache_ttl_hours": 12`)
	})

	t.Run("overwrites existing config", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		// Save initial config
		cfg1 := &Config{CacheTTLHours: 12}
		require.NoError(t, SaveConfig(cfg1))

		// Save new config
		cfg2 := &Config{CacheTTLHours: 36}
		require.NoError(t, SaveConfig(cfg2))

		// Verify new value
		loaded, err := LoadConfig()
		require.NoError(t, err)
		assert.Equal(t, 36, loaded.CacheTTLHours)
	})
}

func TestGetCacheTTL(t *testing.T) {
	t.Run("returns configured TTL as duration", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		cfg := &Config{CacheTTLHours: 12}
		require.NoError(t, SaveConfig(cfg))

		ttl := GetCacheTTL()
		assert.Equal(t, 12*time.Hour, ttl)
	})

	t.Run("returns default TTL when no config exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		ttl := GetCacheTTL()
		assert.Equal(t, time.Duration(DefaultCacheTTLHours)*time.Hour, ttl)
	})
}

func TestGetCacheTTLHours(t *testing.T) {
	t.Run("returns configured TTL in hours", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		cfg := &Config{CacheTTLHours: 48}
		require.NoError(t, SaveConfig(cfg))

		hours := GetCacheTTLHours()
		assert.Equal(t, 48, hours)
	})

	t.Run("returns default TTL when no config exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		hours := GetCacheTTLHours()
		assert.Equal(t, DefaultCacheTTLHours, hours)
	})
}
