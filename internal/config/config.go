// Package config provides centralized configuration management for the application.
// It has no external dependencies to avoid circular imports with other internal packages.
package config

import (
	"os"
	"path/filepath"
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
