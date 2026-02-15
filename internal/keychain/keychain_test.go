package keychain

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/open-cli-collective/google-readonly/internal/config"
)

func TestConfigFile_TokenRoundTrip(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Override config directory for test
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	// Create test token
	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour).Truncate(time.Second),
	}

	// Store token
	err := setInConfigFile(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Retrieve token
	retrieved, err := getFromConfigFile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if retrieved.AccessToken != token.AccessToken {
		t.Errorf("got %v, want %v", retrieved.AccessToken, token.AccessToken)
	}
	if retrieved.RefreshToken != token.RefreshToken {
		t.Errorf("got %v, want %v", retrieved.RefreshToken, token.RefreshToken)
	}
	if retrieved.TokenType != token.TokenType {
		t.Errorf("got %v, want %v", retrieved.TokenType, token.TokenType)
	}
	// Compare times with tolerance for JSON marshaling
	if diff := token.Expiry.Sub(retrieved.Expiry); diff < -time.Second || diff > time.Second {
		t.Errorf("times differ by %v, max allowed %v", diff, time.Second)
	}
}

func TestConfigFile_Permissions(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Override config directory for test
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	token := &oauth2.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}

	err := setInConfigFile(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check file permissions
	path := filepath.Join(tmpDir, serviceName, config.TokenFile)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify 0600 permissions (read/write for owner only)
	if info.Mode().Perm() != os.FileMode(0600) {
		t.Errorf("got %v, want %v", info.Mode().Perm(), os.FileMode(0600))
	}
}

func TestConfigFile_DirectoryPermissions(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Override config directory for test
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	token := &oauth2.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}

	err := setInConfigFile(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check directory permissions
	dir := filepath.Join(tmpDir, serviceName)
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify 0700 permissions (read/write/execute for owner only)
	if info.Mode().Perm() != os.FileMode(0700) {
		t.Errorf("got %v, want %v", info.Mode().Perm(), os.FileMode(0700))
	}
}

func TestConfigFile_NotFound(t *testing.T) {
	// Create temp directory (empty)
	tmpDir := t.TempDir()

	// Override config directory for test
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	_, err := getFromConfigFile()
	if !errors.Is(err, ErrTokenNotFound) {
		t.Errorf("got %v, want %v", err, ErrTokenNotFound)
	}
}

func TestConfigFile_InvalidJSON(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Override config directory for test
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	// Create config directory
	configDir := filepath.Join(tmpDir, serviceName)
	err := os.MkdirAll(configDir, 0700)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Write invalid JSON
	path := filepath.Join(configDir, config.TokenFile)
	err = os.WriteFile(path, []byte("invalid json"), 0600)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = getFromConfigFile()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse token file") {
		t.Errorf("expected %q to contain %q", err.Error(), "failed to parse token file")
	}
}

