package gmail

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	gmailapi "google.golang.org/api/gmail/v1"
)

func TestGetConfigDir(t *testing.T) {
	t.Run("uses XDG_CONFIG_HOME if set", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		dir, err := getConfigDir()
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(tmpDir, "google-readonly"), dir)

		// Verify directory was created
		info, err := os.Stat(dir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("uses ~/.config if XDG_CONFIG_HOME not set", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")

		dir, err := getConfigDir()
		require.NoError(t, err)

		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".config", "google-readonly")
		assert.Equal(t, expected, dir)
	})

	t.Run("creates directory with correct permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		dir, err := getConfigDir()
		require.NoError(t, err)

		info, err := os.Stat(dir)
		require.NoError(t, err)
		// Check directory permissions (0700)
		assert.Equal(t, os.FileMode(0700), info.Mode().Perm())
	})
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

func TestSaveToken(t *testing.T) {
	t.Run("saves token with correct permissions", func(t *testing.T) {
		tmpDir := t.TempDir()
		tokenPath := filepath.Join(tmpDir, "token.json")

		token := &oauth2.Token{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			TokenType:    "Bearer",
		}

		// Actually call saveToken
		err := saveToken(tokenPath, token)
		require.NoError(t, err)

		// Verify file permissions
		info, err := os.Stat(tokenPath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

		// Verify content can be read back
		readToken, err := tokenFromFile(tokenPath)
		require.NoError(t, err)
		assert.Equal(t, "test-access-token", readToken.AccessToken)
		assert.Equal(t, "test-refresh-token", readToken.RefreshToken)
		assert.Equal(t, "Bearer", readToken.TokenType)
	})

	t.Run("creates file if it does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		tokenPath := filepath.Join(tmpDir, "subdir", "token.json")

		// Create parent directory (saveToken doesn't create parents)
		err := os.MkdirAll(filepath.Dir(tokenPath), 0755)
		require.NoError(t, err)

		token := &oauth2.Token{AccessToken: "new-token"}

		err = saveToken(tokenPath, token)
		require.NoError(t, err)

		// Verify file was created
		_, err = os.Stat(tokenPath)
		assert.NoError(t, err)
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tokenPath := filepath.Join(tmpDir, "token.json")

		// Save initial token
		err := saveToken(tokenPath, &oauth2.Token{AccessToken: "old-token"})
		require.NoError(t, err)

		// Save new token
		err = saveToken(tokenPath, &oauth2.Token{AccessToken: "new-token"})
		require.NoError(t, err)

		// Verify new content
		readToken, err := tokenFromFile(tokenPath)
		require.NoError(t, err)
		assert.Equal(t, "new-token", readToken.AccessToken)
	})
}

func TestClientConstants(t *testing.T) {
	assert.Equal(t, "google-readonly", configDirName)
	assert.Equal(t, "credentials.json", credentialsFile)
	assert.Equal(t, "token.json", tokenFile)
}

func TestGetLabelName(t *testing.T) {
	t.Run("returns name for cached label", func(t *testing.T) {
		client := &Client{
			labels: map[string]*gmailapi.Label{
				"Label_123": {Id: "Label_123", Name: "Work"},
				"Label_456": {Id: "Label_456", Name: "Personal"},
			},
			labelsLoaded: true,
		}

		assert.Equal(t, "Work", client.GetLabelName("Label_123"))
		assert.Equal(t, "Personal", client.GetLabelName("Label_456"))
	})

	t.Run("returns ID for uncached label", func(t *testing.T) {
		client := &Client{
			labels:       map[string]*gmailapi.Label{},
			labelsLoaded: true,
		}

		assert.Equal(t, "Unknown_Label", client.GetLabelName("Unknown_Label"))
	})

	t.Run("returns ID when labels not loaded", func(t *testing.T) {
		client := &Client{
			labels:       nil,
			labelsLoaded: false,
		}

		assert.Equal(t, "Label_123", client.GetLabelName("Label_123"))
	})
}

func TestGetLabels(t *testing.T) {
	t.Run("returns nil when labels not loaded", func(t *testing.T) {
		client := &Client{
			labels:       nil,
			labelsLoaded: false,
		}

		result := client.GetLabels()
		assert.Nil(t, result)
	})

	t.Run("returns all cached labels", func(t *testing.T) {
		label1 := &gmailapi.Label{Id: "Label_1", Name: "Work"}
		label2 := &gmailapi.Label{Id: "Label_2", Name: "Personal"}

		client := &Client{
			labels: map[string]*gmailapi.Label{
				"Label_1": label1,
				"Label_2": label2,
			},
			labelsLoaded: true,
		}

		result := client.GetLabels()
		assert.Len(t, result, 2)
		assert.Contains(t, result, label1)
		assert.Contains(t, result, label2)
	})

	t.Run("returns empty slice for empty cache", func(t *testing.T) {
		client := &Client{
			labels:       map[string]*gmailapi.Label{},
			labelsLoaded: true,
		}

		result := client.GetLabels()
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})
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
