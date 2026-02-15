package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGetConfigDir(t *testing.T) {
	t.Run("uses XDG_CONFIG_HOME if set", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		dir, err := GetConfigDir()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dir != filepath.Join(tmpDir, DirName) {
			t.Errorf("got %v, want %v", dir, filepath.Join(tmpDir, DirName))
		}

		// Verify directory was created
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !info.IsDir() {
			t.Error("got false, want true")
		}
	})

	t.Run("uses ~/.config if XDG_CONFIG_HOME not set", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")

		dir, err := GetConfigDir()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".config", DirName)
		if dir != expected {
			t.Errorf("got %v, want %v", dir, expected)
		}
	})

	t.Run("creates directory with correct permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		dir, err := GetConfigDir()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Mode().Perm() != os.FileMode(0700) {
			t.Errorf("got %v, want %v", info.Mode().Perm(), os.FileMode(0700))
		}
	})
}

func TestGetCredentialsPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	path, err := GetCredentialsPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != filepath.Join(tmpDir, DirName, CredentialsFile) {
		t.Errorf("got %v, want %v", path, filepath.Join(tmpDir, DirName, CredentialsFile))
	}
}

func TestGetTokenPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	path, err := GetTokenPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != filepath.Join(tmpDir, DirName, TokenFile) {
		t.Errorf("got %v, want %v", path, filepath.Join(tmpDir, DirName, TokenFile))
	}
}

func TestShortenPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	if DirName != "google-readonly" {
		t.Errorf("got %v, want %v", DirName, "google-readonly")
	}
	if CredentialsFile != "credentials.json" {
		t.Errorf("got %v, want %v", CredentialsFile, "credentials.json")
	}
	if TokenFile != "token.json" {
		t.Errorf("got %v, want %v", TokenFile, "token.json")
	}
	if ConfigFile != "config.json" {
		t.Errorf("got %v, want %v", ConfigFile, "config.json")
	}
	if DefaultCacheTTLHours != 24 {
		t.Errorf("got %v, want %v", DefaultCacheTTLHours, 24)
	}
}

func TestGetConfigPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	path, err := GetConfigPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != filepath.Join(tmpDir, DirName, ConfigFile) {
		t.Errorf("got %v, want %v", path, filepath.Join(tmpDir, DirName, ConfigFile))
	}
}

func TestLoadConfig(t *testing.T) {
	t.Run("returns default config when file does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.CacheTTLHours != DefaultCacheTTLHours {
			t.Errorf("got %v, want %v", cfg.CacheTTLHours, DefaultCacheTTLHours)
		}
	})

	t.Run("loads config from file", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		// Create config directory and file
		configDir := filepath.Join(tmpDir, DirName)
		if err := os.MkdirAll(configDir, DirPerm); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		configData := `{"cache_ttl_hours": 48}`
		if err := os.WriteFile(filepath.Join(configDir, ConfigFile), []byte(configData), TokenPerm); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.CacheTTLHours != 48 {
			t.Errorf("got %v, want %v", cfg.CacheTTLHours, 48)
		}
	})

	t.Run("applies default for zero or negative TTL", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		configDir := filepath.Join(tmpDir, DirName)
		if err := os.MkdirAll(configDir, DirPerm); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		configData := `{"cache_ttl_hours": 0}`
		if err := os.WriteFile(filepath.Join(configDir, ConfigFile), []byte(configData), TokenPerm); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.CacheTTLHours != DefaultCacheTTLHours {
			t.Errorf("got %v, want %v", cfg.CacheTTLHours, DefaultCacheTTLHours)
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		configDir := filepath.Join(tmpDir, DirName)
		if err := os.MkdirAll(configDir, DirPerm); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := os.WriteFile(filepath.Join(configDir, ConfigFile), []byte("not json"), TokenPerm); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err := LoadConfig()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestSaveConfig(t *testing.T) {
	t.Run("saves config to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		cfg := &Config{CacheTTLHours: 12}
		err := SaveConfig(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify file was created
		path, _ := GetConfigPath()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(string(data), `"cache_ttl_hours": 12`) {
			t.Errorf("expected %q to contain %q", string(data), `"cache_ttl_hours": 12`)
		}
	})

	t.Run("overwrites existing config", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		// Save initial config
		cfg1 := &Config{CacheTTLHours: 12}
		if err := SaveConfig(cfg1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Save new config
		cfg2 := &Config{CacheTTLHours: 36}
		if err := SaveConfig(cfg2); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify new value
		loaded, err := LoadConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if loaded.CacheTTLHours != 36 {
			t.Errorf("got %v, want %v", loaded.CacheTTLHours, 36)
		}
	})
}

func TestGetCacheTTL(t *testing.T) {
	t.Run("returns configured TTL as duration", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		cfg := &Config{CacheTTLHours: 12}
		if err := SaveConfig(cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		ttl := GetCacheTTL()
		if ttl != 12*time.Hour {
			t.Errorf("got %v, want %v", ttl, 12*time.Hour)
		}
	})

	t.Run("returns default TTL when no config exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		ttl := GetCacheTTL()
		if ttl != time.Duration(DefaultCacheTTLHours)*time.Hour {
			t.Errorf("got %v, want %v", ttl, time.Duration(DefaultCacheTTLHours)*time.Hour)
		}
	})
}

func TestGetCacheTTLHours(t *testing.T) {
	t.Run("returns configured TTL in hours", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		cfg := &Config{CacheTTLHours: 48}
		if err := SaveConfig(cfg); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		hours := GetCacheTTLHours()
		if hours != 48 {
			t.Errorf("got %v, want %v", hours, 48)
		}
	})

	t.Run("returns default TTL when no config exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		hours := GetCacheTTLHours()
		if hours != DefaultCacheTTLHours {
			t.Errorf("got %v, want %v", hours, DefaultCacheTTLHours)
		}
	})
}
