package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-cli-collective/cli-common/statedirtest"
)

// hermeticConfig isolates the §3.1 7-var env set so os.UserConfigDir /
// os.UserCacheDir never resolve to the developer's real dirs. Helper-local so
// these in-package tests stay independent of the credtest leaf used by tests
// in sibling packages.
func hermeticConfig(t *testing.T) string {
	t.Helper()
	return statedirtest.Hermetic(t)
}

func TestGetConfigDir(t *testing.T) {
	t.Run("creates the resolved config dir", func(t *testing.T) {
		hermeticConfig(t)
		base, err := os.UserConfigDir()
		if err != nil {
			t.Fatalf("UserConfigDir: %v", err)
		}
		want := filepath.Join(base, DirName)

		dir, err := GetConfigDir()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dir != want {
			t.Errorf("got %v, want %v", dir, want)
		}

		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !info.IsDir() {
			t.Error("got false, want true")
		}
	})

	t.Run("creates directory with correct permissions", func(t *testing.T) {
		hermeticConfig(t)

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
	hermeticConfig(t)
	base, _ := os.UserConfigDir()

	path, err := GetCredentialsPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(base, DirName, CredentialsFile)
	if path != want {
		t.Errorf("got %v, want %v", path, want)
	}
}

func TestGetTokenPath(t *testing.T) {
	hermeticConfig(t)
	base, _ := os.UserConfigDir()

	path, err := GetTokenPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(base, DirName, TokenFile)
	if path != want {
		t.Errorf("got %v, want %v", path, want)
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
}

func TestGetConfigPath(t *testing.T) {
	hermeticConfig(t)
	base, _ := os.UserConfigDir()

	path, err := GetConfigPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(base, DirName, ConfigFileYAML)
	if path != want {
		t.Errorf("got %v, want %v", path, want)
	}
}

func TestLoadConfig(t *testing.T) {
	t.Run("returns default config when file does not exist", func(t *testing.T) {
		hermeticConfig(t)

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.CredentialRef != DefaultCredentialRef {
			t.Errorf("got %v, want %v", cfg.CredentialRef, DefaultCredentialRef)
		}
	})

	t.Run("config.yml wins over legacy config.json when both present", func(t *testing.T) {
		hermeticConfig(t)
		dir, err := GetConfigDir()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ConfigFile), []byte(`{"credential_ref":"google-readonly/legacy"}`), TokenPerm); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, ConfigFileYAML), []byte("credential_ref: google-readonly/yml\n"), TokenPerm); err != nil {
			t.Fatal(err)
		}
		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.CredentialRef != "google-readonly/yml" {
			t.Errorf("config.yml must win: got %v, want google-readonly/yml", cfg.CredentialRef)
		}
	})

	t.Run("loads config from legacy JSON", func(t *testing.T) {
		hermeticConfig(t)
		dir, err := GetConfigDir()
		if err != nil {
			t.Fatal(err)
		}

		configData := `{"credential_ref":"google-readonly/from-json"}`
		if err := os.WriteFile(filepath.Join(dir, ConfigFile), []byte(configData), TokenPerm); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.CredentialRef != "google-readonly/from-json" {
			t.Errorf("got %v, want google-readonly/from-json", cfg.CredentialRef)
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		hermeticConfig(t)
		dir, err := GetConfigDir()
		if err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(filepath.Join(dir, ConfigFile), []byte("not json"), TokenPerm); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = LoadConfig()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestSaveConfig(t *testing.T) {
	t.Run("saves config to file", func(t *testing.T) {
		hermeticConfig(t)

		cfg := &Config{CredentialRef: "google-readonly/test"}
		err := SaveConfig(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		path, _ := GetConfigPath()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(string(data), "credential_ref: google-readonly/test") {
			t.Errorf("expected %q to contain credential_ref", string(data))
		}
	})

	t.Run("overwrites existing config", func(t *testing.T) {
		hermeticConfig(t)

		cfg1 := &Config{CredentialRef: "google-readonly/a"}
		if err := SaveConfig(cfg1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cfg2 := &Config{CredentialRef: "google-readonly/b"}
		if err := SaveConfig(cfg2); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		loaded, err := LoadConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if loaded.CredentialRef != "google-readonly/b" {
			t.Errorf("got %v, want google-readonly/b", loaded.CredentialRef)
		}
	})

	t.Run("writes atomically with correct perms and no stale tmp", func(t *testing.T) {
		hermeticConfig(t)
		if err := SaveConfig(&Config{}); err != nil {
			t.Fatalf("save: %v", err)
		}
		final, _ := GetConfigPath()
		info, err := os.Stat(final)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if info.Mode().Perm() != TokenPerm {
			t.Errorf("config file mode = %v, want %v", info.Mode().Perm(), TokenPerm)
		}
		// No stale *.tmp in the dir after a successful save.
		dir := filepath.Dir(final)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("readdir: %v", err)
		}
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".tmp") {
				t.Errorf("stale temp file left in %s: %s", dir, e.Name())
			}
		}
	})
}

func TestCacheDirResolvers(t *testing.T) {
	t.Run("GetCacheDir is under os.UserCacheDir()/DirName, not the config tree", func(t *testing.T) {
		hermeticConfig(t)
		base, err := os.UserCacheDir()
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
		hermeticConfig(t)
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
		// Expected legacy cache dir is computed from the resolver — no
		// hard-coded per-OS paths. LegacyCacheDir = configDirPath() + "/cache".
		cfgDir, err := configDirPath()
		if err != nil {
			t.Fatalf("configDirPath: %v", err)
		}
		if want := filepath.Join(cfgDir, legacyCacheSubdir); lp != want {
			t.Errorf("LegacyCacheDir = %q, want %q", lp, want)
		}

		gp, err := GetConfigPathNoCreate()
		if err != nil {
			t.Fatalf("GetConfigPathNoCreate: %v", err)
		}
		if _, serr := os.Stat(filepath.Dir(gp)); !os.IsNotExist(serr) {
			t.Errorf("GetConfigPathNoCreate must not create the config dir %q (stat err=%v)", filepath.Dir(gp), serr)
		}
		if want := filepath.Join(cfgDir, ConfigFileYAML); gp != want {
			t.Errorf("GetConfigPathNoCreate = %q, want %q", gp, want)
		}
	})

	t.Run("GetConfigDir still creates", func(t *testing.T) {
		hermeticConfig(t)
		cd, err := GetConfigDir()
		if err != nil {
			t.Fatalf("GetConfigDir: %v", err)
		}
		if info, serr := os.Stat(cd); serr != nil || !info.IsDir() {
			t.Errorf("GetConfigDir must create the dir: stat err=%v", serr)
		}
	})
}
