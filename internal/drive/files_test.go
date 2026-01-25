package drive

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/drive/v3"
)

func TestParseFile(t *testing.T) {
	t.Run("parses basic file", func(t *testing.T) {
		f := &drive.File{
			Id:           "123",
			Name:         "test.txt",
			MimeType:     "text/plain",
			Size:         1024,
			CreatedTime:  "2024-01-15T10:30:00Z",
			ModifiedTime: "2024-01-16T14:00:00Z",
			Parents:      []string{"parent1"},
			WebViewLink:  "https://drive.google.com/file/d/123",
			Shared:       true,
		}

		result := ParseFile(f)

		assert.Equal(t, "123", result.ID)
		assert.Equal(t, "test.txt", result.Name)
		assert.Equal(t, "text/plain", result.MimeType)
		assert.Equal(t, int64(1024), result.Size)
		assert.Equal(t, 2024, result.CreatedTime.Year())
		assert.Equal(t, 2024, result.ModifiedTime.Year())
		assert.Equal(t, []string{"parent1"}, result.Parents)
		assert.Equal(t, "https://drive.google.com/file/d/123", result.WebViewLink)
		assert.True(t, result.Shared)
	})

	t.Run("parses file with owners", func(t *testing.T) {
		f := &drive.File{
			Id:       "123",
			Name:     "shared.txt",
			MimeType: "text/plain",
			Owners: []*drive.User{
				{EmailAddress: "owner1@example.com"},
				{EmailAddress: "owner2@example.com"},
			},
		}

		result := ParseFile(f)

		assert.Equal(t, []string{"owner1@example.com", "owner2@example.com"}, result.Owners)
	})

	t.Run("handles empty timestamps", func(t *testing.T) {
		f := &drive.File{
			Id:           "123",
			Name:         "no-times.txt",
			MimeType:     "text/plain",
			CreatedTime:  "",
			ModifiedTime: "",
		}

		result := ParseFile(f)

		assert.True(t, result.CreatedTime.IsZero())
		assert.True(t, result.ModifiedTime.IsZero())
	})

	t.Run("handles malformed timestamps", func(t *testing.T) {
		f := &drive.File{
			Id:           "123",
			Name:         "bad-times.txt",
			MimeType:     "text/plain",
			CreatedTime:  "not-a-timestamp",
			ModifiedTime: "also-not-valid",
		}

		result := ParseFile(f)

		assert.True(t, result.CreatedTime.IsZero())
		assert.True(t, result.ModifiedTime.IsZero())
	})

	t.Run("handles nil owners", func(t *testing.T) {
		f := &drive.File{
			Id:       "123",
			Name:     "no-owners.txt",
			MimeType: "text/plain",
			Owners:   nil,
		}

		result := ParseFile(f)

		assert.Nil(t, result.Owners)
	})

	t.Run("handles empty owners slice", func(t *testing.T) {
		f := &drive.File{
			Id:       "123",
			Name:     "empty-owners.txt",
			MimeType: "text/plain",
			Owners:   []*drive.User{},
		}

		result := ParseFile(f)

		assert.Nil(t, result.Owners)
	})
}

func TestGetTypeName(t *testing.T) {
	tests := []struct {
		mimeType string
		expected string
	}{
		// Google Workspace types
		{MimeTypeFolder, "Folder"},
		{MimeTypeDocument, "Document"},
		{MimeTypeSpreadsheet, "Spreadsheet"},
		{MimeTypePresentation, "Presentation"},
		{MimeTypeDrawing, "Drawing"},
		{MimeTypeForm, "Form"},
		{MimeTypeSite, "Site"},
		{MimeTypeShortcut, "Shortcut"},
		// Common file types
		{"application/pdf", "PDF"},
		{"text/plain", "Text"},
		{"text/html", "HTML"},
		{"text/csv", "CSV"},
		{"application/zip", "ZIP"},
		// Prefix-based types
		{"image/png", "Image"},
		{"image/jpeg", "Image"},
		{"video/mp4", "Video"},
		{"video/quicktime", "Video"},
		{"audio/mpeg", "Audio"},
		{"audio/wav", "Audio"},
		// Unknown types
		{"application/octet-stream", "application/octet-stream"},
		{"unknown/type", "unknown/type"},
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			result := GetTypeName(tt.mimeType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsGoogleWorkspaceFile(t *testing.T) {
	t.Run("returns true for Google Workspace files", func(t *testing.T) {
		workspaceTypes := []string{
			MimeTypeDocument,
			MimeTypeSpreadsheet,
			MimeTypePresentation,
			MimeTypeDrawing,
			MimeTypeForm,
			MimeTypeSite,
		}

		for _, mimeType := range workspaceTypes {
			assert.True(t, IsGoogleWorkspaceFile(mimeType), "expected true for %s", mimeType)
		}
	})

	t.Run("returns false for non-Workspace files", func(t *testing.T) {
		nonWorkspaceTypes := []string{
			MimeTypeFolder,
			MimeTypeShortcut,
			"application/pdf",
			"text/plain",
			"image/png",
			"video/mp4",
			"application/octet-stream",
		}

		for _, mimeType := range nonWorkspaceTypes {
			assert.False(t, IsGoogleWorkspaceFile(mimeType), "expected false for %s", mimeType)
		}
	})
}
