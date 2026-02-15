package drive

import (
	"testing"

	"github.com/open-cli-collective/google-readonly/internal/testutil"
)

func TestListCommand(t *testing.T) {
	cmd := newListCommand()

	t.Run("has correct use", func(t *testing.T) {
		testutil.Equal(t, cmd.Use, "list [folder-id]")
	})

	t.Run("accepts zero or one argument", func(t *testing.T) {
		err := cmd.Args(cmd, []string{})
		testutil.NoError(t, err)

		err = cmd.Args(cmd, []string{"folder-id"})
		testutil.NoError(t, err)

		err = cmd.Args(cmd, []string{"folder-id", "extra"})
		testutil.Error(t, err)
	})

	t.Run("has max flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("max")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "m")
		testutil.Equal(t, flag.DefValue, "25")
	})

	t.Run("has type flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("type")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "t")
	})

	t.Run("has json flag", func(t *testing.T) {
		flag := cmd.Flags().Lookup("json")
		testutil.NotNil(t, flag)
		testutil.Equal(t, flag.Shorthand, "j")
		testutil.Equal(t, flag.DefValue, "false")
	})

	t.Run("has short description", func(t *testing.T) {
		testutil.Contains(t, cmd.Short, "List")
	})
}

func TestBuildListQuery(t *testing.T) {
	t.Run("builds query for root folder", func(t *testing.T) {
		query, err := buildListQuery("", "")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "trashed = false")
		testutil.Contains(t, query, "'root' in parents")
	})

	t.Run("builds query for specific folder", func(t *testing.T) {
		query, err := buildListQuery("folder123", "")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "'folder123' in parents")
		testutil.NotContains(t, query, "'root' in parents")
	})

	t.Run("adds type filter for document", func(t *testing.T) {
		query, err := buildListQuery("", "document")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "mimeType = 'application/vnd.google-apps.document'")
	})

	t.Run("adds type filter for spreadsheet", func(t *testing.T) {
		query, err := buildListQuery("", "spreadsheet")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "mimeType = 'application/vnd.google-apps.spreadsheet'")
	})

	t.Run("adds type filter for presentation", func(t *testing.T) {
		query, err := buildListQuery("", "presentation")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "mimeType = 'application/vnd.google-apps.presentation'")
	})

	t.Run("adds type filter for folder", func(t *testing.T) {
		query, err := buildListQuery("", "folder")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "mimeType = 'application/vnd.google-apps.folder'")
	})

	t.Run("adds type filter for pdf", func(t *testing.T) {
		query, err := buildListQuery("", "pdf")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "mimeType = 'application/pdf'")
	})

	t.Run("adds type filter for image", func(t *testing.T) {
		query, err := buildListQuery("", "image")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "mimeType contains 'image/'")
	})

	t.Run("adds type filter for video", func(t *testing.T) {
		query, err := buildListQuery("", "video")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "mimeType contains 'video/'")
	})

	t.Run("adds type filter for audio", func(t *testing.T) {
		query, err := buildListQuery("", "audio")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "mimeType contains 'audio/'")
	})

	t.Run("returns error for unknown type", func(t *testing.T) {
		_, err := buildListQuery("", "unknown")
		testutil.Error(t, err)
		testutil.Contains(t, err.Error(), "unknown file type")
	})

	t.Run("accepts type aliases", func(t *testing.T) {
		query, err := buildListQuery("", "doc")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "application/vnd.google-apps.document")

		query, err = buildListQuery("", "sheet")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "application/vnd.google-apps.spreadsheet")

		query, err = buildListQuery("", "slides")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "application/vnd.google-apps.presentation")
	})

	t.Run("is case insensitive for type", func(t *testing.T) {
		query, err := buildListQuery("", "DOCUMENT")
		testutil.NoError(t, err)
		testutil.Contains(t, query, "application/vnd.google-apps.document")
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
				testutil.Error(t, err)
			} else {
				testutil.NoError(t, err)
				testutil.Equal(t, result, tt.expected)
			}
		})
	}
}

// Tests for formatSize moved to internal/format/format_test.go
