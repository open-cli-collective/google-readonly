package gmail

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gmailapi "google.golang.org/api/gmail/v1"

	"github.com/open-cli-collective/google-readonly/internal/auth"
)

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

// TestDeprecatedWrappers verifies that the deprecated wrappers delegate correctly to the auth package
func TestDeprecatedWrappers(t *testing.T) {
	t.Run("GetConfigDir delegates to auth package", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		gmailDir, err := GetConfigDir()
		require.NoError(t, err)

		authDir, err := auth.GetConfigDir()
		require.NoError(t, err)

		assert.Equal(t, authDir, gmailDir)
	})

	t.Run("GetCredentialsPath delegates to auth package", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", tmpDir)

		gmailPath, err := GetCredentialsPath()
		require.NoError(t, err)

		authPath, err := auth.GetCredentialsPath()
		require.NoError(t, err)

		assert.Equal(t, authPath, gmailPath)
	})

	t.Run("ShortenPath delegates to auth package", func(t *testing.T) {
		home, err := os.UserHomeDir()
		require.NoError(t, err)

		testPath := filepath.Join(home, ".config", "test")

		gmailResult := ShortenPath(testPath)
		authResult := auth.ShortenPath(testPath)

		assert.Equal(t, authResult, gmailResult)
	})
}
