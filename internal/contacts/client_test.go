package contacts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetConfigDir(t *testing.T) {
	t.Run("returns valid path", func(t *testing.T) {
		dir, err := getConfigDir()
		assert.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.Contains(t, dir, "google-readonly")
	})
}

func TestConstants(t *testing.T) {
	t.Run("config dir name is correct", func(t *testing.T) {
		assert.Equal(t, "google-readonly", configDirName)
	})

	t.Run("credentials file name is correct", func(t *testing.T) {
		assert.Equal(t, "credentials.json", credentialsFile)
	})
}
