package auth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfigDir(t *testing.T) {
	t.Run("uses XDG_CONFIG_HOME if set", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		dir, err := GetConfigDir()
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(tmpDir, ConfigDirName), dir)

		// Verify directory was created
		info, err := os.Stat(dir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("uses ~/.config if XDG_CONFIG_HOME not set", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")

		dir, err := GetConfigDir()
		require.NoError(t, err)

		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".config", ConfigDirName)
		assert.Equal(t, expected, dir)
	})

	t.Run("creates directory with correct permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		dir, err := GetConfigDir()
		require.NoError(t, err)

		info, err := os.Stat(dir)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0700), info.Mode().Perm())
	})
}

func TestGetCredentialsPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	path, err := GetCredentialsPath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmpDir, ConfigDirName, CredentialsFile), path)
}

func TestGetTokenPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	path, err := GetTokenPath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmpDir, ConfigDirName, TokenFile), path)
}

func TestShortenPath(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

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
			result := ShortenPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "google-readonly", ConfigDirName)
	assert.Equal(t, "credentials.json", CredentialsFile)
	assert.Equal(t, "token.json", TokenFile)
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
