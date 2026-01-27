// Package config provides centralized configuration management for the application.
// It has no external dependencies to avoid circular imports with other internal packages.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	// DirName is the name of the configuration directory
	DirName = "google-readonly"
	// CredentialsFile is the name of the OAuth credentials file
	CredentialsFile = "credentials.json"
	// TokenFile is the name of the OAuth token file (fallback storage)
	TokenFile = "token.json"
)

// File and directory permission constants for consistent security settings.
const (
	// DirPerm is the permission for config directories (owner read/write/execute only)
	DirPerm = 0700
	// TokenPerm is the permission for token files (owner read/write only)
	TokenPerm = 0600
	// OutputDirPerm is the permission for output directories (owner all, group/other read/execute)
	OutputDirPerm = 0755
	// OutputFilePerm is the permission for output files (owner read/write, group/other read)
	OutputFilePerm = 0644
)

// GetConfigDir returns the configuration directory path, creating it if needed.
// Uses XDG_CONFIG_HOME if set, otherwise ~/.config/google-readonly
func GetConfigDir() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configHome = filepath.Join(home, ".config")
	}
	configDir := filepath.Join(configHome, DirName)

	if err := os.MkdirAll(configDir, DirPerm); err != nil {
		return "", err
	}

	return configDir, nil
}

// GetCredentialsPath returns the full path to credentials.json
func GetCredentialsPath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, CredentialsFile), nil
}

// GetTokenPath returns the full path to token.json (fallback storage)
func GetTokenPath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, TokenFile), nil
}

// ShortenPath replaces the home directory prefix with ~ for display purposes.
// This prevents exposing full paths including usernames in error messages.
func ShortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if len(path) >= len(home) && path[:len(home)] == home {
		return "~" + path[len(home):]
	}
	return path
}

const (
	// ConfigFile is the name of the user configuration file
	ConfigFile = "config.json"
	// DefaultCacheTTLHours is the default cache TTL in hours
	DefaultCacheTTLHours = 24
)

// Config represents user-configurable settings
type Config struct {
	CacheTTLHours int `json:"cache_ttl_hours"`
}

// GetConfigPath returns the full path to config.json
func GetConfigPath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ConfigFile), nil
}

// LoadConfig loads the user configuration from config.json
// Returns default config if file doesn't exist
func LoadConfig() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config
			return &Config{
				CacheTTLHours: DefaultCacheTTLHours,
			}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Apply defaults for unset values
	if cfg.CacheTTLHours <= 0 {
		cfg.CacheTTLHours = DefaultCacheTTLHours
	}

	return &cfg, nil
}

// SaveConfig saves the user configuration to config.json
func SaveConfig(cfg *Config) error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, TokenPerm)
}

// GetCacheTTL returns the configured cache TTL duration
func GetCacheTTL() time.Duration {
	cfg, err := LoadConfig()
	if err != nil {
		return time.Duration(DefaultCacheTTLHours) * time.Hour
	}
	return time.Duration(cfg.CacheTTLHours) * time.Hour
}

// GetCacheTTLHours returns the configured cache TTL in hours
func GetCacheTTLHours() int {
	cfg, err := LoadConfig()
	if err != nil {
		return DefaultCacheTTLHours
	}
	return cfg.CacheTTLHours
}
