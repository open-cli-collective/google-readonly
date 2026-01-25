//go:build darwin

package keychain

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"golang.org/x/oauth2"
)

// getToken retrieves the OAuth token from macOS Keychain
func getToken() (*oauth2.Token, error) {
	token, err := getFromKeychain()
	if err == nil {
		return token, nil
	}

	// Fall back to config file
	return getFromConfigFile()
}

// setToken stores the OAuth token in macOS Keychain
func setToken(token *oauth2.Token) error {
	err := setInKeychain(token)
	if err == nil {
		return nil
	}

	// Fall back to config file
	fmt.Printf("Warning: keychain storage failed, using config file: %v\n", err)
	return setInConfigFile(token)
}

// deleteToken removes the OAuth token from storage
func deleteToken() error {
	// Try to delete from both keychain and file
	keychainErr := deleteFromKeychain()
	fileErr := deleteFromConfigFile()

	// Return keychain error if both fail, otherwise nil
	if keychainErr != nil && fileErr != nil {
		return keychainErr
	}
	return nil
}

// getStorageBackend returns the current storage backend
func getStorageBackend() StorageBackend {
	// Check if token exists in keychain
	_, err := getFromKeychain()
	if err == nil {
		return BackendKeychain
	}

	// Check if token exists in file
	_, err = getFromConfigFile()
	if err == nil {
		return BackendFile
	}

	// Default to keychain (preferred backend)
	return BackendKeychain
}

// macOS Keychain implementation using security CLI

func getFromKeychain() (*oauth2.Token, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-s", serviceName,
		"-a", tokenKey,
		"-w")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read from keychain: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(output))), &token); err != nil {
		return nil, fmt.Errorf("failed to parse token from keychain: %w", err)
	}

	return &token, nil
}

func setInKeychain(token *oauth2.Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to serialize token: %w", err)
	}

	// Delete existing entry (ignore error if not exists)
	_ = deleteFromKeychain()

	// Add new entry using interactive mode to avoid exposing token in process list.
	// The -i flag reads commands from stdin, keeping sensitive data out of ps output.
	cmd := exec.Command("security", "-i")

	// Build the command to send via stdin
	// Note: The password value is quoted to handle special characters in JSON
	stdinCmd := fmt.Sprintf("add-generic-password -s %q -a %q -w %q -U\n",
		serviceName, tokenKey, string(data))
	cmd.Stdin = strings.NewReader(stdinCmd)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to store in keychain: %w", err)
	}

	return nil
}

func deleteFromKeychain() error {
	cmd := exec.Command("security", "delete-generic-password",
		"-s", serviceName,
		"-a", tokenKey)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete from keychain: %w", err)
	}

	return nil
}
