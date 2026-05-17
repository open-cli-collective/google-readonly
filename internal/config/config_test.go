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
	t.Parallel()
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
			t.Parallel()
			result := ShortenPath(tt.input)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	t.Parallel()
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
	if path != filepath.Join(tmpDir, DirName, ConfigFileYAML) {
		t.Errorf("got %v, want %v", path, filepath.Join(tmpDir, DirName, ConfigFileYAML))
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

	t.Run("config.yml wins over legacy config.json when both present", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)
		dir := filepath.Join(tmpDir, DirName)
		if err := os.MkdirAll(dir, DirPerm); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ConfigFile), []byte(`{"cache_ttl_hours": 99}`), TokenPerm); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ConfigFileYAML), []byte("cache_ttl_hours: 7\n"), TokenPerm); err != nil {
			t.Fatal(err)
		}
		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.CacheTTLHours != 7 {
			t.Errorf("config.yml must win: got %v, want 7", cfg.CacheTTLHours)
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
		if !strings.Contains(string(data), "cache_ttl_hours: 12") {
			t.Errorf("expected %q to contain %q", string(data), "cache_ttl_hours: 12")
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

func TestCacheDirResolvers(t *testing.T) {
	// Each subtest gets its OWN hermetic env + fresh temp dir: GetCacheDir is
	// creating, so a shared env would let one subtest's mkdir defeat
	// another's "non-creating" assertion.
	hermeticCfg := func(t *testing.T) string {
		t.Helper()
		d := t.TempDir()
		t.Setenv("HOME", d)
		t.Setenv("XDG_CACHE_HOME", filepath.Join(d, "xcache"))
		t.Setenv("LOCALAPPDATA", filepath.Join(d, "localappdata"))
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(d, "xconfig"))
		return d
	}

	t.Run("GetCacheDir is under os.UserCacheDir()/DirName, not the config tree", func(t *testing.T) {
		hermeticCfg(t)
		base, err := os.UserCacheDir() // computed in-test — no hard-coded per-OS strings
		if err != nil {
			t.Fatalf("UserCacheDir: %v", err)
		}
		want := filepath.Join(base, DirName)

		got, err := GetCacheDir()
		if err != nil {
			t.Fatalf("GetCacheDir: %v", err)
		}
		if got != want {
			t.Errorf("GetCacheDir = %q, want %q", got, want)
		}
		cfgDir, err := configDirPath()
		if err != nil {
			t.Fatalf("configDirPath: %v", err)
		}
		if strings.HasPrefix(got, cfgDir) {
			t.Errorf("cache dir %q must not be under the config dir %q", got, cfgDir)
		}
		if info, serr := os.Stat(got); serr != nil || !info.IsDir() {
			t.Errorf("GetCacheDir must create the dir: stat err=%v", serr)
		}
	})

	t.Run("CacheDirPath and LegacyCacheDir are non-creating", func(t *testing.T) {
		d := hermeticCfg(t)
		cp, err := CacheDirPath()
		if err != nil {
			t.Fatalf("CacheDirPath: %v", err)
		}
		if _, serr := os.Stat(cp); !os.IsNotExist(serr) {
			t.Errorf("CacheDirPath must not create %q (stat err=%v)", cp, serr)
		}
		lp, err := LegacyCacheDir()
		if err != nil {
			t.Fatalf("LegacyCacheDir: %v", err)
		}
		if _, serr := os.Stat(lp); !os.IsNotExist(serr) {
			t.Errorf("LegacyCacheDir must not create %q (stat err=%v)", lp, serr)
		}
		if want := filepath.Join(filepath.Join(d, "xconfig"), DirName, legacyCacheSubdir); lp != want {
			t.Errorf("LegacyCacheDir = %q, want %q", lp, want)
		}

		gp, err := GetConfigPathNoCreate()
		if err != nil {
			t.Fatalf("GetConfigPathNoCreate: %v", err)
		}
		if _, serr := os.Stat(filepath.Dir(gp)); !os.IsNotExist(serr) {
			t.Errorf("GetConfigPathNoCreate must not create the config dir %q (stat err=%v)", filepath.Dir(gp), serr)
		}
		if want := filepath.Join(filepath.Join(d, "xconfig"), DirName, ConfigFileYAML); gp != want {
			t.Errorf("GetConfigPathNoCreate = %q, want %q", gp, want)
		}
	})

	t.Run("GetConfigDir still creates", func(t *testing.T) {
		hermeticCfg(t)
		cd, err := GetConfigDir()
		if err != nil {
			t.Fatalf("GetConfigDir: %v", err)
		}
		if info, serr := os.Stat(cd); serr != nil || !info.IsDir() {
			t.Errorf("GetConfigDir must create the dir: stat err=%v", serr)
		}
	})
}
