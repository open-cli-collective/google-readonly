package auth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-cli-collective/google-readonly/internal/config"
)

// TestDeprecatedWrappers verifies that auth package wrappers delegate to config package
func TestDeprecatedWrappers(t *testing.T) {
	t.Run("GetConfigDir delegates to config package", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		authDir, err := GetConfigDir()
		require.NoError(t, err)

		configDir, err := config.GetConfigDir()
		require.NoError(t, err)

		assert.Equal(t, configDir, authDir)
	})

	t.Run("GetCredentialsPath delegates to config package", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		authPath, err := GetCredentialsPath()
		require.NoError(t, err)

		configPath, err := config.GetCredentialsPath()
		require.NoError(t, err)

		assert.Equal(t, configPath, authPath)
	})

	t.Run("GetTokenPath delegates to config package", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		authPath, err := GetTokenPath()
		require.NoError(t, err)

		configPath, err := config.GetTokenPath()
		require.NoError(t, err)

		assert.Equal(t, configPath, authPath)
	})

	t.Run("ShortenPath delegates to config package", func(t *testing.T) {
		home, err := os.UserHomeDir()
		require.NoError(t, err)

		testPath := filepath.Join(home, ".config", "test")

		authResult := ShortenPath(testPath)
		configResult := config.ShortenPath(testPath)

		assert.Equal(t, configResult, authResult)
	})

	t.Run("Constants match config package", func(t *testing.T) {
		assert.Equal(t, config.DirName, ConfigDirName)
		assert.Equal(t, config.CredentialsFile, CredentialsFile)
		assert.Equal(t, config.TokenFile, TokenFile)
	})
}

func TestAllScopes(t *testing.T) {
	assert.Len(t, AllScopes, 3)
	assert.Contains(t, AllScopes, "https://www.googleapis.com/auth/gmail.readonly")
	assert.Contains(t, AllScopes, "https://www.googleapis.com/auth/calendar.readonly")
	assert.Contains(t, AllScopes, "https://www.googleapis.com/auth/contacts.readonly")
}

func TestTokenFromFile(t *testing.T) {
	t.Run("reads valid token file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tokenPath := filepath.Join(tmpDir, "token.json")

		tokenData := `{
			"access_token": "test-access-token",
			"token_type": "Bearer",
			"refresh_token": "test-refresh-token",
			"expiry": "2024-01-01T00:00:00Z"
		}`
		err := os.WriteFile(tokenPath, []byte(tokenData), 0600)
		require.NoError(t, err)

		token, err := tokenFromFile(tokenPath)
		require.NoError(t, err)
		assert.Equal(t, "test-access-token", token.AccessToken)
		assert.Equal(t, "Bearer", token.TokenType)
		assert.Equal(t, "test-refresh-token", token.RefreshToken)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := tokenFromFile("/nonexistent/token.json")
		assert.Error(t, err)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		tokenPath := filepath.Join(tmpDir, "token.json")

		err := os.WriteFile(tokenPath, []byte("not valid json"), 0600)
		require.NoError(t, err)

		_, err = tokenFromFile(tokenPath)
		assert.Error(t, err)
	})
}
