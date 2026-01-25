package keychain

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)

	// Retrieve token
	retrieved, err := getFromConfigFile()
	require.NoError(t, err)

	assert.Equal(t, token.AccessToken, retrieved.AccessToken)
	assert.Equal(t, token.RefreshToken, retrieved.RefreshToken)
	assert.Equal(t, token.TokenType, retrieved.TokenType)
	// Compare times with tolerance for JSON marshaling
	assert.WithinDuration(t, token.Expiry, retrieved.Expiry, time.Second)
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
	require.NoError(t, err)

	// Check file permissions
	path := filepath.Join(tmpDir, serviceName, config.TokenFile)
	info, err := os.Stat(path)
	require.NoError(t, err)

	// Verify 0600 permissions (read/write for owner only)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
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
	require.NoError(t, err)

	// Check directory permissions
	dir := filepath.Join(tmpDir, serviceName)
	info, err := os.Stat(dir)
	require.NoError(t, err)

	// Verify 0700 permissions (read/write/execute for owner only)
	assert.Equal(t, os.FileMode(0700), info.Mode().Perm())
}

func TestConfigFile_NotFound(t *testing.T) {
	// Create temp directory (empty)
	tmpDir := t.TempDir()

	// Override config directory for test
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalXDG)

	_, err := getFromConfigFile()
	assert.ErrorIs(t, err, ErrTokenNotFound)
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
	require.NoError(t, err)

	// Write invalid JSON
	path := filepath.Join(configDir, config.TokenFile)
	err = os.WriteFile(path, []byte("invalid json"), 0600)
	require.NoError(t, err)

	_, err = getFromConfigFile()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse token file")
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
	require.NoError(t, err)

	// Store second token
	token2 := &oauth2.Token{
		AccessToken: "second-token",
		TokenType:   "Bearer",
	}
	err = setInConfigFile(token2)
	require.NoError(t, err)

	// Retrieve should return second token
	retrieved, err := getFromConfigFile()
	require.NoError(t, err)
	assert.Equal(t, "second-token", retrieved.AccessToken)
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
	require.NoError(t, err)

	// Delete token
	err = deleteFromConfigFile()
	require.NoError(t, err)

	// Should be gone
	_, err = getFromConfigFile()
	assert.ErrorIs(t, err, ErrTokenNotFound)
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
	assert.NoError(t, err)
}

func TestMigrateFromFile_NoFile(t *testing.T) {
	// Migration should succeed (no-op) when file doesn't exist
	err := MigrateFromFile("/nonexistent/path/token.json")
	assert.NoError(t, err)
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
	require.NoError(t, err)

	// If secure storage has a token (e.g., from real keychain), migration is skipped
	// In that case, we test the direct file parsing instead
	if IsSecureStorage() && HasStoredToken() {
		t.Skip("Secure storage already has a token, migration would be skipped")
	}

	err = MigrateFromFile(tokenPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse token file")
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
	require.NoError(t, err)
	err = os.WriteFile(tokenPath, data, 0600)
	require.NoError(t, err)

	// Run migration
	err = MigrateFromFile(tokenPath)
	require.NoError(t, err)

	// Verify token was stored (uses GetToken to check all backends)
	retrieved, err := GetToken()
	require.NoError(t, err)
	assert.Equal(t, "migrated-token", retrieved.AccessToken)

	// Clean up: delete the token we just stored
	defer DeleteToken()

	// Verify backup was created
	backupPath := tokenPath + ".backup"
	_, err = os.Stat(backupPath)
	assert.NoError(t, err)

	// Verify original was removed
	_, err = os.Stat(tokenPath)
	assert.True(t, os.IsNotExist(err))
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
	assert.ErrorIs(t, err, ErrTokenNotFound)

	// Store a token in config file
	token := &oauth2.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}
	err = setInConfigFile(token)
	require.NoError(t, err)

	// Should successfully retrieve from config file
	retrieved, err := getFromConfigFile()
	require.NoError(t, err)
	assert.Equal(t, "test-token", retrieved.AccessToken)
}

func TestGetStorageBackend(t *testing.T) {
	// Just verify it returns a valid backend
	backend := GetStorageBackend()
	assert.Contains(t, []StorageBackend{BackendKeychain, BackendSecretTool, BackendFile}, backend)
}

func TestIsSecureStorage(t *testing.T) {
	// This will vary by platform - just verify it returns a bool
	result := IsSecureStorage()
	assert.IsType(t, true, result)
}

func TestTokenFilePath(t *testing.T) {
	// Test that tokenFilePath delegates to config package
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	path, err := tokenFilePath()
	require.NoError(t, err)

	configPath, err := config.GetTokenPath()
	require.NoError(t, err)

	assert.Equal(t, configPath, path)
}

func TestServiceNameConstant(t *testing.T) {
	// Verify serviceName matches config.DirName
	assert.Equal(t, config.DirName, serviceName)
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
	require.NoError(t, err)
	assert.Equal(t, "initial-token", token.AccessToken)
	assert.Equal(t, 1, mock.calls)

	// current should remain the same (same pointer)
	assert.Same(t, initialToken, pts.current)
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
	require.NoError(t, err)
	assert.Equal(t, "refreshed-token", token.AccessToken)
	assert.Equal(t, 1, mock.calls)

	// Verify current was updated to the refreshed token
	assert.Equal(t, "refreshed-token", pts.current.AccessToken)
	assert.Equal(t, "new-refresh-token", pts.current.RefreshToken)
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
	require.NoError(t, err)
	assert.Equal(t, "new-token", token.AccessToken)

	// Verify current was set
	require.NotNil(t, pts.current)
	assert.Equal(t, "new-token", pts.current.AccessToken)
}

func TestPersistentTokenSource_BaseError(t *testing.T) {
	// Mock returns an error
	mock := &mockTokenSource{
		token: nil,
		err:   assert.AnError,
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
	assert.Error(t, err)
	assert.Nil(t, token)
	assert.Equal(t, 1, mock.calls)

	// current should remain unchanged on error
	assert.Equal(t, "initial-token", pts.current.AccessToken)
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
		require.NoError(t, err)
		assert.Equal(t, "stable-token", token.AccessToken)
	}

	// Verify mock was called 3 times
	assert.Equal(t, 3, mock.calls)

	// current should still be the same
	assert.Same(t, stableToken, pts.current)
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
	require.NoError(t, err)
	assert.Equal(t, "token-1", pts.current.AccessToken)
	originalCurrent := pts.current

	// Second call: token1 -> token2 (change detected)
	mock.token = token2
	_, err = pts.Token()
	require.NoError(t, err)
	assert.Equal(t, "token-2", pts.current.AccessToken)
	assert.NotSame(t, originalCurrent, pts.current) // current was updated

	// Third call: token2 -> token3 (same AccessToken, no change)
	secondCurrent := pts.current
	mock.token = token3
	_, err = pts.Token()
	require.NoError(t, err)
	// current should not have changed since AccessToken is the same
	assert.Same(t, secondCurrent, pts.current)
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
	require.NoError(t, err)

	// Should return the token from base, not current
	assert.Equal(t, "from-base", token.AccessToken)
}
