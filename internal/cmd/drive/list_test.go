package drive

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListCommand(t *testing.T) {
	cmd := newListCommand()

	t.Run("has correct use", func(t *testing.T) {
		assert.Equal(t, "list [folder-id]", cmd.Use)
	})

	t.Run("accepts zero or one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		assert.NoError(t, err)

		err = cmd.Args(cmd, []string{"folder-id"})
		assert.NoError(t, err)

		err = cmd.Args(cmd, []string{"folder-id", "extra"})
		assert.Error(t, err)
	})

	t.Run("has max flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("max")
		assert.NotNil(t, flag)
		assert.Equal(t, "m", flag.Shorthand)
		assert.Equal(t, "25", flag.DefValue)
	})

	t.Run("has type flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("type")
		assert.NotNil(t, flag)
		assert.Equal(t, "t", flag.Shorthand)
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		assert.NotNil(t, flag)
		assert.Equal(t, "j", flag.Shorthand)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("has short description", func(t *testing.T) {
		assert.Contains(t, cmd.Short, "List")
	})
}

func TestBuildListQuery(t *testing.T) {
	t.Run("builds query for root folder", func(t *testing.T) {
		query, err := buildListQuery("", "")
		assert.NoError(t, err)
		assert.Contains(t, query, "trashed = false")
		assert.Contains(t, query, "'root' in parents")
	})

	t.Run("builds query for specific folder", func(t *testing.T) {
		query, err := buildListQuery("folder123", "")
		assert.NoError(t, err)
		assert.Contains(t, query, "'folder123' in parents")
		assert.NotContains(t, query, "'root' in parents")
	})

	t.Run("adds type filter for document", func(t *testing.T) {
		query, err := buildListQuery("", "document")
		assert.NoError(t, err)
		assert.Contains(t, query, "mimeType = 'application/vnd.google-apps.document'")
	})

	t.Run("adds type filter for spreadsheet", func(t *testing.T) {
		query, err := buildListQuery("", "spreadsheet")
		assert.NoError(t, err)
		assert.Contains(t, query, "mimeType = 'application/vnd.google-apps.spreadsheet'")
	})

	t.Run("adds type filter for presentation", func(t *testing.T) {
		query, err := buildListQuery("", "presentation")
		assert.NoError(t, err)
		assert.Contains(t, query, "mimeType = 'application/vnd.google-apps.presentation'")
	})

	t.Run("adds type filter for folder", func(t *testing.T) {
		query, err := buildListQuery("", "folder")
		assert.NoError(t, err)
		assert.Contains(t, query, "mimeType = 'application/vnd.google-apps.folder'")
	})

	t.Run("adds type filter for pdf", func(t *testing.T) {
		query, err := buildListQuery("", "pdf")
		assert.NoError(t, err)
		assert.Contains(t, query, "mimeType = 'application/pdf'")
	})

	t.Run("adds type filter for image", func(t *testing.T) {
		query, err := buildListQuery("", "image")
		assert.NoError(t, err)
		assert.Contains(t, query, "mimeType contains 'image/'")
	})

	t.Run("adds type filter for video", func(t *testing.T) {
		query, err := buildListQuery("", "video")
		assert.NoError(t, err)
		assert.Contains(t, query, "mimeType contains 'video/'")
	})

	t.Run("adds type filter for audio", func(t *testing.T) {
		query, err := buildListQuery("", "audio")
		assert.NoError(t, err)
		assert.Contains(t, query, "mimeType contains 'audio/'")
	})

	t.Run("returns error for unknown type", func(t *testing.T) {
		_, err := buildListQuery("", "unknown")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown file type")
	})

	t.Run("accepts type aliases", func(t *testing.T) {
		query, err := buildListQuery("", "doc")
		assert.NoError(t, err)
		assert.Contains(t, query, "application/vnd.google-apps.document")

		query, err = buildListQuery("", "sheet")
		assert.NoError(t, err)
		assert.Contains(t, query, "application/vnd.google-apps.spreadsheet")

		query, err = buildListQuery("", "slides")
		assert.NoError(t, err)
		assert.Contains(t, query, "application/vnd.google-apps.presentation")
	})

	t.Run("is case insensitive for type", func(t *testing.T) {
		query, err := buildListQuery("", "DOCUMENT")
		assert.NoError(t, err)
		assert.Contains(t, query, "application/vnd.google-apps.document")
	})
}

func TestGetMimeTypeFilter(t *testing.T) {
	tests := []struct {
		fileType string
		expected string
		hasError bool
	}{
		{"document", "mimeType = 'application/vnd.google-apps.document'", false},
		{"doc", "mimeType = 'application/vnd.google-apps.document'", false},
		{"spreadsheet", "mimeType = 'application/vnd.google-apps.spreadsheet'", false},
		{"sheet", "mimeType = 'application/vnd.google-apps.spreadsheet'", false},
		{"presentation", "mimeType = 'application/vnd.google-apps.presentation'", false},
		{"slides", "mimeType = 'application/vnd.google-apps.presentation'", false},
		{"folder", "mimeType = 'application/vnd.google-apps.folder'", false},
		{"pdf", "mimeType = 'application/pdf'", false},
		{"image", "mimeType contains 'image/'", false},
		{"video", "mimeType contains 'video/'", false},
		{"audio", "mimeType contains 'audio/'", false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.fileType, func(t *testing.T) {
			result, err := getMimeTypeFilter(tt.fileType)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
		{1610612736, "1.5 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatSize(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}
