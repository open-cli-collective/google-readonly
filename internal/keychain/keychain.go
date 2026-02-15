// Package keychain provides secure storage for OAuth tokens using platform-native
// secure storage mechanisms (macOS Keychain, Linux secret-tool) with file fallback.
package keychain

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"

	"github.com/open-cli-collective/google-readonly/internal/config"
)

const (
	serviceName = config.DirName
	tokenKey    = "oauth_token" //nolint:gosec // Not a credential; key name for keychain lookup
)

// StorageBackend represents where tokens are stored
type StorageBackend string

// StorageBackend constants define where OAuth tokens are persisted.
const (
	BackendKeychain   StorageBackend = "Keychain"    // macOS Keychain
	BackendSecretTool StorageBackend = "secret-tool" // Linux libsecret
	BackendFile       StorageBackend = "config file" // File fallback
)

var (
	// ErrTokenNotFound indicates no token exists in storage
	ErrTokenNotFound = errors.New("no token found in secure storage")
)

// tokenFilePath returns the full path to the token file
func tokenFilePath() (string, error) {
	return config.GetTokenPath()
}

// GetToken retrieves the OAuth token from secure storage
func GetToken() (*oauth2.Token, error) {
	return getToken()
}

// SetToken stores the OAuth token in secure storage
func SetToken(token *oauth2.Token) error {
	return setToken(token)
}

// DeleteToken removes the OAuth token from secure storage
func DeleteToken() error {
	return deleteToken()
}

// HasStoredToken returns true if a token exists in secure storage
func HasStoredToken() bool {
	_, err := GetToken()
	return err == nil
}

// GetStorageBackend returns the current storage backend being used
func GetStorageBackend() StorageBackend {
	return getStorageBackend()
}

// IsSecureStorage returns true if using secure storage (keychain/secret-tool)
func IsSecureStorage() bool {
	backend := GetStorageBackend()
	return backend == BackendKeychain || backend == BackendSecretTool
}

// MigrateFromFile migrates token.json to secure storage if it exists
func MigrateFromFile(tokenFilePath string) error {
	// Check if token file exists
	if _, err := os.Stat(tokenFilePath); os.IsNotExist(err) {
		return nil // Nothing to migrate
	}

	// Check if already migrated (token in secure storage)
	if IsSecureStorage() && HasStoredToken() {
		return nil // Already migrated
	}

	// Read token from file
	f, err := os.Open(tokenFilePath) //nolint:gosec // Path from user config directory
	if err != nil {
		return fmt.Errorf("opening token file: %w", err)
	}
	defer f.Close()

	var token oauth2.Token
	if err := json.NewDecoder(f).Decode(&token); err != nil {
		return fmt.Errorf("parsing token file: %w", err)
	}

	// Store in secure storage
	if err := SetToken(&token); err != nil {
		return fmt.Errorf("storing token in secure storage: %w", err)
	}

	// Securely delete old token file (overwrite with zeros before removal)
	if err := secureDelete(tokenFilePath); err != nil {
		// Non-fatal - token is now in secure storage
		fmt.Fprintf(os.Stderr, "Warning: could not securely delete old token file: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "Migrated token to secure storage. Old token file securely deleted.\n")
	}

	return nil
}

// secureDelete overwrites a file with zeros before deleting it to prevent
// forensic recovery of sensitive data.
func secureDelete(path string) error {
	// Get file size
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Already gone
		}
		return err
	}

	// Overwrite with zeros
	f, err := os.OpenFile(path, os.O_WRONLY, 0) //nolint:gosec // Path from user config directory
	if err != nil {
		// If we can't open for writing, try to delete anyway
		return os.Remove(path)
	}

	zeros := make([]byte, info.Size())
	_, _ = f.Write(zeros) // Best effort overwrite
	_ = f.Sync()          // Flush to disk
	_ = f.Close()         // Ignore close error, we're deleting anyway

	return os.Remove(path)
}

// File-based storage implementation (fallback)

func getFromConfigFile() (*oauth2.Token, error) {
	path, err := tokenFilePath()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path) //nolint:gosec // Path from user config directory
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("opening token file: %w", err)
	}
	defer f.Close()

	var token oauth2.Token
	if err := json.NewDecoder(f).Decode(&token); err != nil {
		return nil, fmt.Errorf("parsing token file: %w", err)
	}

	return &token, nil
}

func setInConfigFile(token *oauth2.Token) error {
	path, err := tokenFilePath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, config.DirPerm); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Write token with restricted permissions
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, config.TokenPerm) //nolint:gosec // Path from user config directory
	if err != nil {
		return fmt.Errorf("creating token file: %w", err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(token); err != nil {
		return fmt.Errorf("writing token: %w", err)
	}

	return nil
}

func deleteFromConfigFile() error {
	path, err := tokenFilePath()
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting token file: %w", err)
	}

	return nil
}
