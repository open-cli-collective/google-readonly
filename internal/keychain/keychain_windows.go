//go:build windows

package keychain

import (
	"golang.org/x/oauth2"
)

// Windows does not have native secure storage support like macOS Keychain
// or Linux secret-tool. All token operations fall back to config file storage.

// getToken retrieves the OAuth token from Windows storage (config file only)
func getToken() (*oauth2.Token, error) {
	return getFromConfigFile()
}

// setToken stores the OAuth token in Windows storage (config file only)
func setToken(token *oauth2.Token) error {
	return setInConfigFile(token)
}

// deleteToken removes the OAuth token from storage
func deleteToken() error {
	return deleteFromConfigFile()
}

// getStorageBackend returns the current storage backend (always file on Windows)
func getStorageBackend() StorageBackend {
	_, err := getFromConfigFile()
	if err == nil {
		return BackendFile
	}
	return BackendFile
}
