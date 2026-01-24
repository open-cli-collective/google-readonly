//go:build linux

package keychain

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"golang.org/x/oauth2"
)

// isSecretToolAvailable checks if secret-tool is installed
func isSecretToolAvailable() bool {
	_, err := exec.LookPath("secret-tool")
	return err == nil
}

// getToken retrieves the OAuth token from Linux secret storage
func getToken() (*oauth2.Token, error) {
	if isSecretToolAvailable() {
		token, err := getFromSecretTool()
		if err == nil {
			return token, nil
		}
	}

	// Fall back to config file
	return getFromConfigFile()
}

// setToken stores the OAuth token in Linux secret storage
func setToken(token *oauth2.Token) error {
	if isSecretToolAvailable() {
		err := setInSecretTool(token)
		if err == nil {
			return nil
		}
		fmt.Printf("Warning: secret-tool storage failed, using config file: %v\n", err)
	}

	// Fall back to config file
	return setInConfigFile(token)
}

// deleteToken removes the OAuth token from storage
func deleteToken() error {
	var secretErr, fileErr error

	if isSecretToolAvailable() {
		secretErr = deleteFromSecretTool()
	}
	fileErr = deleteFromConfigFile()

	// Return secret-tool error if both fail, otherwise nil
	if secretErr != nil && fileErr != nil {
		return secretErr
	}
	return nil
}

// getStorageBackend returns the current storage backend
func getStorageBackend() StorageBackend {
	if isSecretToolAvailable() {
		// Check if token exists in secret-tool
		_, err := getFromSecretTool()
		if err == nil {
			return BackendSecretTool
		}
	}

	// Check if token exists in file
	_, err := getFromConfigFile()
	if err == nil {
		return BackendFile
	}

	// Default to secret-tool if available, otherwise file
	if isSecretToolAvailable() {
		return BackendSecretTool
	}
	return BackendFile
}

// Linux secret-tool implementation

func getFromSecretTool() (*oauth2.Token, error) {
	cmd := exec.Command("secret-tool", "lookup",
		"service", serviceName,
		"account", tokenKey)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to read from secret-tool: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(output))), &token); err != nil {
		return nil, fmt.Errorf("failed to parse token from secret-tool: %w", err)
	}

	return &token, nil
}

func setInSecretTool(token *oauth2.Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to serialize token: %w", err)
	}

	// Delete existing entry (ignore error if not exists)
	_ = deleteFromSecretTool()

	cmd := exec.Command("secret-tool", "store",
		"--label", "google-readonly OAuth Token",
		"service", serviceName,
		"account", tokenKey)
	cmd.Stdin = strings.NewReader(string(data))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to store in secret-tool: %w", err)
	}

	return nil
}

func deleteFromSecretTool() error {
	cmd := exec.Command("secret-tool", "clear",
		"service", serviceName,
		"account", tokenKey)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete from secret-tool: %w", err)
	}

	return nil
}
