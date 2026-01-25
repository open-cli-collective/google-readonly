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

func TestGetExportMimeType(t *testing.T) {
	t.Run("returns correct MIME type for Document exports", func(t *testing.T) {
		tests := []struct {
			format   string
			expected string
		}{
			{"pdf", "application/pdf"},
			{"docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
			{"txt", "text/plain"},
			{"html", "text/html"},
			{"md", "text/markdown"},
		}

		for _, tt := range tests {
			t.Run(tt.format, func(t *testing.T) {
				result, err := GetExportMimeType(MimeTypeDocument, tt.format)
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("returns correct MIME type for Spreadsheet exports", func(t *testing.T) {
		tests := []struct {
			format   string
			expected string
		}{
			{"pdf", "application/pdf"},
			{"xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
			{"csv", "text/csv"},
		}

		for _, tt := range tests {
			t.Run(tt.format, func(t *testing.T) {
				result, err := GetExportMimeType(MimeTypeSpreadsheet, tt.format)
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("returns correct MIME type for Presentation exports", func(t *testing.T) {
		result, err := GetExportMimeType(MimeTypePresentation, "pptx")
		assert.NoError(t, err)
		assert.Equal(t, "application/vnd.openxmlformats-officedocument.presentationml.presentation", result)
	})

	t.Run("returns correct MIME type for Drawing exports", func(t *testing.T) {
		result, err := GetExportMimeType(MimeTypeDrawing, "png")
		assert.NoError(t, err)
		assert.Equal(t, "image/png", result)
	})

	t.Run("returns error for unsupported format", func(t *testing.T) {
		_, err := GetExportMimeType(MimeTypeDocument, "xyz")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")
	})

	t.Run("returns error for non-exportable file type", func(t *testing.T) {
		_, err := GetExportMimeType("application/pdf", "docx")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not support export")
	})

	t.Run("returns error for format not matching file type", func(t *testing.T) {
		// csv is valid for spreadsheets but not documents
		_, err := GetExportMimeType(MimeTypeDocument, "csv")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not supported for Google Document")
	})
}

func TestGetSupportedExportFormats(t *testing.T) {
	t.Run("returns formats for Document", func(t *testing.T) {
		formats := GetSupportedExportFormats(MimeTypeDocument)
		assert.Contains(t, formats, "pdf")
		assert.Contains(t, formats, "docx")
		assert.Contains(t, formats, "txt")
	})

	t.Run("returns formats for Spreadsheet", func(t *testing.T) {
		formats := GetSupportedExportFormats(MimeTypeSpreadsheet)
		assert.Contains(t, formats, "xlsx")
		assert.Contains(t, formats, "csv")
	})

	t.Run("returns nil for non-exportable file", func(t *testing.T) {
		formats := GetSupportedExportFormats("application/pdf")
		assert.Nil(t, formats)
	})
}

func TestGetFileExtension(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"pdf", ".pdf"},
		{"docx", ".docx"},
		{"xlsx", ".xlsx"},
		{"pptx", ".pptx"},
		{"txt", ".txt"},
		{"html", ".html"},
		{"md", ".md"},
		{"csv", ".csv"},
		{"png", ".png"},
		{"svg", ".svg"},
		{"jpg", ".jpg"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := GetFileExtension(tt.format)
			assert.Equal(t, tt.expected, result)
		})
	}
}