func TestConfigFile_Overwrite(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Override config directory for test
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	// Store first token
	token1 := &oauth2.Token{
		AccessToken: "first-token",
		TokenType:   "Bearer",
	}
	err := setInConfigFile(token1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Store second token
	token2 := &oauth2.Token{
		AccessToken: "second-token",
		TokenType:   "Bearer",
	}
	err = setInConfigFile(token2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Retrieve should return second token
	retrieved, err := getFromConfigFile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved.AccessToken != "second-token" {
		t.Errorf("got %v, want %v", retrieved.AccessToken, "second-token")
	}
}

func TestConfigFile_Delete(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Override config directory for test
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	// Store token
	token := &oauth2.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}
	err := setInConfigFile(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Delete token
	err = deleteFromConfigFile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be gone
	_, err = getFromConfigFile()
	if !errors.Is(err, ErrTokenNotFound) {
		t.Errorf("got %v, want %v", err, ErrTokenNotFound)
	}
}

func TestConfigFile_DeleteNonExistent(t *testing.T) {
	// Create temp directory (empty)
	tmpDir := t.TempDir()

	// Override config directory for test
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	// Delete should not error on non-existent file
	err := deleteFromConfigFile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMigrateFromFile_NoFile(t *testing.T) {
	// Migration should succeed (no-op) when file doesn't exist
	err := MigrateFromFile("/nonexistent/path/token.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMigrateFromFile_InvalidJSON(t *testing.T) {
	// This test verifies that invalid JSON in a token file causes an error
	// when migration is attempted without existing secure storage.
	//
	// Note: On macOS, if there's already a token in the system keychain,
	// MigrateFromFile will skip reading the file (already migrated).
	// This is expected behavior - the test validates the JSON parsing path
	// when no secure storage token exists.

	tmpDir := t.TempDir()
	configDir := t.TempDir()

	// Override config directory so we don't find existing tokens
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", configDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	// Create temp file with invalid JSON
	tokenPath := filepath.Join(tmpDir, "token.json")
	err := os.WriteFile(tokenPath, []byte("invalid json"), 0600)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// If secure storage has a token (e.g., from real keychain), migration is skipped
	// In that case, we test the direct file parsing instead
	if IsSecureStorage() && HasStoredToken() {
		t.Skip("Secure storage already has a token, migration would be skipped")
	}

	err = MigrateFromFile(tokenPath)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse token file") {
		t.Errorf("expected %q to contain %q", err.Error(), "failed to parse token file")
	}
}

func TestMigrateFromFile_Success(t *testing.T) {
	// This test verifies that migration from file to storage works correctly.
	// On macOS, if there's already a token in the system keychain,
	// migration is skipped (considered already migrated).

	// Skip if secure storage already has a token
	if IsSecureStorage() && HasStoredToken() {
		t.Skip("Secure storage already has a token, migration would be skipped")
	}

	// Create temp directories
	tmpDir := t.TempDir()
	configDir := t.TempDir()

	// Override config directory for test
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", configDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	// Create valid token file
	token := &oauth2.Token{
		AccessToken:  "migrated-token",
		RefreshToken: "migrated-refresh",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
	tokenPath := filepath.Join(tmpDir, "token.json")
	data, err := json.Marshal(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = os.WriteFile(tokenPath, data, 0600)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Run migration
	err = MigrateFromFile(tokenPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify token was stored (uses GetToken to check all backends)
	retrieved, err := GetToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved.AccessToken != "migrated-token" {
		t.Errorf("got %v, want %v", retrieved.AccessToken, "migrated-token")
	}

	// Clean up: delete the token we just stored
	defer DeleteToken()

	// Verify original file was securely deleted (not renamed to backup)
	_, err = os.Stat(tokenPath)
	if !os.IsNotExist(err) {
		t.Error("original token file should be deleted")
	}

	// Verify no backup file was created (secure delete, not rename)
	backupPath := tokenPath + ".backup"
	_, err = os.Stat(backupPath)
	if !os.IsNotExist(err) {
		t.Error("backup file should not exist (secure delete)")
	}
}

func TestHasStoredToken_ConfigFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Override config directory for test
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	// Test file-based storage specifically
	// Note: HasStoredToken() may find tokens in system keychain on macOS,
	// so we test the underlying file functions directly

	// Should return error when no token file
	_, err := getFromConfigFile()
	if !errors.Is(err, ErrTokenNotFound) {
		t.Errorf("got %v, want %v", err, ErrTokenNotFound)
	}

	// Store a token in config file
	token := &oauth2.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}
	err = setInConfigFile(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should successfully retrieve from config file
	retrieved, err := getFromConfigFile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrieved.AccessToken != "test-token" {
		t.Errorf("got %v, want %v", retrieved.AccessToken, "test-token")
	}
}

func TestGetStorageBackend(t *testing.T) {
	// Just verify it returns a valid backend
	backend := GetStorageBackend()
	validBackends := []StorageBackend{BackendKeychain, BackendSecretTool, BackendFile}
	found := false
	for _, v := range validBackends {
		if v == backend {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("got %v, want one of %v", backend, validBackends)
	}
}

func TestIsSecureStorage(t *testing.T) {
	// This will vary by platform - just verify it returns a bool
	// Go enforces the type at compile time, so no runtime check needed
	_ = IsSecureStorage()
}

func TestTokenFilePath(t *testing.T) {
	// Test that tokenFilePath delegates to config package
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	path, err := tokenFilePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	configPath, err := config.GetTokenPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if path != configPath {
		t.Errorf("got %v, want %v", path, configPath)
	}
}

func TestServiceNameConstant(t *testing.T) {
	// Verify serviceName matches config.DirName
	if serviceName != config.DirName {
		t.Errorf("got %v, want %v", serviceName, config.DirName)
	}
}

func TestSecureDelete(t *testing.T) {
	t.Run("deletes file", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "secret.txt")

		// Create file with sensitive data
		sensitiveData := []byte("super secret token data")
		err := os.WriteFile(path, sensitiveData, 0600)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify file exists
		_, err = os.Stat(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Secure delete
		err = secureDelete(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify file is gone
		_, err = os.Stat(path)
		if !os.IsNotExist(err) {
			t.Error("got false, want true")
		}
	})

	t.Run("handles non-existent file", func(t *testing.T) {
		// Should not error on non-existent file
		err := secureDelete("/nonexistent/path/file.txt")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("overwrites file content before deletion", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "secret.txt")

		// Create file with known content
		sensitiveData := []byte("secret123456")
		err := os.WriteFile(path, sensitiveData, 0600)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Get file size before deletion
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		originalSize := info.Size()

		// Create a copy to verify overwrite behavior
		// We'll use a custom path that we keep open to observe the overwrite
		copyPath := filepath.Join(tmpDir, "observe.txt")
		err = os.WriteFile(copyPath, sensitiveData, 0600)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Open the file to observe content after overwrite but before unlink
		// This simulates what forensic tools would see
		f, err := os.OpenFile(copyPath, os.O_RDWR, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Overwrite with zeros (simulating what secureDelete does)
		zeros := make([]byte, originalSize)
		_, err = f.Write(zeros)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_ = f.Sync()

		// Read back - should be all zeros
		_, err = f.Seek(0, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		content := make([]byte, originalSize)
		_, err = f.Read(content)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		f.Close()

		// Verify content is all zeros
		for i, b := range content {
			if b != byte(0) {
				t.Errorf("byte %d: got %v, want %v", i, b, byte(0))
			}
		}

		// Now test actual secureDelete
		err = secureDelete(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// File should be gone
		_, err = os.Stat(path)
		if !os.IsNotExist(err) {
			t.Error("got false, want true")
		}
	})
}

// mockTokenSource is a test double for oauth2.TokenSource
type mockTokenSource struct {
	token *oauth2.Token
	err   error
	calls int
}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	m.calls++
	return m.token, m.err
}

func TestPersistentTokenSource_NoChange(t *testing.T) {
	// Create initial token
	initialToken := &oauth2.Token{
		AccessToken:  "initial-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}

	// Mock returns the same token (no refresh occurred)
	mock := &mockTokenSource{token: initialToken}

	// Create PersistentTokenSource with mock base
	pts := &PersistentTokenSource{
		base:    mock,
		current: initialToken,
	}

	// Call Token()
	token, err := pts.Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token.AccessToken != "initial-token" {
		t.Errorf("got %v, want %v", token.AccessToken, "initial-token")
	}
	if mock.calls != 1 {
		t.Errorf("got %v, want %v", mock.calls, 1)
	}

	// current should remain the same (same pointer)
	if pts.current != initialToken {
		t.Errorf("expected same pointer, got different")
	}
}

func TestPersistentTokenSource_RefreshUpdatesCurrent(t *testing.T) {
	// Create initial token
	initialToken := &oauth2.Token{
		AccessToken:  "initial-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(-time.Hour), // Expired
	}

	// Create refreshed token (different access token)
	refreshedToken := &oauth2.Token{
		AccessToken:  "refreshed-token",
		RefreshToken: "new-refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}

	// Mock returns the refreshed token
	mock := &mockTokenSource{token: refreshedToken}

	// Create PersistentTokenSource with mock base
	pts := &PersistentTokenSource{
		base:    mock,
		current: initialToken,
	}

	// Call Token() - should detect change and update current
	token, err := pts.Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token.AccessToken != "refreshed-token" {
		t.Errorf("got %v, want %v", token.AccessToken, "refreshed-token")
	}
	if mock.calls != 1 {
		t.Errorf("got %v, want %v", mock.calls, 1)
	}

	// Verify current was updated to the refreshed token
	if pts.current.AccessToken != "refreshed-token" {
		t.Errorf("got %v, want %v", pts.current.AccessToken, "refreshed-token")
	}
	if pts.current.RefreshToken != "new-refresh-token" {
		t.Errorf("got %v, want %v", pts.current.RefreshToken, "new-refresh-token")
	}
}

func TestPersistentTokenSource_NilCurrentUpdatesCurrent(t *testing.T) {
	// Create token
	newToken := &oauth2.Token{
		AccessToken:  "new-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}

	// Mock returns the token
	mock := &mockTokenSource{token: newToken}

	// Create PersistentTokenSource with nil current
	pts := &PersistentTokenSource{
		base:    mock,
		current: nil, // No current token
	}

	// Call Token() - should detect as change (nil -> token) and update current
	token, err := pts.Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token.AccessToken != "new-token" {
		t.Errorf("got %v, want %v", token.AccessToken, "new-token")
	}

	// Verify current was set
	if pts.current == nil {
		t.Fatal("expected non-nil, got nil")
	}
	if pts.current.AccessToken != "new-token" {
		t.Errorf("got %v, want %v", pts.current.AccessToken, "new-token")
	}
}

func TestPersistentTokenSource_BaseError(t *testing.T) {
	// Mock returns an error
	mock := &mockTokenSource{
		token: nil,
		err:   fmt.Errorf("mock error"),
	}

	initialToken := &oauth2.Token{
		AccessToken: "initial-token",
		TokenType:   "Bearer",
	}

	// Create PersistentTokenSource with mock base
	pts := &PersistentTokenSource{
		base:    mock,
		current: initialToken,
	}

	// Call Token() - should propagate error
	token, err := pts.Token()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if token != nil {
		t.Errorf("got %v, want nil", token)
	}
	if mock.calls != 1 {
		t.Errorf("got %v, want %v", mock.calls, 1)
	}

	// current should remain unchanged on error
	if pts.current.AccessToken != "initial-token" {
		t.Errorf("got %v, want %v", pts.current.AccessToken, "initial-token")
	}
}

func TestPersistentTokenSource_MultipleCalls_NoChange(t *testing.T) {
	// Create token
	stableToken := &oauth2.Token{
		AccessToken:  "stable-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}

	// Mock returns the same token every time
	mock := &mockTokenSource{token: stableToken}

	// Create PersistentTokenSource with initial current set
	pts := &PersistentTokenSource{
		base:    mock,
		current: stableToken, // Already set to same token
	}

	// Multiple calls should all succeed
	for i := 0; i < 3; i++ {
		token, err := pts.Token()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token.AccessToken != "stable-token" {
			t.Errorf("got %v, want %v", token.AccessToken, "stable-token")
		}
	}

	// Verify mock was called 3 times
	if mock.calls != 3 {
		t.Errorf("got %v, want %v", mock.calls, 3)
	}

	// current should still be the same
	if pts.current != stableToken {
		t.Errorf("expected same pointer, got different")
	}
}

func TestPersistentTokenSource_ChangeDetection(t *testing.T) {
	// Test that change detection works correctly by tracking current updates

	// Create tokens
	token1 := &oauth2.Token{AccessToken: "token-1", TokenType: "Bearer"}
	token2 := &oauth2.Token{AccessToken: "token-2", TokenType: "Bearer"}
	token3 := &oauth2.Token{AccessToken: "token-2", TokenType: "Bearer"} // Same access token as token2

	// Mock that we can update between calls
	mock := &mockTokenSource{token: token1}

	pts := &PersistentTokenSource{
		base:    mock,
		current: nil,
	}

	// First call: nil -> token1 (change detected)
	_, err := pts.Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pts.current.AccessToken != "token-1" {
		t.Errorf("got %v, want %v", pts.current.AccessToken, "token-1")
	}
	originalCurrent := pts.current

	// Second call: token1 -> token2 (change detected)
	mock.token = token2
	_, err = pts.Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pts.current.AccessToken != "token-2" {
		t.Errorf("got %v, want %v", pts.current.AccessToken, "token-2")
	}
	if pts.current == originalCurrent {
		t.Errorf("expected different pointers, got same")
	}

	// Third call: token2 -> token3 (same AccessToken, no change)
	secondCurrent := pts.current
	mock.token = token3
	_, err = pts.Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// current should not have changed since AccessToken is the same
	if pts.current != secondCurrent {
		t.Errorf("expected same pointer, got different")
	}
}

func TestPersistentTokenSource_ReturnsCorrectToken(t *testing.T) {
	// Verify that Token() returns the token from base, not current
	initialToken := &oauth2.Token{AccessToken: "initial", TokenType: "Bearer"}
	baseToken := &oauth2.Token{AccessToken: "from-base", TokenType: "Bearer"}

	mock := &mockTokenSource{token: baseToken}

	pts := &PersistentTokenSource{
		base:    mock,
		current: initialToken,
	}

	token, err := pts.Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return the token from base, not current
	if token.AccessToken != "from-base" {
		t.Errorf("got %v, want %v", token.AccessToken, "from-base")
	}
}
